#!/usr/bin/env node
/**
 * SessionStart hook: put this project's memories into context.
 *
 * A memory is a fact about this codebase that the code does not state and git
 * history does not imply — why a constraint exists, what was tried and rejected,
 * a gotcha that cost someone an afternoon. Kept next to the project rather than
 * in a session, because a session ends and the fact does not.
 *
 * Separate from buddy-session-start.mjs on purpose. That hook speaks MCP to a
 * server that may not be installed; this one reads files. Sharing a process
 * would mean a missing buddy costs you your memories, and a malformed memory
 * costs you your buddy.
 *
 * Deliberately no index file. An index is a second copy of every description
 * with nothing keeping it in step, and it goes stale the first time someone
 * edits a memory without touching it — so the listing is built by reading the
 * directory every time.
 *
 * Fails open and silent: no directory, no memories, unreadable file -> emit
 * nothing. Never wedge a session start over a note.
 */
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';

import { hookEnabled } from './lib/switches.mjs';

/** Where memories live, relative to the project root. */
const DIR = join('.dragon-buddy', 'memories');

/**
 * Past this many characters the bodies are dropped and only the one-line
 * descriptions are sent, with the paths to read on demand. Every session pays
 * this cost whether or not a memory turns out to be relevant, so the full text
 * is a convenience for a small set and never a standing tax on a large one.
 *
 * The listing is held to the same budget. Descriptions are one line each, but
 * one line times a few hundred memories is its own standing tax, and nothing
 * else bounds how many a long-lived repo accumulates. Past it, the newest are
 * named and the rest are counted, with the directory to list.
 *
 * DRAGON_BUDDY_MEMORY_MAX_CHARS overrides it; anything that is not a positive
 * integer is ignored rather than read as zero, which would silently send nothing.
 */
const DEFAULT_BUDGET = 6000;

function budget() {
  const n = Number.parseInt(String(process.env.DRAGON_BUDDY_MEMORY_MAX_CHARS ?? ''), 10);
  return Number.isInteger(n) && n > 0 ? n : DEFAULT_BUDGET;
}

/** The project root is wherever .dragon-buddy is, searching upward from cwd. */
function findRoot(start) {
  let dir = resolve(start);
  for (;;) {
    if (existsSync(join(dir, '.dragon-buddy'))) return dir;
    const parent = dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
}

/** Pull `name` and `description` out of the leading frontmatter block. */
function parse(raw, file) {
  const m = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/.exec(raw);
  if (!m) return null;
  const [, front, body] = m;
  const field = (key) => {
    const hit = new RegExp(`^${key}:[ \\t]*(.+)$`, 'm').exec(front);
    return hit ? hit[1].trim().replace(/^["']|["']$/g, '') : '';
  };
  const name = field('name') || file.replace(/\.md$/, '');
  const description = field('description');
  if (!description) return null;
  return { name, description, body: body.trim(), file };
}

function collect(root) {
  const dir = join(root, DIR);
  let entries;
  try {
    entries = readdirSync(dir).filter((f) => f.endsWith('.md')).sort();
  } catch {
    return [];
  }

  const out = [];
  for (const file of entries) {
    try {
      const path = join(dir, file);
      if (!statSync(path).isFile()) continue;
      const memory = parse(readFileSync(path, 'utf8'), file);
      if (memory) out.push({ ...memory, mtime: statSync(path).mtimeMs });
    } catch {
      /* one unreadable memory must not cost the others */
    }
  }
  return out;
}

function render(memories, limit = budget()) {
  const full = memories.reduce((n, m) => n + m.body.length + m.description.length, 0);

  const header =
    `Project memories for this repository, from \`${DIR}/\`. These are facts an ` +
    'earlier session recorded because the code does not state them. They describe ' +
    'what was true when written: if one names a file, function or flag, check it ' +
    'still exists before acting on it.';

  if (full <= limit) {
    const body = memories
      .map((m) => `### ${m.name}\n_${m.description}_\n\n${m.body}`)
      .join('\n\n---\n\n');
    return `${header}\n\n${body}`;
  }

  // Too much to carry every session. Send the descriptions, which are what a
  // relevance decision actually needs, and the path to read for the rest.
  const intro =
    `${header}\n\nListed rather than quoted, because the full set exceeds the ` +
    'context budget. Read the file when one looks relevant.';
  const line = (m) => `- **${m.name}** — ${m.description} (\`${DIR}/${m.file}\`)`;

  // Newest first when the list itself has to be cut: a memory written last week
  // is likelier to be about the code as it stands than one from a year ago.
  // Kept in name order when everything fits, so the listing does not reshuffle
  // every time a memory is touched.
  const byAge = [...memories].sort((a, b) => b.mtime - a.mtime);
  const shown = [];
  let used = 0;
  for (const m of byAge) {
    const cost = line(m).length + 1;
    if (used + cost > limit) break;
    shown.push(m);
    used += cost;
  }
  if (shown.length === memories.length) return `${intro}\n\n${memories.map(line).join('\n')}`;

  const rest = memories.length - shown.length;
  return (
    `${intro}\n\n${shown.map(line).join('\n')}\n\n` +
    `…and ${rest} older ${rest === 1 ? 'memory' : 'memories'} not listed, to stay inside the budget. ` +
    `List \`${DIR}/\` to see them all.`
  );
}

async function readStdin() {
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  return Buffer.concat(chunks).toString('utf8');
}

try {
  if (!hookEnabled('project-memory')) process.exit(0);
  let cwd = process.cwd();
  try {
    const payload = JSON.parse(await readStdin());
    if (typeof payload.cwd === 'string' && payload.cwd) cwd = payload.cwd;
  } catch {
    /* no payload, or not JSON -- process.cwd() is a fine fallback */
  }

  const root = findRoot(cwd);
  const memories = root ? collect(root) : [];

  if (memories.length) {
    process.stdout.write(
      JSON.stringify({
        hookSpecificOutput: {
          hookEventName: 'SessionStart',
          additionalContext: render(memories),
        },
      }),
    );
  }
} catch {
  /* fail open and silent */
}
process.exit(0);
