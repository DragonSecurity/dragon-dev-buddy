---
name: ship-it
description: The pre-deploy gate. Runs the checks that should pass before anything reaches production — tests, security-sensitive diffs, secrets, blast radius, rollback plan — and gives a clear ship or no-ship with the reasons. Use when someone says "ship it", "ready to deploy", "can I push this", "pre-deploy check", "is this safe to release", "about to go to prod", or before any production deploy. A gate that says yes when it should say no is worse than no gate.
---

# Ship It

The last check before production is the cheapest place to catch the thing that would otherwise page someone at 3am. This skill is that gate: it runs the checks that matter before a deploy, weighs what it finds, and gives a straight answer — ship, or do not ship, and exactly why. Its value is entirely in being willing to say no. A gate that always says yes is theatre.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `project.runtime`, `project.deploy_target`, `security.exposure`, `security.data_sensitivity`, `practice.test_command`, `practice.build_command`, `practice.lint_command`, `practice.ci`, `output.reports_dir`.

## Inputs

Ask only for what is missing:
- **What is shipping.** The diff, the branch, the release range. Default to the diff against the deploy branch.
- **Where it is going.** Production, staging, canary. The bar is highest for production with `public` exposure.
- **Anything unusual about this release.** A migration, a config change, a dependency bump, a first deploy of something new — these change which checks matter most.

## Workflow

Run the gates. Each is pass, warn, or fail. Report all of them; do not stop at the first failure, because the user needs the whole picture to decide.

1. **Build and tests.** `practice.build_command` and `practice.test_command` must pass. Run them; do not take "they passed earlier" on faith. A failing build or a red test is an automatic no-ship. Flaky tests are a warn with the specific flake named, not a silent pass.

2. **The diff, read for risk.** Actually read what is shipping, focused on the security-sensitive surface: changes to authentication, authorization, input handling, crypto, secrets handling, or anything on the money or data path. A change there triggers a recommendation to run `secure-code-review` before shipping if it has not been reviewed. Diff size is a signal — a 4000-line diff going to prod on a Friday is a warn on its own.

3. **Secrets in the diff.** Scan what is being shipped for credentials — the same check as `secrets-and-config-audit`, scoped to the diff. Any hit is an automatic no-ship until resolved; a secret reaching production history is not undoable by a later commit.

4. **Migrations and irreversibility.** If the release includes a schema migration or any irreversible operation, check: is it backward-compatible with the currently-running code (for zero-downtime), does it have a down path, and does a column drop or data transform have a backup behind it. An irreversible migration with no backup is a no-ship regardless of how good the code is.

5. **Blast radius.** State what this deploy can break if it is wrong. What depends on the changed code, what data it touches, whether it is behind a flag, and whether it can be rolled out gradually. A change to a shared library that forty services import is a different risk than a change to one leaf handler, and the user should see that named.

6. **Rollback plan.** There must be one, and it must be real. How is this reverted, how fast, and does the revert itself carry risk (a migration that ran, a cache that was poisoned, a message already sent)? "We'd roll back" is not a plan; "revert the deploy, the migration is additive so it can stay, ~3 minutes" is. No credible rollback is a warn that escalates to no-ship as blast radius grows.

7. **Can this actually land.** Every gate above asks whether the change *should* ship. This one asks whether it *can*, and it is the one people skip until a push is rejected. Check the mechanics of the target branch:

   - **Branch rules and protections.** Required status checks, required reviews, linear-history or signed-commit requirements, and whether the branch accepts direct pushes at all. On GitHub: `gh api repos/{owner}/{repo}/rules/branches/{branch}`.
   - **Whether the required checks can run on the route you are taking.** This is the trap. A check supplied by an app that only reports on pull requests can never be satisfied by a direct push, so "push to main" is not merely discouraged, it is impossible. Read each required check and ask what triggers it.
   - **Commit message requirements.** DCO or `Signed-off-by:` trailers, conventional-commit prefixes, an issue reference. These are enforced against every commit in the range, not just the tip, and a rebase or squash can drop a trailer a hook added.
   - **Allowed merge methods.** `gh api repos/{owner}/{repo} --jq '{merge:.allow_merge_commit, squash:.allow_squash_merge, rebase:.allow_rebase_merge}'`. If the release is deliberately split into commits that each stand alone, a squash-only repository will silently destroy that structure — flag it before the merge, not after.
   - **That you can push at all.** Credentials, SSO authorization on the token, write access to a protected branch.

   Failures here are rarely dangerous, but they are always wasted time, and they arrive at the worst moment — after the review is done and the author has moved on. Report a blocker as a fail with the specific rule named and the route that would satisfy it ("main requires the DCO check, which only runs on a pull request — open one rather than pushing"). This gate is cheap; if the user is iterating, run it before the slow gates rather than after.

8. **Give the verdict.** One of:
   - **Ship** — gates pass, risk understood, rollback exists. Say so plainly.
   - **Ship with conditions** — go, but do X first or watch Y after. Name them specifically.
   - **Do not ship** — a gate failed. State which, and what would clear it.

   Do not hedge. The user came for a decision. Give it, with the reasons, and let them override — but make them override a clear no, not a mush.

## Output format

```markdown
## Ship check: [what] → [target]

| Gate | Result | Detail |
| Build | pass/fail | |
| Tests | pass/warn/fail | [count, any flakes named] |
| Sensitive diff | pass/warn | [what security-relevant code changed] |
| Secrets | pass/fail | |
| Migrations | pass/warn/fail/na | [reversible? backup?] |
| Blast radius | — | [what breaks if this is wrong] |
| Rollback | pass/warn/fail | [the actual plan and its time] |
| Can it land | pass/fail | [branch rules, required checks, merge method, signoff] |

## Verdict: **[SHIP / SHIP WITH CONDITIONS / DO NOT SHIP]**
[the reasoning, and any conditions or blockers, specifically]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Ship check on <what>: <verdict> — <one clause>."`
- `kind`: `deploy`
- `skills_used`: `["dragon-dev-buddy:ship-it"]`

`deploy` is the highest-XP observation kind, which fits — the gate before production is the moment that matters most. Report the check regardless of the verdict; a caught no-ship is as much work as a clean ship. Relay the reaction verbatim.

## File output

Usually none — the verdict goes in chat where the decision is made. For a regulated environment that needs a deploy record, write the check to `output.reports_dir` as `YYYY-MM-DD-ship-check-<what>.md`. This skill does not deploy; it decides whether you should. Running the actual deploy is a separate, explicit action the user takes.

## Reference library

Load these for depth when the task calls for it:
- `references/ship-gates.md`: the full gate checklist with pass/warn/fail thresholds, zero-downtime migration patterns, blast-radius estimation, rollback plan patterns by deploy type, and the release shapes that warrant extra caution.

## Worked example

See `examples/ship-it-run.md` for a gate that says no to a payout change and then clears the follow-up. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- Build and tests were actually run for this check, not assumed from earlier.
- The diff was read for security-sensitive changes, and an unreviewed sensitive change was flagged.
- The diff was scanned for secrets; any hit is a hard no-ship.
- Migrations were checked for reversibility and backward-compatibility, not just correctness.
- Blast radius is stated concretely — what breaks if this is wrong, named.
- A real rollback plan exists, with a time, or its absence is the finding.
- The branch rules were read, not assumed — including whether each required check can run on the route being taken, and whether the allowed merge method preserves the commit structure the release depends on.
- The verdict is unambiguous. Ship, conditions, or no-ship — never a hedge the user has to interpret.
- The gate is willing to say no. If it never would, it is not a gate.
