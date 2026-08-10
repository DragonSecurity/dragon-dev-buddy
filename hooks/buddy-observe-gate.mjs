#!/usr/bin/env node
/**
 * Nag once if a turn changed code but never told the buddy.
 *
 * Three roles, dispatched on hook_event_name:
 *
 *   UserPromptSubmit  a new prompt starts a new turn, so a mark still standing
 *                     belongs to the last one: drop it.
 *   PostToolUse       an edit tool marks the session dirty; buddy_observe clears it.
 *   Stop              if the session is still dirty, block once and clear the mark.
 *
 * The mark is keyed by session, but what it has to mean is "this turn changed
 * code" -- and with no turn boundary those are not the same claim. An edit can
 * outlive its turn three ways: a background subagent or workflow whose
 * PostToolUse arrives under the parent's session id minutes after the turn ended,
 * a turn interrupted before Stop ever ran, and a session resumed by id with a
 * mark still on disk from its previous run. Each leaves the mark to be consumed
 * by the *next* Stop, whatever that turn happened to do, so a turn that only read
 * files is nagged for code it never touched. Read out of one session's
 * transcript: a workflow agent wrote a file 2m40s after its turn ended, and the
 * turn after that -- a dozen Bash calls, no edits -- was blocked for it.
 *
 * UserPromptSubmit is the missing boundary. It fires before the turn begins, so
 * unlike a transcript scan it races nothing, and it costs one stat per prompt.
 * Work an interrupted turn left unrecorded stays unrecorded, which is the right
 * trade: a nag the user cannot act on is worse than a quiet miss. Background work
 * keeps its nag either way -- an unconsumed mark survives to the turn where the
 * harness reports the agent finished, which is the first turn that can record it.
 *
 * State lives in a marker file rather than in the transcript because the Stop
 * hook races the transcript writer. buddy_observe is by convention the last tool
 * call of a turn, so its entry is routinely still unflushed when Stop reads the
 * file -- measured over the gate's first three days, that turned 9 of 18 blocks
 * into false positives, each one costing the buddy a duplicate observation.
 * PostToolUse fires when the tool actually runs, so there is nothing to race.
 *
 * Fails open and silent: a broken gate must never wedge a session.
 */
import { appendFileSync, existsSync, mkdirSync, readdirSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';

const EDIT_TOOLS = new Set(['Edit', 'Write', 'NotebookEdit', 'MultiEdit']);
const STATE = join(homedir(), '.claude', 'buddy-gate');
const LOG = join(homedir(), '.claude', 'buddy-gate.log');
const STALE_AFTER_MS = 7 * 24 * 60 * 60 * 1000;
// The tool is named without a server prefix on purpose. It is addressed as
// mcp__buddy__buddy_observe when the server is registered by hand and as
// mcp__plugin_dragon-dev-buddy_buddy__buddy_observe when this pack declares it,
// and naming either one sends the model looking for a tool that is not in its
// list on half of all installs. The bare name is the part that never moves.
const REASON =
  'This turn changed code but never recorded it with the buddy. ' +
  'Call the buddy_observe tool with a one-sentence summary of what you did, ' +
  'passing skills_used if you invoked any skills, then relay the reaction to the ' +
  'user verbatim. Its full name carries a server prefix that depends on how the ' +
  'buddy is installed. If it is not in your tool list it is deferred -- run ' +
  'ToolSearch with query "buddy_observe" first.';

function markPath(sessionId) {
  const safe = String(sessionId ?? '').replace(/[^A-Za-z0-9_-]/g, '');
  return join(STATE, (safe || 'unknown') + '.dirty');
}

/** Local time, to match the timestamps already in the log. */
function stamp(d = new Date()) {
  const pad = (n) => String(n).padStart(2, '0');
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}` +
    `T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  );
}

function log(fields) {
  try {
    appendFileSync(LOG, JSON.stringify({ at: stamp(), ...fields }) + '\n');
  } catch {
    /* the log is a convenience, never a reason to fail the hook */
  }
}

