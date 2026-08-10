# Changelog

Notable changes to the pack. Versions track `.claude-plugin/plugin.json`.

## Unreleased

## 1.3.1 — 2026-08-10

### Fixed

- **The observe gate no longer nags a turn that changed nothing.** Its dirty
  mark is keyed by session, and nothing bounded it to a turn, so an edit could
  outlive the turn that made it and be charged to the next one: a background
  subagent or workflow whose `PostToolUse` arrives under the parent's session id
  after the turn ended, a turn interrupted before `Stop` ever ran, and a session
  resumed by id with a mark still on disk from its previous run. The gate now
  runs on `UserPromptSubmit` and drops any mark left standing, so a mark can only
  mean "this turn changed code". Found in a real transcript: a workflow agent
  wrote a file 2m40s after its turn ended, and the next turn — a dozen `Bash`
  calls, no edits — was blocked for it.

  The reset is logged as `reset`, not as a `clear`, so it does not read as a turn
  that recorded voluntarily in the buddy's compliance stats. Those stats
  overcount prompted turns for as long as the pre-fix log stays inside the
  30-day window.

- The gate dispatches on `hook_event_name` with no default branch. Any event it
  is not wired to used to fall through to the `Stop` handler, the one branch that
  writes a block decision to stdout.

### Added

- `scripts/check-observe-gate.sh`, which runs the gate against a throwaway `HOME`
  and asserts what it marks, clears, resets and blocks. CI runs it.

## 1.3.0 — 2026-08-10

### Added

- **Four fleet and network skills**, for work on devices rather than
  repositories. `change-window` gates a production network change by diffing
  intent against the running config, working out what the change can sever, and
  requiring a rollback path that survives the change severing the path you would
  roll back over. `fleet-drift-audit` finds where devices that should be
  identical are not, and separates deliberate variation from decay.
  `segmentation-review` enumerates every path between two zones rather than the
  one with a firewall on it, then reads the ruleset on each.
  `device-lifecycle` triages firmware and end-of-life exposure across a fleet
  against vendor advisories, and sequences the upgrades.

- A `fleet` block in `config.example.json`, and the `buddy-setup` questions that
  fill it. `fleet.out_of_band` is the load-bearing one: `change-window` cannot
  gate a self-severing change without knowing whether there is a way back in.
  `fleet.managed` stays false for an ordinary application repository, and no
  other skill in the pack reads the block.

- Routing in `setup-routing.md` for a repository that manages a fleet, and the
  rule that `device-lifecycle` comes first when the trigger is a published
  advisory rather than general unease.

  These were written on 2026-08-06 in a second working copy of this repository
  and had never been committed from it. They are unchanged in substance here;
  what changed is the surrounding wiring, which had moved on by twelve commits.

## 1.2.1 — 2026-08-10

### Fixed

- `project-memory` told the user to verify the commit guard with
  `git commit --dry-run`, which **does not execute pre-commit hooks**. It exits 0
  against a guard that would have refused, so the verification step reported a
  working install and proved nothing — the precise failure the step exists to
  catch, written into the instructions for catching it. It now runs the hook
  script directly, which is conclusive and, unlike a real commit attempt, cannot
  leave a commit behind when the guard is broken.

## 1.2.0 — 2026-08-09

### Added

- **`project-memory`**, the seventeenth skill. Records what a codebase taught
  you — constraints with non-obvious reasons, approaches tried and rejected,
  gotchas that fail silently — as one fact per file in `.dragon-buddy/memories/`,
  and loads them at the start of every future session. A session is disposable
  and the knowledge in it is not; without somewhere to put it, each session
  rediscovers what the last one already paid for.

  Most of the skill is about refusing to write things down. A directory that
  restates the README costs context on every session forever and buries the
  memories that matter, so `references/memory-criteria.md` is a keep-or-drop test
  — durable, not derivable, would cost a future session real time — worked
  through on real candidates.

- `hooks/project-memory.mjs`, a `SessionStart` hook that reads the directory and
  puts the memories in context. Separate from `buddy-session-start.mjs` on
  purpose: that hook speaks MCP to a server that may not be installed, this one
  reads files, and sharing a process would mean a missing buddy costs you your
  memories. There is deliberately no index file — an index is a second copy of
  every description with nothing keeping it in step, so the listing is built by
  reading the directory each time. Past a context budget it sends the one-line
  descriptions and the paths instead of the bodies.

