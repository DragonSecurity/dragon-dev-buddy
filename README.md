# dragon-dev-buddy

A Claude plugin for people who write code that has to hold up under attack, with a
persistent companion attached.

27 skills covering the loop from *what could go wrong* through *ship it* to
*write up what happened*, for code and for the network it runs on — and the
working practice around that loop: interrogating a design before it hardens,
guarding the commands that destroy work, handing a session to the next one, and
writing the documents the agent itself reads. Every skill asks your
[buddy-mcp](https://github.com/DragonSecurity/buddy-mcp) companion what to load
before it starts and tells it what happened after, so the buddy gains XP, tracks
streaks, and learns which of your skills fits which kind of task.

## Install

**From the marketplace (Claude Code):**

```sh
/plugin marketplace add DragonSecurity/dragon-dev-buddy
/plugin install dragon-dev-buddy
```

The first command registers this repository as a marketplace; the second installs
the pack out of it. The marketplace entry lives here, in
`.claude-plugin/marketplace.json`, rather than in some index maintained
elsewhere — a listing kept in a second place is a second copy of the version
string with nothing holding it in step, and ours had already drifted.

Leave `autoUpdate` on and Claude Code re-pulls this repository on startup and
compares the `version` in the manifest against the version you have installed. A
higher number is the entire update signal: nothing inspects the skills, the
changelog or the tag. A release that ships new prose and forgets the bump is a
release nobody receives, which is why the version is now enforced by CI rather
than remembered — `go test ./...` fails when `plugin.json`, `marketplace.json`
and `CHANGELOG.md` disagree about what the current version is.

**From a checkout (Claude Code):**

```sh
git clone https://github.com/DragonSecurity/dragon-dev-buddy.git
cd dragon-dev-buddy
bundle="$(./scripts/build-plugin.sh)"   # dist/dragon-dev-buddy-<version>.plugin
version="${bundle##*-}"; version="${version%.plugin}"
unzip "$bundle" -d ~/.claude/plugins/cache/local/dragon-dev-buddy/"$version"/
```

The version comes back out of the filename the script just printed rather than
being typed in a second time. It is whatever `.claude-plugin/plugin.json`
currently says, and the cache directory is keyed by it — a directory that
disagrees with the bundle inside it is how you end up debugging a build you are
not running, and an unset variable there quietly unpacks everything one level up
where nothing looks for it.

**Claude desktop / Cowork:** Customize → Plugins → `+` → Create plugin → Upload
plugin → pick the `dist/dragon-dev-buddy-<version>.plugin` the build just wrote.

Then run `dragon-dev-buddy:buddy-setup` once per project. Every other skill reads
the config it writes, so nothing else has to ask you what your stack is again.

Installing the pack also brings the buddy server, because `.mcp.json` ships
inside the bundle and declares it. That server is compiled from source the first
time it starts, so the machine needs `git` and a Node toolchain and the first
launch is slow; [The buddy](#the-buddy) says why and how much.

### Versioning

The pack is semver, and the thing being versioned is the contract, not the word
count. A skill pack has two kinds of consumer — you, addressing a skill by name,
and the buddy server the skills talk to — so those are the two things a major
bump is reserved for:

- **Major** — a skill is renamed or removed, or the buddy contract changes: a
  tool this pack calls takes a different shape, or the required `buddy-mcp` major
  moves. Something outside the pack that referred to it now refers to nothing.
- **Minor** — a new skill, or a new mode inside an existing one. Everything that
  worked before still works and still answers to the same name.
- **Patch** — corrections that leave every name and every promise where it was.

Rewriting a workflow, sharpening a description or replacing a whole reference
file is a minor or a patch however much of the diff it accounts for, because
nothing that pointed at the pack has to change. Conversely, renaming one skill is
a major even though it is a one-line diff, because every `CLAUDE.md`, hook and
habit that names it breaks at once.

What changed in each release is in [CHANGELOG.md](CHANGELOG.md). A release of
this pack is a `vX.Y.Z` tag cut from `main`, which
`.github/workflows/release.yml` turns into a GitHub release with the bundle
attached — this pack is not published to a package registry any more than the
buddy server is. [CONTRIBUTING.md](CONTRIBUTING.md#releasing) has the steps and
the three files that have to agree before the tag exists.

## The buddy

The pack ships its own MCP server declaration. `.mcp.json` here declares the
`buddy` stdio server as `npx -y github:DragonSecurity/buddy-mcp#semver:^2`, so
installing the plugin brings a compatible companion with it. Before that the
server was wired by absolute path in one person's global config, which made the
dependency invisible: the pack read as installable and was not.

That specifier is a git spec, not a registry one. buddy-mcp is never published to
the npm registry — its releases are GitHub releases, and its **git tags are the
distribution channel**. `#semver:^2` resolves the range against the tags in that
repository, so `v2.1.0` is what `^2` picks up today and a `v3.0.0` tag is outside
it. The caret means exactly what it means on the registry — the same releases,
resolved from a different place.

Installing from a tag means the server is built on your machine, and that cost is
real rather than hidden. npm clones the matching tag, installs its
devDependencies — TypeScript among them — and runs its `prepare` script, which is
the compile. So the first launch after a new tag lands takes tens of seconds
instead of none, and the machine needs `git` and a full Node toolchain present,
not just a Node runtime able to execute a prebuilt file. npm caches the built
package, so the build is paid once per version rather than once per session; a
sandbox with no network or no `git` is where this fails, and it fails at the
first `buddy_status` rather than at install time.

The range is a major on purpose. This pack requires **buddy-mcp 2.x** and takes
any tag inside it, so fixes and new tools arrive without a release here. A
breaking change to the shape of the buddy tools is a buddy-mcp 3.0, and `^2`
declines it rather than letting every skill in the pack start calling a contract
that no longer exists on a machine where nothing was changed. That is what the
major is *for*: it is the only signal buddy-mcp has that the tool contract these
skills are written against has moved, and the caret is what turns the signal into
a refusal instead of a broken session. Moving to a buddy major is therefore a
deliberate release of this pack — the range moves, the skills get checked against
the new tools, and the pack takes a major bump of its own.

If you are working on buddy-mcp itself you want the server running from your
checkout rather than from a tag:

```sh
npm install && npm run build
claude mcp add buddy-dev --scope user -- node /absolute/path/to/buddy-mcp/dist/index.js
```

That entry does not replace the one this pack declares — it joins it. Claude Code
registers a plugin's servers under `plugin:<plugin>:<server>` and suppresses a
duplicate only when the command matches exactly, so `node …/dist/index.js` and
`npx -y github:…` are two different servers and both are started, each offering a
full set of `buddy_*` tools. They also share one buddy: neither declaration sets
`BUDDY_HOME`, so both open `~/.buddy-mcp/buddy.db`. Every tool call there loads
the buddy row, edits it in memory and writes the whole row back, so two processes
interleaving that lose one another's XP, level and daily-bonus flag, and one task
reported to both tool sets is recorded twice with nothing afterwards to tell the
duplicate from the real observation. Run one at a time — either remove the
checkout entry, or disable this plugin's server from `/plugins` while you are
working on the server.

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
finds the server via `$BUDDY_MCP_PATH`, then your own MCP config, then the
`.mcp.json` shipped beside it in the bundle — which is the only declaration a
marketplace install has, because Claude Code registers a plugin's servers for the
session without writing them into your config — and finally a `buddy-mcp` on
`PATH`, which exists only if you installed one globally yourself, from a checkout
or from the same git spec, since there is nothing on the registry to install. The
`Stop` gate logs every decision to `~/.claude/buddy-gate.log`.

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
| `project-memory` | Records what this codebase taught you — constraints, rejected approaches, gotchas — and loads it at the start of every future session. |
| `session-handoff` | Compacts a session in flight into a document a cold agent can act on — state, what was ruled out, the next concrete action — redacted, and written outside the working tree. |
| `skill-authoring` | Write the documents an agent reads — a skill in this pack, a `CLAUDE.md`, an `AGENTS.md` — against the levers that decide whether it fires and what it makes the agent do. |
| `security-audit-orchestrator` | Chains the pack in dependency order for a full audit in one pass. Use it when you do not yet know what you are looking for. |

**Fleet and network**

For work on devices rather than repositories. These read the `fleet` block of the
config; the rest of the pack ignores it.

| Skill | What it does |
| --- | --- |
| `change-window` | The pre-change gate for devices: intent vs running config, what the change can sever, a rollback you can still reach. |
| `fleet-drift-audit` | Where devices that should be identical are not, and which of that drift changes your security posture. |
| `segmentation-review` | Every path between two zones, not just the one with a firewall on it, then the ruleset on each. |
| `device-lifecycle` | Firmware and EoL exposure across the fleet, triaged for what applies, with an upgrade sequence. |

**Design and build**

| Skill | What it does |
| --- | --- |
| `design-interview` | Works the design tree in rounds, asking the whole frontier at once with a recommendation on every question, until nothing is left silently assumed. |
| `threat-model` | STRIDE pass over a system or feature, ranked, with mitigations that map to real files. |
| `secure-feature-build` | Idea to spec to implementation with the abuse cases written before the code. |
| `debug-and-fix` | Reproduce, isolate, fix, and prove the fix with a test. |
| `codebase-design` | The vocabulary for deep modules — interface, seam, adapter, depth — and the procedure for putting a scattered control behind one of them. |
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
| `git-guardrails` | Install a `PreToolUse` hook that refuses the git commands which destroy work no clone can recover — force pushes, pushes to a protected branch, `reset --hard`, `clean -f`, `branch -D` — and leaves ordinary branch pushes alone. |
| `runbook-wizard` | Turn the steps only a human can take — a console with no API, a credential the agent must never hold — into an interactive script: plan mode by default, prompts validated against the shape the provider issues, resumable stages, and no credential written anywhere git will commit. |

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

Six skills and four reference files are derived from MIT-licensed work by Matt
Pocock. [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md) reproduces that licence
and names which files came from where.
