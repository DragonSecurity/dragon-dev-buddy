# Changelog

Notable changes to the pack. Versions track `.claude-plugin/plugin.json`.

## Unreleased

## 1.5.0 — 2026-08-18

### Added

- **`related_repos` in the config, so a repo can say which other repos it is
  entangled with.** Previously the only way an agent knew that this repo's
  plugin, SDK or upstream lived next door was a memory file on one machine,
  which does not travel and is not reviewable. `buddy-setup` now asks for these
  in the existing batch-one message and records them.

  Two decisions worth keeping. Entanglement is **declared, never inferred** —
  a projects folder holds dozens of sibling directories and proximity says
  nothing, so nothing is written that the user did not name. And `path` and
  `url` are separate optional fields rather than one polymorphic `source`: they
  answer different questions (can I read it from here, versus which repo is this
  canonically), a repo commonly has both, and an scp-style remote
  (`git@host:org/repo.git`) has no URL scheme to parse. `path` is the only field
  anything can resolve.

  `session-handoff` is the first skill to pull the key by name: a handoff is
  routinely picked up on a different machine, where a relative path resolves to
  nothing or to the wrong checkout, so a cross-repo reference now carries the
  `url` as well.

### Fixed

- **`skill-authoring` step 9 described the release machinery wrongly, in the two
  ways that cost a CI round.** It said the version lives on an entry in
  `.claude-plugin/marketplace.json`; that key no longer exists — since 1.4.3 the
  entry pins a release archive by URL and `sha256`, which the release workflow
  rewrites on the tag and which nobody should hand-edit. And it said the tests
  skipping `## Unreleased` means work in flight does not force a bump. That is
  true of `go test ./...` and false of CI, which fails any change that moves the
  shipped bundle without moving the version. Editing a skill is a bump in the
  same pull request, and the quality bar now says so.

## 1.4.3 — 2026-08-11

### Changed

- **The pack is now published as the zip it builds, not the tree it builds
  from.** The marketplace entry's source was `"./"`, which resolves to the
  checkout sitting beside `marketplace.json`. That is why the bundle could be
  wrong for two releases without anyone here noticing: in a working tree every
  file a skill reaches for exists whether or not `build-plugin.sh` packs it, so
  1.4.0 and 1.4.1 installed a working `git-guardrails` on this machine and a
  broken one everywhere else.

  The entry is now an `archive` source — the release asset, pinned by SHA-256.
  The artifact being distributed is the artifact being run, so a file left out
  of the bundle now fails the install for everybody, including whoever left it
  out. The digest matters as much as the URL: a release asset can be deleted and
  re-uploaded, so an unpinned URL would be no more immutable than a moved tag.

  Nothing changes for anyone installing from a local checkout, which is a
  separate marketplace.

- **The marketplace entry no longer carries a `version`.** Claude Code resolves
  a plugin's version from `plugin.json` before the marketplace entry, so this
  copy never decided anything. It is also unkeepable now that the release
  workflow rewrites the entry after the tag: it would sit one release behind
  `plugin.json` for as long as the generated pull request stayed open, failing a
  check on every single release.

- **The release workflow points the marketplace at each release it cuts.** A new
  `marketplace` job rewrites the entry's URL and digest and opens a pull request
  with the change. It is a pull request rather than a push because the
  organization's `require-dco` ruleset lists no bypass actors — `GITHUB_TOKEN`
  is refused on `main` exactly as a person is. The digest is taken from the
  asset downloaded back out of the release rather than the file on the runner,
  since the pin describes what users fetch; the two are compared and a mismatch
  fails the release.

  GitHub does not start workflows for events raised with `GITHUB_TOKEN`, so
  `ci.yml` will not run on that generated pull request and its check list will
  be empty. The job runs `go test ./...` against the edited tree before opening
  it, so a red tree never becomes a pull request.

- `TestMarketplaceSourceResolvesToThisPack` is now
  `TestMarketplaceSourceIsAPinnedRelease`, holding the entry to an archive
  source with a well-formed digest and a URL naming a version the changelog
  records. It checks against *a* released version rather than the newest one on
  purpose: between the tag and the generated pull request merging, the entry
  legitimately points one release back, and a check that goes red on every
  release is a check that gets turned off.

## 1.4.2 — 2026-08-11

### Fixed