- `scripts/pre-commit-memory-guard.sh`, and a `.gitignore` rule naming
  `.dragon-buddy/memories/` specifically. Memories stay on the machine: they are
  working notes, and they collect paths, hosts and names that read very
  differently in a public repository than in an editor. The ignore rule is the
  control; the guard is the backstop for `git add -f` and for a repository whose
  rule drifted. It refuses to commit the directory at all rather than scanning it
  for secrets — "this does not get committed" is a fact, where "no credentials in
  this file" is a judgement call with false negatives.

  The guard resolves `core.hooksPath` rather than assuming `.git/hooks`. A
  machine with that set globally — dotfiles, husky — never consults `.git/hooks`,
  so the obvious install produces a guard that is never invoked, and a guard that
  fails open is worse than none. The skill installs where git actually looks and
  then verifies it refuses something.

- Tests pinning the ignore rule, the guard's presence and executable bit, and
  that every hook script in `hooks/` is registered in `hooks.json`. The last one
  exists because a written, installed-looking, never-registered hook is exactly
  how the observe gate's clear half went missing for a whole release cycle.

### Fixed

- `scripts/build-plugin.sh` now ships the commit guard. The skill tells the user
  to install it from `${CLAUDE_PLUGIN_ROOT}`, and `scripts/` was excluded from
  the bundle — so on the documented marketplace install the path did not exist.

## 1.1.3 — 2026-08-09

### Fixed

- The gate's nag named `mcp__buddy__buddy_observe` outright, which is only the
  tool's name when the buddy server is registered by hand. Installed the
  documented way, this pack declares the server itself and the tool is
  `mcp__plugin_dragon-dev-buddy_buddy__buddy_observe` — so the instruction sent
  the model looking for a tool that was not in its list, and the `ToolSearch`
  query it suggested returned nothing. The message names `buddy_observe` now and
  says the prefix depends on the install.

- The matcher test only covered the hand-registered name. It covers both, so a
  future edit cannot fix one install by breaking the other.

## 1.1.2 — 2026-08-09

### Fixed

- The observe gate blocked every turn that touched a file, whether or not the
  turn had called `buddy_observe`. The PostToolUse matcher read
  `buddy_observe`, but an MCP tool is addressed as `mcp__<server>__<tool>` and
  the client matches the whole name, so the clear half of the gate was never
  invoked once. The mark was set on every edit and nothing ever removed it.

  Found by the logging added in 1.1.1, which showed ten `mark` events and zero
  `clear` events in a session that had recorded three times. A test now pins
  the matcher against the fully qualified tool name — anchored, because
  `buddy_observe` occurs inside `mcp__buddy__buddy_observe` and an unanchored
  check passes against the broken matcher.

## 1.1.1 — 2026-08-09

### Fixed

- The observe gate logged only its Stop decision, which made a false nag
  impossible to diagnose: a block on a turn that *did* call `buddy_observe` and a
  block on a turn that recorded nothing produce the same single line. The mark
  and clear transitions are logged now, each with the session id that caused
  them, and a block records the mark's own mtime. That is what distinguishes a
  turn that never recorded from one whose mark was rewritten after it cleared —
  which is what a background subagent does when its `PostToolUse` arrives under
  the parent session's id.

## 1.1.0 — 2026-08-09

### Added

- **Plugin hooks** (`hooks/hooks.json`), so the buddy contract is enforced by the
  client instead of asked for in prose. `SessionStart` calls `buddy_status` and
  returns the card as context, `PostToolUse` marks the turn dirty on an edit and
  clears the mark on `buddy_observe`, and `Stop` blocks once if the mark
  survives. An instruction in `CLAUDE.md` is advisory and gets skipped — most
  often when the client defers MCP tools behind a search step, so `buddy_*` is
  not in the tool list to begin with. Across the 38 code-editing sessions
  measured before the hooks landed, 26% never asked for the opening status and
  16% never recorded the work at all. Both hooks fail open and silent, so a
  missing buddy or a protocol change costs you the check-in rather than the
  session.
- `secure-code-review`: **PR mode** — reviewing a pull request rather than a bare
  diff. Covers fetching `base...head` rather than the branch tip, reading removed
  lines as deliberately as added ones, treating the PR description as a
  hypothesis to check the diff against, and handling a fork PR as untrusted code
  that is never executed locally.
