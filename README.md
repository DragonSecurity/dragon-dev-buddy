# dragon-dev-buddy

A Claude plugin for people who write code that has to hold up under attack, with a
persistent companion attached.

16 skills covering the loop from *what could go wrong* through *ship it* to
*write up what happened*. Every skill asks your
[buddy-mcp](https://github.com/DragonSecurity/buddy-mcp) companion what to load
before it starts and tells it what happened after, so the buddy gains XP, tracks
streaks, and learns which of your skills fits which kind of task.

## Install

**From a checkout (Claude Code):**

```sh
git clone https://github.com/DragonSecurity/dragon-dev-buddy.git
cd dragon-dev-buddy
./scripts/build-plugin.sh          # writes dist/dragon-dev-buddy.plugin
unzip dist/dragon-dev-buddy.plugin -d ~/.claude/plugins/cache/local/dragon-dev-buddy/1.0.0/
```

**Claude desktop / Cowork:** Customize → Plugins → `+` → Create plugin → Upload
plugin → pick `dist/dragon-dev-buddy.plugin`.

Then run `dragon-dev-buddy:buddy-setup` once per project. Every other skill reads
the config it writes, so nothing else has to ask you what your stack is again.

## The buddy

These skills assume the `buddy` MCP server is registered. From your
[buddy-mcp](https://github.com/DragonSecurity/buddy-mcp) checkout:

```sh
npm install && npm run build
claude mcp add buddy --scope user -- node /absolute/path/to/buddy-mcp/dist/index.js
```

Tools used by this pack: `buddy_advise` before work, `buddy_observe` after it,
and `buddy_status`, `buddy_skills`, `buddy_rename` for looking after the
companion itself. If the server is absent every skill still works; it just skips
the advice and the reporting.

### Hooks

Installing this plugin registers two hooks from `hooks/hooks.json`; there is
nothing to add to `settings.json`.

| Event | What it does |
| --- | --- |
| `SessionStart` | Calls `buddy_status` and puts the card in context, so the check-in happens whether or not the model remembers to ask. |
| `PostToolUse` | Marks the session dirty on an edit, clears the mark on `buddy_observe`. |
| `Stop` | Blocks once if the session is still dirty, so work does not go unrecorded. |

An instruction in `CLAUDE.md` is advisory and gets skipped — most often when the
client defers MCP tools behind `ToolSearch`, so `buddy_*` is not in the tool list
at all. Hooks are enforced by the client, which is why the calls live here.

Both hooks fail open and silent: no buddy, no server, or a protocol change means
they emit nothing rather than wedging your session. `buddy-session-start.mjs`
finds the server via `$BUDDY_MCP_PATH`, then your MCP config, then `buddy-mcp` on
`PATH`. The `Stop` gate logs every decision to `~/.claude/buddy-gate.log`.

The gate tracks the turn with a marker file under `~/.claude/buddy-gate/` rather
than by reading the transcript. `Stop` races the transcript writer, and since
`buddy_observe` is normally the last tool call of a turn, its entry is the one
most likely to be missing when the gate reads — which nagged turns that had
already recorded, and cost the buddy a duplicate observation each time.
`PostToolUse` fires when the tool runs, so there is nothing to race.

The two halves feed each other. `buddy_observe(skills_used)` is what trains the
ranking that `buddy_advise` returns, which is why every skill in this pack passes
it — a skill that reports nothing makes the advice worse for every skill.

## Skills

**Start here**

| Skill | What it does |
| --- | --- |
| `buddy-setup` | Interviews you once, writes `.dragon-buddy/config.json`, hatches the buddy. |
| `buddy-companion` | How to keep the companion healthy and read what it has learned. |
| `security-audit-orchestrator` | Chains the pack in dependency order for a full audit in one pass. Use it when you do not yet know what you are looking for. |

**Design and build**

| Skill | What it does |
| --- | --- |
| `threat-model` | STRIDE pass over a system or feature, ranked, with mitigations that map to real files. |
| `secure-feature-build` | Idea to spec to implementation with the abuse cases written before the code. |
| `debug-and-fix` | Reproduce, isolate, fix, and prove the fix with a test. |
| `refactor-safely` | Change structure without changing behavior, with a characterization-test net. |

**Audit**

| Skill | What it does |
| --- | --- |
| `secure-code-review` | Adversarial review of a diff, a file, one PR or a batch of them. Findings only, each with a concrete failure path. |
| `dependency-audit` | Triage SCA output into what is actually reachable, then a staged upgrade plan. |
| `secrets-and-config-audit` | Hunt live credentials, weak config and IaC misconfiguration. Rotation comes first. |

**Respond**

| Skill | What it does |
| --- | --- |
| `vuln-triage` | Turn a report or CVE into severity, real exploitability, and a fix. |
| `security-test-writer` | Convert a finding into a regression test that fails before the fix. |
| `hardening-playbook` | Close the gap between shipped and defensible, in priority order. |
| `incident-response` | Triage, contain, eradicate, recover, and write the timeline while it is fresh. |
| `pentest-report` | Turn findings into a report an engineer can act on and an exec can read. |

**Ship**

| Skill | What it does |
| --- | --- |
| `ship-it` | The pre-deploy gate: tests, security checks, blast radius, rollback. |

## Conventions

- Config lives at `.dragon-buddy/config.json`. Add it to `.gitignore` if the
  `engagement` block names a client.
- Reports are written as markdown to `output.reports_dir` (default
  `docs/security/`), one file per run, dated.
- Skills report themselves to the buddy as `dragon-dev-buddy:<skill-name>`, which
  is exactly how buddy-mcp's skill discovery qualifies plugin skills.
- Skills that can produce exploit code check `engagement.authorized_scope` first
  and will not go further without it.

## Development

The pack is documentation, so nothing here fails the way a compiler failure
does — a broken reference link or a skill missing from the README just quietly
does less than it claims. The test suite is what makes that loud.

```sh
go test ./...
```

It asserts, across every skill: frontmatter name matches the directory, the
description carries a `Use when` trigger clause and fits inside Claude Code's
1024-character limit, every `references/` and `examples/` file is both reachable
from a SKILL.md and actually exists (including cross-skill handoffs), every skill
ships a worked example and a quality bar, every skill observes under its own
qualified name, no local absolute path leaked into the docs, and the README lists
exactly the skills the pack contains — count included.

CI runs the same command on every push and pull request. See
[CONTRIBUTING.md](CONTRIBUTING.md) for what adding a skill involves.

## License

Apache-2.0. See [LICENSE](LICENSE).