- **Two skills shipped an install step that could not work anywhere but this
  checkout.** `git-guardrails` tells the user to copy
  `${CLAUDE_PLUGIN_ROOT}/scripts/block-dangerous-git.sh`, and `runbook-wizard`
  copies `${CLAUDE_PLUGIN_ROOT}/scripts/wizard-template.sh`. Neither file was in
  the list `scripts/build-plugin.sh` zips, so neither was in the bundle: from a
  marketplace install the copy fails, nothing is written, and the only symptom is
  a shell error the user has to already be reading for.

  For `git-guardrails` that is the precise failure the skill was written to
  prevent. Its own reference explains that a hook which fails open is worse than
  no hook, because what remains is a belief that destructive git is impossible
  here — and a guard that was never installed at all is that same belief with
  even less behind it. It was broken for every install of 1.4.0 and 1.4.1; only
  this repository, which happens to hold the file at the path the skill names,
  ever worked.

- **`THIRD-PARTY-NOTICES.md` is now in the bundle.** Six skills in it are derived
  from MIT-licensed work, and MIT requires its notice to travel with substantial
  portions. A copy that exists only in this repository does nothing for someone
  holding the installed plugin, which is the artifact those skills actually reach
  people through.

### Added

- `TestBundledScriptsExist` reads the zip list in `scripts/build-plugin.sh` and
  requires every `${CLAUDE_PLUGIN_ROOT}` path a skill names to both exist and be
  bundled. Checking only that the file exists passes in this checkout forever,
  which is exactly how the defect above survived two releases. Confirmed to
  discriminate: against the 1.4.1 build script it names all three missing files.
- `TestNoticesShipWithTheBundle`, holding the licence obligation to the artifact
  rather than to the repository.
- CI's shipped-file list now includes the three scripts an install receives and
  the notices file, so changing one without bumping the version fails the build.
  It names them individually rather than taking all of `scripts/`, because that
  directory also holds the build and the checks, which ship to nobody.

### Changed

- This repository now installs its own guard, at `.claude/settings.json`, pointed
  at `scripts/block-dangerous-git.sh` rather than at a copy under
  `.claude/hooks/`. The skill tells other repositories to copy the script,
  because there the source is a version-keyed plugin path that a link would
  dangle into on the next update. Here the repository *is* the source, so a
  second copy would only be a file free to drift from the one the tests run.

## 1.4.1 — 2026-08-11

### Fixed

- **The git guard no longer refuses `git push --tags`.** A push with no refspec
  sends the current branch, so that is the destination the guard judges. `--tags`
  is the exception: it sends tags and no branch at all, and the fallback read it
  as a push *of* whatever branch you were standing on. From `main` — which is
  exactly where you stand when cutting a release, having just fast-forwarded to
  the merge commit — publishing a release was refused by `push-protected`, citing
  a branch the command was never going to touch.

  `--follow-tags` is deliberately not included in the fix: it pushes the current
  branch alongside the tags, so from `main` it is a push to `main` and is still
  refused. An explicit refspec is still judged either way, so
  `git push origin main --tags` remains blocked.

  This is the failure mode the skill itself warns about — a guard that blocks
  ordinary work gets uninstalled, and an uninstalled guard protects nothing —
  reached by a rule that was individually correct, through a fallback three
  functions away from it.

### Added

- `scripts/check-git-guardrails.sh`, and a CI step that runs it. The guard
  shipped in 1.4.0 with no automated test and the defect above was found by hand
  during an install. It runs 48 cases against the real script, in a throwaway
  repository whose checked-out branch it controls, because a third of the rules
  depend on the branch the caller is standing on and running them in the pack's
  own checkout would make the result depend on whoever last ran `git checkout`.
  The allow table is longer than the block table on purpose: a guard that starts
  refusing ordinary work fails as completely as one that stops refusing, and only
  the first kind gets uninstalled.

## 1.4.0 — 2026-08-10

Six new skills, taking the pack from 21 to 27, and four harvests into skills
that were already here. All of it is derived from Matt Pocock's MIT-licensed
skill collection at https://github.com/mattpocock/skills, rewritten against
this pack's conventions and given the security argument that is the reason each
one earns a permanently loaded description. `THIRD-PARTY-NOTICES.md` is new and
records the licence, the file-by-file provenance, and which upstream skill each
piece came from.

