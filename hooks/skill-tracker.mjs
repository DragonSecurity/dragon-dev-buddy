#!/usr/bin/env node
/**
 * Tell the buddy which skills a session actually loaded, instead of relying on
 * the model to remember.
 *
 * buddy_advise ranks skills by what you have historically used for each kind of
 * work, and the only thing it learns from is `skills_used` on buddy_observe --
 * a list the model has to assemble from memory at the end of a task, often many
 * tool calls after the skill was loaded. When it forgets, the ranking learns
 * that the skill was not used, which is the one lesson it must not learn.
 *
 * Two roles, dispatched on hook_event_name:
 *
 *   PostToolUse  a successful Skill call appends its name to this session's
 *                pending list; a successful buddy_observe deletes the list it
 *                was sent.
 *   PreToolUse   on buddy_observe, merge the pending names into `skills_used`
 *                through updatedInput, so they land in the same call -- with the
 *                same kind and in the same transaction as the XP.
 *
 * Merging here rather than having the server read a file afterwards is the whole
 * design. The server runs one process per session and has no session id, so it
 * could not tell whose file was whose; and a second write after the observation
 * would have to guess the kind the observation was classified as.
 *
 * The list moves pending -> inflight when it is merged and is deleted only when
 * the observe call succeeds. A call that is refused, fails validation or errors
 * leaves it inflight, and the next observe sends it again. Nothing is lost to a
 * failed call, and the dedupe below keeps a retry from counting a skill twice.
 *
 * updatedInput replaces the tool's input rather than merging into it (measured
 * against Claude Code 2.1.280: a field the hook left out was dropped), so the
 * model's own arguments are spread back in. It also grants nothing: with no
 * permissionDecision, a call the user has not allowed is still refused.
 *
 * Fails open and silent: any error emits nothing, and the observe call goes
 * ahead exactly as the model wrote it.
 */
import { appendFileSync, mkdirSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';

import { hookEnabled } from './lib/switches.mjs';

const STATE = join(homedir(), '.claude', 'buddy-gate');
const LOG = join(homedir(), '.claude', 'buddy-gate.log');
const STALE_AFTER_MS = 7 * 24 * 60 * 60 * 1000;

/**
 * The buddy server's own limits on skills_used (zod: at most 10 entries of at
 * most 80 characters). Exceed either and the server rejects the whole call, so
 * a hook meant to add a skill would instead cost the user the observation.
 */
const MAX_SKILLS = 10;
const MAX_NAME = 80;

/**
 * A skill name is an identifier: letters, digits and the separators real names
 * use (`plugin:skill-name`). The Skill tool's input is model-written, and
 * anything else is not a name but a payload riding into a database and back out
 * into every later session's context.
 */
const SKILL_NAME = /^[A-Za-z0-9._:@/-]+$/;

function skillName(value) {
  if (typeof value !== 'string') return null;
  const name = value.trim();
  return name.length > 0 && name.length <= MAX_NAME && SKILL_NAME.test(name) ? name : null;
}

function paths(sessionId) {
  const safe = String(sessionId ?? '').replace(/[^A-Za-z0-9_-]/g, '') || 'unknown';
  return { pending: join(STATE, `${safe}.skills`), inflight: join(STATE, `${safe}.skills.inflight`) };
}

/** Names in a list file, one per line, validated on the way out as well as in. */
function readList(path) {
  try {
    return readFileSync(path, 'utf8').split('\n').map(skillName).filter(Boolean);
  } catch {
    return [];
  }
}

function isObserve(tool) {
  return typeof tool === 'string' && tool.endsWith('buddy_observe');
}

function failed(payload) {
  const response = payload.tool_response;
  return Boolean(response && typeof response === 'object' && (response.is_error || response.isError));
}

/** Local time, to match the timestamps the observe gate writes to the same log. */
function stamp(d = new Date()) {
  const pad = (n) => String(n).padStart(2, '0');
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}` +
    `T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  );
}

/**
 * Shares the gate's log so one file tells the whole story of a turn. The buddy's
 * compliance count reads that log and counts only `clear`, `reset` and `stop`, so
 * a new event name is invisible to it by construction.
 */
function log(fields) {
  try {
    appendFileSync(LOG, JSON.stringify({ at: stamp(), ...fields }) + '\n');
  } catch {
    /* the log is a convenience, never a reason to fail the hook */
  }
}

/** Drop lists from sessions that ended without ever observing. */
function prune() {
  try {
    const cutoff = Date.now() - STALE_AFTER_MS;
    for (const name of readdirSync(STATE)) {
      if (!/\.skills(\.inflight)?$/.test(name)) continue;
      const path = join(STATE, name);
      if (statSync(path).mtimeMs < cutoff) rmSync(path, { force: true });
    }
  } catch {
    /* nothing to prune, or not ours to touch */
  }
}

function postTool(payload) {
  if (failed(payload)) return;
  const { pending, inflight } = paths(payload.session_id);

  if (payload.tool_name === 'Skill') {
    const name = skillName(payload.tool_input?.skill);
    if (!name) return;
    mkdirSync(STATE, { recursive: true });
    appendFileSync(pending, name + '\n');
  } else if (isObserve(payload.tool_name)) {
    rmSync(inflight, { force: true });
    prune();
  }
}

function preTool(payload) {
  if (!isObserve(payload.tool_name)) return;
  const input = payload.tool_input;
  if (!input || typeof input !== 'object' || Array.isArray(input)) return;

  const { pending, inflight } = paths(payload.session_id);
  const tracked = [...new Set([...readList(inflight), ...readList(pending)])];
  if (tracked.length === 0) return;

  // Written before the pending list is removed, so an interruption between the
  // two leaves a name in both files -- which the dedupe absorbs -- rather than
  // in neither.
  writeFileSync(inflight, tracked.join('\n') + '\n');
  rmSync(pending, { force: true });

  // The model's own entries go first and are passed through untouched: this
  // hook adds to what the model said, it never edits it.
  const own = Array.isArray(input.skills_used) ? input.skills_used : [];
  const seen = new Set(own.map((s) => (typeof s === 'string' ? s.trim() : s)));
  const added = tracked.filter((s) => !seen.has(s));
  if (added.length === 0) return;
  const merged = [...own, ...added].slice(0, MAX_SKILLS);
  const kept = merged.slice(own.length);
  if (kept.length === 0) return;

  log({ event: 'skills', session: payload.session_id ?? null, added: kept });
  process.stdout.write(
    JSON.stringify({
      hookSpecificOutput: {
        hookEventName: 'PreToolUse',
        updatedInput: { ...input, skills_used: merged },
      },
    }),
  );
}

async function readStdin() {
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  return Buffer.concat(chunks).toString('utf8');
}

try {
  if (hookEnabled('skill-tracker')) {
    const payload = JSON.parse(await readStdin());
    if (payload.hook_event_name === 'PostToolUse') postTool(payload);
    else if (payload.hook_event_name === 'PreToolUse') preTool(payload);
  }
} catch {
  /* fail open */
}
process.exit(0);
