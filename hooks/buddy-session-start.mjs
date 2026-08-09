#!/usr/bin/env node
/**
 * SessionStart hook: call the real buddy_status and put the card in context.
 *
 * Why this is a command hook and not an `mcp_tool` hook: Claude Code parses an
 * mcp_tool hook's output as hook JSON. buddy_status returns a markdown card, so
 * the card is silently discarded -- the tool runs, its side effects land, and
 * nothing reaches the model. `hookSpecificOutput.additionalContext` is the only
 * channel into context, so we speak MCP stdio ourselves and fill it.
 *
 * Calling the tool rather than reading the database keeps the streak, heartbeat
 * and save side effects intact, and avoids duplicating engine logic here.
 *
 * Fails open and silent: no buddy, no server, bad protocol -> emit nothing. A
 * companion must never be able to wedge a session start.
 */
import { spawn } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';

const TIMEOUT_MS = 10_000;

/**
 * Find the buddy server the way the user actually installed it, in order of how
 * specific the signal is. Reading their MCP config means we spawn exactly what
 * Claude Code spawns, whatever layout they chose.
 */
function resolveServer() {
  if (process.env.BUDDY_MCP_PATH) {
    return { command: 'node', args: [process.env.BUDDY_MCP_PATH] };
  }
  for (const file of ['.claude.json', join('.claude', 'settings.json')]) {
    try {
      const cfg = JSON.parse(readFileSync(join(homedir(), file), 'utf8'));
      const server = cfg?.mcpServers?.buddy;
      if (server?.command) {
        return { command: server.command, args: server.args ?? [] };
      }
    } catch {
      /* not there, or not ours to parse -- try the next signal */
    }
  }
  // The declaration this pack ships, which for anyone who installed from the
  // marketplace is the only one there is. Claude Code registers a plugin's
  // .mcp.json for the session without writing it into the user's own MCP
  // config, so the loop above finds nothing on exactly the install path the
  // README documents, and without this the hook falls through to a name that is
  // only on PATH if someone put it there. Located from this file rather than
  // from CLAUDE_PLUGIN_ROOT so it still resolves when the hook is run by hand
  // out of a checkout.
  try {
    const cfg = JSON.parse(readFileSync(new URL('../.mcp.json', import.meta.url), 'utf8'));
    const server = cfg?.mcpServers?.buddy;
    if (server?.command) {
      return { command: server.command, args: server.args ?? [] };
    }
  } catch {
    /* not next to us, or not ours to parse -- try the next signal */
  }
  // Last resort. buddy-mcp is never published to the npm registry, so this name
  // is on PATH only if the user installed one globally themselves, out of a
  // checkout or from the same git spec `.mcp.json` declares. Spawning a command
  // that is not there costs nothing: the error handler below fails open like
  // every other path through this hook.
  return { command: 'buddy-mcp', args: [] };
}

const { command, args } = resolveServer();
const child = spawn(command, args, { stdio: ['pipe', 'pipe', 'ignore'] });

const send = (msg) => {
  try {
    child.stdin.write(JSON.stringify(msg) + '\n');
  } catch {
    finish(null);
  }
};

let done = false;
function finish(card) {
  if (done) return;
  done = true;
  if (card) {
    process.stdout.write(
      JSON.stringify({
        hookSpecificOutput: {
          hookEventName: 'SessionStart',
          additionalContext:
            'Your buddy, checked automatically at session start. ' +
            'Show this card to the user as-is:\n\n' +
            card,
        },
      }),
    );
  }
  child.kill();
  process.exit(0);
}

setTimeout(() => finish(null), TIMEOUT_MS).unref?.();
child.on('error', () => finish(null));
child.on('exit', () => finish(null));

let buf = '';
child.stdout.on('data', (chunk) => {
  buf += chunk;
  let nl;
  while ((nl = buf.indexOf('\n')) !== -1) {
    const line = buf.slice(0, nl);
    buf = buf.slice(nl + 1);
    if (!line.trim()) continue;
    let msg;
    try {
      msg = JSON.parse(line);
    } catch {
      continue; // servers are allowed to chatter; only JSON-RPC lines matter
    }
    if (msg.id === 1) {
      send({ jsonrpc: '2.0', method: 'notifications/initialized' });
      send({
        jsonrpc: '2.0',
        id: 2,
        method: 'tools/call',
        params: { name: 'buddy_status', arguments: {} },
      });
    } else if (msg.id === 2) {
      finish(msg.result?.content?.find((c) => c.type === 'text')?.text || null);
    }
  }
});

send({
  jsonrpc: '2.0',
  id: 1,
  method: 'initialize',
  params: {
    protocolVersion: '2025-06-18',
    capabilities: {},
    clientInfo: { name: 'dragon-dev-buddy-session-hook', version: '1.0.0' },
  },
});
