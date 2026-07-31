# Buddy operations reference

Mechanics, state, and what to do when it goes quiet. Figures below match buddy-mcp v2.

## Stages

| Stage | Emoji | From level |
| --- | --- | --- |
| Egg | 🥚 | 1 |
| Hatchling | 🐣 | 2 |
| Whelp | 🦎 | 5 |
| Dragon | 🐉 | 10 |
| Elder | 🐲 | 20 |
| Ascendant | ✨ | 35 |

Levelling costs `100 + 60n + 8n²` XP where `n = level - 1`. Level 2 is 100 XP, level 10 is a little over 1,000. One observation can carry an idle buddy up two levels at once.

## XP

Base, by kind:

| Kind | Base | Kind | Base |
| --- | --- | --- | --- |
| `deploy` | 30 | `refactor` | 20 |
| `feature` | 26 | `other` | 18 |
| `bugfix` | 24 | `docs` | 16 |
| `test` | 22 | `config` | 14 |

Then: `+25` if it is the first observation of the day, multiplied by an energy factor of `0.7 + 0.3 × (energy/100)`, multiplied by a streak factor of `1 + min(0.5, streak × 0.1)`. Floor of 1 XP; the buddy never learns nothing.

The daily bonus is keyed off the last *observation* day, not the last *seen* day. Checking in with `buddy_status` in the morning keeps your streak but does not spend the bonus.

## Energy

Starts at 100. Drains 4 per observation, recovers 10 per hour while you are away, clamped to 0–100. Below 25 the buddy stops reacting to the work and complains instead, and the energy multiplier drags XP down toward 70%.

Twenty-five observations in a session will flatten it. That is the intended ceiling, not a bug to route around.

## Mood

Score out of 100: `100 − neglect + streak bonus − drain`.

- **Neglect** kicks in after 18 hours of silence, at 1.2 points per hour beyond that, capped at 85.
- **Streak bonus** is `min(15, streak × 3)`, and only applies while you have been seen within 24 hours. A streak stops cheering it up the moment you stop keeping it.
- **Drain** is `(30 − energy) × 0.5` when energy is under 30.

Tiers: radiant ≥ 85, good ≥ 65, ok ≥ 45, low ≥ 25, bad below.

## Streaks

Counted in your local calendar days, not UTC, so a late night does not silently break one. `buddy_status` is enough to keep a streak alive. Missing a day resets to 1, but `longestStreak` is never lowered.

## State

Everything lives in `~/.buddy-mcp/`:

```
~/.buddy-mcp/buddy.db     SQLite (node:sqlite, WAL). State, events, skill registry, nudges.
```

Set `BUDDY_HOME` to relocate it — useful for a throwaway buddy while testing.

Note the directory name. `~/.buddy-mcp` is deliberately **not** `~/.buddy`, which belongs to a different companion project (`@fiorastudio/buddy`). If both are installed they keep separate state and do not interfere.

To start over, delete the database. You get a different buddy, with a different name and personality.

## Rename versus respawn

- **Rename** (`buddy_rename`) changes the display name only. Level, XP, personality, streak, energy, observation history and milestones all survive. A milestone is recorded noting the old name.
- **Respawn** means deleting the state and hatching fresh. Everything is lost, including the personality, which is the part people actually get attached to. There is no in-place personality edit, by design.

## Diagnosing a silent buddy

Work in order. Stop at the first thing that is wrong.

**1. Is the server registered?**

```sh
claude mcp list | grep buddy
```

Nothing back means it was never added:

```sh
claude mcp add buddy --scope user -- node /absolute/path/to/buddy-mcp/dist/index.js
```

**2. Is it pointing at the right build?**

Check the path in the registration against the checkout you are developing. A registration left over from another companion project, or from an older clone, is the most common cause of "the tools exist but they are the wrong tools."

Tell-tale: `buddy_skills`, `buddy_rename` and `buddy_advise` are missing, but tools like `buddy_pet`, `buddy_dream` or `buddy_respawn` are present. That is a different server (the Fable `~/.buddy` build). This pack targets buddy-mcp, whose surface is `buddy_status`, `buddy_observe`, `buddy_advise`, `buddy_skills`, `buddy_rename`. In particular `buddy_advise` (rank skills for a task before starting) only exists on buddy-mcp v2+ — if it is absent, the registered server is either older or the Fable build, and skills fall back to the static routing table in `buddy-setup`'s `references/setup-routing.md`.

**3. Is `dist/` stale?**

The server runs compiled output, not `src/`. After pulling changes:

```sh
cd /path/to/buddy-mcp && npm run build
```

Then restart the MCP connection. A `dist/` older than `src/` explains a tool that exists in the source but not in the session.

**4. Does it start at all?**

```sh
node /path/to/buddy-mcp/dist/index.js
```

It should print `buddy-mcp <version> ready (state: ...)` to stderr and then sit waiting on stdin. Anything else is a startup failure and the message is the diagnosis. Note that stdout is the MCP channel; human-facing output goes to stderr on purpose.

**5. Is the state readable?**

A corrupt database will surface as an error on every tool. Move it aside rather than deleting it, in case you want the history back:

```sh
mv ~/.buddy-mcp/buddy.db ~/.buddy-mcp/buddy.db.broken-$(date +%s)
```

The next call hatches a fresh buddy. Say clearly that this is a new companion, not the old one recovered.

## What never to do

- Never invent a status card. If the tool did not return one, report the failure.
- Never call `buddy_observe` speculatively to "top up" XP. The record is only useful because it is honest.
- Never report one observation per finding. One completed run, one observation.