- `secure-code-review`: **batch review** of several PRs in one pass — triage by
  what each diff touches, published before the reviews start; per-PR trust
  positions; and a cross-PR pass for the defects a per-PR review structurally
  cannot see (add-then-loosen across two PRs, merge-order constraints, one
  pattern repeated across many PRs).
- `secure-code-review`: `references/pr-review.md` with the `gh`/`git` mechanics,
  the batch triage depth table, and how to post a review back without approving.
- `ship-it`: a **can this land** gate. Every other gate in the skill asks whether
  a change should ship; none asked whether it can, and that is the one people
  skip until a push is rejected. It reads the branch rules, checks whether each
  required status check can even run on the route being taken, and checks the
  allowed merge methods — a check supplied by an app that only reports on pull
  requests can never be satisfied by a direct push, and the rejection arrives
  after the review is done and the author has moved on.
- `dependency-audit`: an **inbound mode** for the case where the update PRs
  already exist and nobody knows which to merge. It rejoins the main workflow at
  the reachability step rather than duplicating it, treats a stalled automerge as
  a broken control to diagnose before triaging a single PR by hand, reads the
  newly resolved packages in the lockfile as the meaningful diff rather than the
  version in the manifest, and closes superseded PRs instead of merging two that
  rewrite the same lines.
- `internal/skillpack`: a Go test suite that validates the pack's structure and
  conventions. See [CONTRIBUTING.md](CONTRIBUTING.md).
- CI running `gofmt`, `go vet` and `go test` on every push and pull request, and
  building the installable bundle as an artifact.
- `scripts/build-plugin.sh` to produce `dist/dragon-dev-buddy-<version>.plugin`.
  The version is in the filename because two downloads called the same thing
  cannot be told apart once they are in the same directory, and the one that
  gets installed is whichever overwrote the other.
- `.claude-plugin/marketplace.json`, so the pack installs with
  `/plugin marketplace add DragonSecurity/dragon-dev-buddy` and Claude Code's
  auto-update can pull the repository and compare the published version against
  the installed one. The only marketplace entry used to live outside the
  repository and be maintained by hand, which meant a second copy of the version
  string with nothing keeping it in step — and it had already drifted.
- `.mcp.json` declaring the `buddy` stdio server every skill in this pack talks
  to. The server was previously wired by absolute path in one person's global
  config, so the dependency was invisible and the pack was not actually
  installable by anyone else. The specifier is the git form,
  `github:DragonSecurity/buddy-mcp#semver:^2`, because that server is never
  published to the npm registry — its releases are GitHub releases and its git
  tags are the distribution channel, so `^2` resolves against those tags. The
  range is a major on purpose: it picks up fixes within the major on its own,
  and a breaking change to the shape of the buddy tools becomes a major bump
  that this range correctly refuses.
- `CONTRIBUTING.md`, `SECURITY.md`, this changelog.

### Fixed

- The `Stop` gate nagged turns that had already recorded their work. It decided
  by reading the transcript, and `Stop` races the transcript writer — since
  `buddy_observe` is by convention the last tool call of a turn, its entry is
  exactly the one most likely to be missing when the gate reads. Nine of the
  eighteen blocks measured were false positives, each costing the buddy a
  duplicate observation, inflated XP and a duplicate row in the skill-ranking
  data. The gate now tracks the turn with a marker file written by `PostToolUse`,
  which fires when the tool actually runs, so there is nothing left to race. It
  was ported from Python to Node in the same change: `python3` is not guaranteed
  present on macOS, and a hook that fails open and silent would have been
  invisibly dead rather than loudly broken.
- `secure-code-review` and `dependency-audit` disagreed about lockfiles. The
  batch triage table put "lockfile-only" in the skim row while the new inbound
  mode called the lockfile the meaningful diff of an update PR; both were right
  about their own case and neither said so, leaving the reader to guess. The
  distinction is whether anybody chose the change — incidental churn from
  someone's feature is noise, a dependency update PR's lockfile is the entire
  payload — and each reference now names the other case.
- `ship-it` had a worked example that no `SKILL.md` section pointed at, so the
  model never loaded it. Found by the new test suite.
- The README omitted `security-audit-orchestrator` entirely and undercounted the
  pack at fourteen skills. Also found by the test suite, which now pins the count.
- A local absolute path (`/Users/...`) in a `buddy-companion` example.

## 1.0.0

Initial pack: 16 skills, `buddy-setup` onboarding, and the buddy MCP reporting
contract.