/** True when the tool call clearly errored, so an observe should not count. */
function failed(payload) {
  const response = payload.tool_response;
  return Boolean(response && typeof response === 'object' && (response.is_error || response.isError));
}

/** Drop marks from sessions that ended without ever stopping cleanly. */
function prune() {
  try {
    const cutoff = Date.now() - STALE_AFTER_MS;
    for (const name of readdirSync(STATE)) {
      const path = join(STATE, name);
      if (name.endsWith('.dirty') && statSync(path).mtimeMs < cutoff) rmSync(path, { force: true });
    }
  } catch {
    /* nothing to prune, or not ours to touch */
  }
}

/**
 * Close the previous turn. A mark alive at this point was written by a turn that
 * has already ended, so it can no longer be evidence about the one starting.
 *
 * Logged as `reset` rather than `clear`: the buddy reads this log to work out how
 * often work gets recorded without being asked, and counts a `clear` carrying
 * had:true as a turn that recorded voluntarily. Nothing recorded anything here.
 */
function userPrompt(payload) {
  const path = markPath(payload.session_id);
  if (existsSync(path)) {
    let markedAt = null;
    try {
      markedAt = stamp(new Date(statSync(path).mtimeMs));
    } catch {
      /* the mark is the point, not its timestamp */
    }
    rmSync(path, { force: true });
    log({ event: 'reset', session: payload.session_id ?? null, markedAt });
  }
  prune();
}

function postTool(payload) {
  const tool = payload.tool_name ?? '';
  const path = markPath(payload.session_id);

  // Both transitions are logged, not just the Stop decision they feed. A block
  // on a turn that did call buddy_observe is indistinguishable, from the Stop
  // entry alone, from a block on a turn that never recorded anything -- the
  // question is always which mark was written after the clear, and by what. The
  // session id is the discriminator: a background subagent whose PostToolUse
  // arrives under this session's id re-dirties a turn that already cleared, and
  // nothing but these two lines can show it happening.
  if (EDIT_TOOLS.has(tool)) {
    mkdirSync(STATE, { recursive: true });
    writeFileSync(path, tool);
    log({ event: 'mark', tool, session: payload.session_id ?? null });
  } else if (tool.endsWith('buddy_observe') && !failed(payload)) {
    const had = existsSync(path);
    rmSync(path, { force: true });
    log({ event: 'clear', tool, session: payload.session_id ?? null, had });
  }
}

function stop(payload) {
  // Already blocked once this turn -- let it stop, or we loop forever.
  if (payload.stop_hook_active) return;

  const path = markPath(payload.session_id);
  if (!existsSync(path)) {
    log({ event: 'stop', block: false, session: payload.session_id ?? null });
    return;
  }

  // The mark's own mtime is the evidence: a block whose mark was written after
  // this turn's clear is a re-dirty, and one whose mark predates it is a turn
  // that genuinely never recorded. Read it before the unlink, or it is gone.
  let markedAt = null;
  try {
    markedAt = stamp(new Date(statSync(path).mtimeMs));
  } catch {
    /* the mark is the point, not its timestamp */
  }

  // Clear before blocking: failing to nag beats wedging the session.
  rmSync(path, { force: true });
  prune();

  log({ event: 'stop', block: true, session: payload.session_id ?? null, markedAt });
  process.stdout.write(JSON.stringify({ decision: 'block', reason: REASON }));
}

async function readStdin() {
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  return Buffer.concat(chunks).toString('utf8');
}

try {
  const payload = JSON.parse(await readStdin());
  // Dispatched by name, with no default. An event this gate was never wired to
  // must not fall through to stop(), which is the one branch that writes a block
  // to stdout -- on UserPromptSubmit that stdout is read back as context, and on
  // anything else it is a session wedged by a hook that had no opinion.
  if (payload.hook_event_name === 'PostToolUse') postTool(payload);
  else if (payload.hook_event_name === 'UserPromptSubmit') userPrompt(payload);
  else if (payload.hook_event_name === 'Stop') stop(payload);
} catch {
  /* fail open */
}
process.exit(0);