The shape of the release is one admission: the pack was strong on the security
loop and had almost nothing on the practice around it. Every gap below is a
place where an existing skill was already relying on material the pack had
never written down.

### Added

- **`session-handoff`** — compacts a session in flight into a document a cold
  agent can act on within a minute: the goal, what is done with the SHA that
  proves it, what is mid-change and its exact state, what was ruled out and why,
  and the next concrete action. The pack had `project-memory` and read that as
  covering continuity. It does not: a memory is a durable fact about the
  codebase, loaded into every future session, and the state of one task in
  flight is neither durable nor a fact about the codebase. What was actually
  being lost was the rejected approach — the expensive omission, because the
  next session reaches the same wall by the same route, and usually lands on the
  approach that was abandoned, since it is the obvious one. The skill runs the
  phase-boundary check first (continuing, clearing or compacting is usually the
  better move), redacts every secret, and writes outside the working tree —
  `output.reports_dir` is read only so it can be avoided, because a directory
  that exists to be handed to other people is queued for distribution.

- **`skill-authoring`** — writes and edits the documents an agent consumes: a
  skill in this pack, a `CLAUDE.md`, an `AGENTS.md`, a reference file something
  points at. It works the levers that decide whether the agent reaches the
  material at all (the description as a context pointer, worded to trigger) and
  whether it takes the same process every run (completion criteria with a
  checkable bound, progressive disclosure, the no-op pass). Until now the
  pack's own authoring rules lived in `CONTRIBUTING.md` and in the test file,
  which is to say they were discoverable by whoever already knew — so the
  contract that decides whether a skill fires at all was the one thing the pack
  never taught. In a security pack a skill that does not fire is a control that
  was never applied, and a review that never ran is indistinguishable from a
  clean one. Ships the test contract as a checklist so a draft can be graded
  before `go test ./...` is run, and names the pack's own duplication debt — the
  `**Advise first.**` paragraph copied into every skill but two — as debt rather
  than paying it in the diff that added a skill.

- **`design-interview`** — works the design as a tree of decisions, computing
  the frontier each round and asking the whole of it at once with a recommended
  answer attached to every question, until nothing is left silently assumed.
  Every other skill in the pack opens with an `Inputs` section that asks once,
  flat, and that is the right shape only when the design is settled and merely
  undocumented. Nothing owned the case where the design itself is still argued
  about, so `threat-model` and `secure-feature-build` were modelling and
  specifying over whatever the first person to type had guessed. An unasked
  question is an undocumented assumption, and that is where the abuse case
  nobody wrote down lives; the expensive version is subtler still — a decision
  everyone believes is settled and that is quietly false, which no one asks
  about precisely because no one is uncertain about it.

- **`codebase-design`** — the vocabulary for deep modules (module, interface,
  implementation, depth, seam, adapter, leverage, locality) and the procedure
  for putting a scattered control behind one. Three skills were already using
  these words as though they had been defined somewhere: `security-test-writer`
  step 2 said "find the real seam" and never said what a seam is,
  `refactor-safely` extracts code without naming a shape to extract toward, and
  `secure-feature-build` asks for a design that makes the abuse case
  structurally impossible without saying where the structure goes. The security
  argument is the one the pack had been making implicitly: "every query must
  filter by tenant" is not a control, it is a convention with forty enforcement
  points and no enforcement, and depth is what converts it into something
  omission cannot express.

- **`runbook-wizard`** — turns a procedure only a human can carry out into an
  interactive bash script: a console with no API, a credential the agent must
  never hold, an approval from a named human, a cutover whose middle is a
  judgement call. It rules the agent out step by step first and does the agent's
  half immediately, so what survives into stages is only what genuinely needs a
  person. The pack could gate a deploy and could not hand over the manual half
  of one, which meant those steps were being re-explained to an agent every time
  or done from memory at the wrong hour. Ships the wizard library as
  `scripts/wizard-template.sh` — plan mode as the default for the whole run,
  prompts validated in-process against the shape the provider issues, secrets
  read hidden and piped to `gh` over stdin rather than argv, mode 600 on
  everything written, a gitignore-and-tracked refusal that fires before the
  human is asked to paste anything, and per-stage state so a cutover that stops
  half-way resumes instead of repeating a revocation. bash 3.2 throughout,
  because the machine in the room during a cutover is a macOS laptop.

