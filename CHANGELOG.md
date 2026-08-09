# Changelog

Notable changes to the pack. Versions track `.claude-plugin/plugin.json`.

## Unreleased

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
