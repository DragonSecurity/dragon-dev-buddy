#!/usr/bin/env node
/**
 * Nag once if a turn changed code but never told the buddy.
 *
 * Two roles, dispatched on hook_event_name:
 *
 *   PostToolUse  an edit tool marks the session dirty; buddy_observe clears it.
 *   Stop         if the session is still dirty, block once and clear the mark.
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
const REASON =
  'This turn changed code but never recorded it with the buddy. ' +
  'Call mcp__buddy__buddy_observe with a one-sentence summary of what you did, ' +
  'passing skills_used if you invoked any skills, then relay the reaction to the ' +
  'user verbatim. If the tool is not in your tool list it is deferred -- run ' +
  'ToolSearch with query "select:mcp__buddy__buddy_observe" first.';

function markPath(sessionId) {
  const safe = String(sessionId ?? '').replace(/[^A-Za-z0-9_-]/g, '');
  return join(STATE, (safe || 'unknown') + '.dirty');
}

/** Local time, to match the timestamps already in the log. */
function stamp() {
  const d = new Date();
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

function postTool(payload) {
  const tool = payload.tool_name ?? '';
  const path = markPath(payload.session_id);

  if (EDIT_TOOLS.has(tool)) {
    mkdirSync(STATE, { recursive: true });
    writeFileSync(path, tool);
  } else if (tool.endsWith('buddy_observe') && !failed(payload)) {
    rmSync(path, { force: true });
  }
}

function stop(payload) {
  // Already blocked once this turn -- let it stop, or we loop forever.
  if (payload.stop_hook_active) return;

  const path = markPath(payload.session_id);
  if (!existsSync(path)) {
    log({ event: 'stop', block: false });
    return;
  }

  // Clear before blocking: failing to nag beats wedging the session.
  rmSync(path, { force: true });
  prune();

  log({ event: 'stop', block: true });
  process.stdout.write(JSON.stringify({ decision: 'block', reason: REASON }));
}

async function readStdin() {
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  return Buffer.concat(chunks).toString('utf8');
}

try {
  const payload = JSON.parse(await readStdin());
  if (payload.hook_event_name === 'PostToolUse') postTool(payload);
  else stop(payload);
} catch {
  /* fail open */
}
process.exit(0);