- **`git-guardrails`** — installs a `PreToolUse` hook on `Bash` that refuses the
  git commands which destroy work no clone can recover: force pushes, pushes to
  a protected branch, `reset --hard`, `clean -f`, `branch -D`. `ship-it` gated
  what reached production and nothing gated the agent's own shell, where
  `git reset --hard` is as easy to type as `git status` and only one of the two
  is undoable — uncommitted work has no reflog. The design constraint is what
  makes it survive: it does not block `git push`, because a blanket push block
  is uninstalled the first afternoon it lands on an ordinary branch, and an
  uninstalled guard is worse than none since people still believe it is there.
  Ships as `scripts/block-dangerous-git.sh`. The pause it buys is also the one
  in which a leaked-credential history rewrite gets decided by a person rather
  than reflexively — rotation closes the exposure, the rewrite only hides the
  evidence — and that case hands off to `secrets-and-config-audit`.

### Changed

- **`debug-and-fix` now gates on a red-capable command.** Step 1 was "reproduce
  it first"; it is now a bar with a name attached — one command you have already
  run at least once, that drives the actual bug path and asserts the user's
  exact symptom, is deterministic, takes seconds and runs unattended. Isolation
  does not begin before that command exists, and the quality bar checks it,
  because reading code to build a theory before anything exists that can
  contradict the theory is the specific failure that costs whole afternoons.
  `references/debugging-method.md` roughly doubled to support it: ten ways to
  build a loop ranked best-first, the three axes to tighten it on (faster,
  sharper, more deterministic), instrumentation hygiene including the unique
  `[DEBUG-a4f2]` prefix that makes cleanup a provably complete grep, a
  measure-first branch for performance regressions where logs are the wrong
  tool, intermittent bugs reframed around raising the reproduction rate rather
  than chasing a clean repro, and an honest script for when no loop can be built
  — name which of the ten you tried, then ask for the one thing that would
  unblock it.

- **`security-test-writer` now rules out the tautological test.** Discrimination
  was the whole proof: see it red on the vulnerable code, green on the fix. That
  is necessary and not sufficient. An assertion that recomputes its expected
  value the way the code does — signing a token with the verifier's own helper,
  comparing output to `sanitise(input)`, blessing a snapshot of current
  behaviour — can discriminate cleanly and still be incapable of ever
  disagreeing with the implementation. Expected values now have to come from an
  independent source: a hand-written literal, the exploit payload, the spec.
  The reference gained that rule and the two other structural anti-patterns,
  implementation-coupled and horizontal slicing, with the note that coupling
  matters more here than elsewhere because a security test exists to outlive
  several refactors of the code it guards.

- **`secure-code-review` gained a spec-conformance axis.** New step 7 locates
  the originating spec — an issue reference in the commits or PR body, a path
  the user passed, a `secure-feature-build` spec under `output.reports_dir` —
  and reports three buckets against it, each quoting the spec line it rests on:
  missing or partial, not asked for, implemented wrongly. If no spec resolves it
  says so and skips, rather than reconstructing one from the diff, since a spec
  derived from the change can only ever agree with the change. The axis is
  reported under its own heading and never merged or reranked into the severity
  list: code that is individually secure can still omit the control the spec
  required, and merged into one list the louder axis decides the verdict while
  the quiet one reads as passing. The security case is the "not asked for"
  bucket — scope added during implementation was never specified, never threat
  modelled and never reviewed, so it arrives as attack surface with no abuse
  case attached.

- **`secure-feature-build`'s spec template gained an `Out of scope` section**,
  and the quality bar now checks that nothing from that list quietly arrived in
  the implementation. A spec that never draws its own boundary gets scope added
  during the build, and added scope is unspecified, unmodelled and unreviewed:
  no abuse case covers it and no negative test constrains it.

- **The existing skills now hand off to the new ones.** `threat-model` and
  `secure-feature-build` route to `design-interview` when the design is still
  unsettled rather than modelling over guesses; `refactor-safely`,
  `security-test-writer` and `debug-and-fix` route to `codebase-design` for the
  seam vocabulary they were already using; `secure-code-review` stops at a file
  or PR boundary and runs `session-handoff` rather than thinning the remaining
  reviews. `threat-model` also states the distinction it had been leaving to
  the reader: a trust boundary is not a seam, they are worth making coincide,
  and drawing seams as boundaries inflates the model with elements that produce
  filler.

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
