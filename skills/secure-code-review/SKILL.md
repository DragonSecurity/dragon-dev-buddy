---
name: secure-code-review
description: Adversarial security review of a diff, file, module, one pull request or a batch of them. Produces findings with a concrete exploit path, evidence at a line number, and a fix, ranked by severity. Use when someone says "review this for security", "is this safe", "security review this PR", "review these PRs", "review all the open PRs", "check this before it ships", "look at this handler", "did I do this auth check right", or after any change to authentication, authorization, input handling, crypto or file handling. Reads code as an attacker, not as a linter.
---

# Secure Code Review

A linter tells you what is unusual. A reviewer tells you what is exploitable. The difference is that a finding must come with a path an attacker can actually walk, from an input they control to an outcome they want.

This skill reviews with that bar. Anything that cannot be written as "an attacker who has X does Y and gets Z" is not a finding here. It goes in a short "worth cleaning up" list at the end, or it does not appear at all.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `project.primary_language`, `project.stack`, `security.exposure`, `security.data_sensitivity`, `security.auth_model`, `security.known_risk_areas`, `practice.sast_tool`, `output.reports_dir`, `skill_level`.

If a threat model exists in `output.reports_dir`, read it first and target the review at its top-ranked threats. A targeted review is worth several untargeted ones.

## Inputs

Ask only for what is missing:
- **What to review.** A diff, a branch, a file, a directory, one PR number, or several. Default to the working diff if the user is vague and a git repo is present. If the ask names more than one PR, or names none but a set ("the open PRs", "everything from this sprint"), go to **PR mode** below before anything else.
- **What the code is supposed to do.** Correct-looking code that does the wrong thing is invisible without this.
- **Whether it has shipped.** Changes the urgency of the recommendation, not the severity of the finding.

## Workflow

1. **Establish the trust position.** Before reading, answer: what does the caller of this code control, and what privilege does this code run with? Every finding is a gap between those two. Write it down in one line; it is the frame for everything after.

2. **Trace the inputs.** For each entry point in scope, follow every attacker-controlled value to where it is used. Query params, body fields, headers, cookies, path segments, uploaded filenames, queue messages, webhook bodies, environment in a multi-tenant runtime. Note every place one reaches a sink: a query, a filesystem path, a shell, a template, a redirect, a deserializer, a comparison that decides access.

3. **Check authorization at every object access.** Separately and explicitly, because this is where real applications break. For each place the code fetches or mutates an object identified by something from the request, ask: is ownership checked, and is it checked *here* or assumed to have happened earlier? "Assumed earlier" is a finding unless you can point at where.

4. **Work the language and framework list.** Load `references/review-patterns.md` and walk the section for this stack. Every ecosystem has a set of constructs that are almost always wrong, and they are worth checking mechanically rather than hoping to notice.

   If the codebase generates its API clients from a spec — an OpenAPI/Huma backend with generated Go/TS SDKs or a generated Terraform provider — load `references/codegen-pipeline.md` instead of reviewing generated files line by line. The review effort belongs on the spec, the handler, and the generator config, not the regenerated output.

5. **Read the error and logging paths.** The success path gets attention; the failure path leaks. Check what error responses distinguish, what gets logged, whether tokens or bodies land in logs, and whether a failure leaves the system in a partially-mutated state.

6. **Look for what is absent.** The hardest findings are omissions: no rate limit on the reset endpoint, no audit record on the privileged action, no revocation path for the long-lived token, no `LIMIT` on a query that grows. Scanning for absent controls needs a list, not intuition; the reference has one.

7. **Write the findings.** Each one gets: an attack sentence, evidence at `file:line`, severity with reasoning, and a fix. Rank with the rubric in the reference. If the fix is a one-line change, write the code. If it is structural, describe the shape and say why the one-line version is insufficient.

8. **State your coverage.** What you read, what you skipped, and what you could only check superficially. A review that implies completeness it did not achieve is how a finding gets missed twice.

## PR mode

A pull request is the workflow above plus three things a bare diff does not have: an author's claim about intent, a trust level, and a place to put the answer. Load `references/pr-review.md` for the commands and the full ruleset; the parts that change how you review:

- **Get the change, not the branch.** `gh pr diff <n>` gives `base...head`. Diffing against the tip of `main` shows unrelated commits and wastes the review on code nobody proposed.
- **Read the removals.** A deleted authorization check is a finding with the same attack sentence as a missing one. Attention drifts to added lines; go and look at the `-` side deliberately.
- **Open the file around any hunk touching auth or validation.** A diff shows what moved, not what is now true — the check that protects the changed line may be outside the hunk, present or freshly deleted.
- **Treat the description as a hypothesis.** It is the author's account of what they did, and it fills the "what is this supposed to do" input only until the diff contradicts it. A change that does more than it claims gets a line in the review even when the extra part is harmless.
- **A fork PR is untrusted code.** `isCrossRepository: true` means do not run its tests, build, install scripts or CI locally, and treat its CI config, lockfile and new dependencies as the first thing to read.
- **Draft before you post, and never approve.** Posting to GitHub is visible to the author and to anyone watching the repo — confirm with the user first. Approval is a human decision about accepted risk.

### Reviewing several PRs at once

Do not loop the full workflow over every PR in order. Depth is finite and spending it evenly is how the one dangerous PR gets the same three minutes as a lockfile bump.

1. **Enumerate and scope.** `gh pr list --json number,title,author,isCrossRepository,files,additions` — or take the numbers the user gave. Confirm the set back to the user before starting if you inferred it.
2. **Triage by what is touched, not by number or age.** Rank with the table in `references/pr-review.md`: auth, new request handlers and dangerous sinks earn the full workflow; CI, IaC and dependency manifests route to `secrets-and-config-audit` or `dependency-audit`; generated output, lockfiles, docs and formatting get a skim.
3. **Publish the triage before the reviews.** The user sees which PRs are getting depth and can redirect while it is still cheap.
4. **Review each in its own trust position.** Establish step 1 of the workflow per PR. A batch invites carrying assumptions from one diff to the next, and the assumption is usually "auth was handled".
5. **Then look across them.** This is the part a per-PR review cannot do: PRs touching the same file reviewed as their merged result, an endpoint added in one PR and its middleware loosened in another, and merge-order constraints where one PR is only safe once another lands. The reference lists these.
6. **Collapse repeats into one pattern.** The same defect in five PRs is one finding with five sites, usually a missing shared helper. Five copies bury everything else.
7. **Say what got which depth.** Per-PR coverage, and explicitly which PRs were skimmed and why. A batch review that reads as uniform when it was not is worse than reviewing fewer PRs.

## Output format

```markdown
# Security review: [scope]
[date] · [n] findings · [n] critical, [n] high, [n] medium, [n] low

**Trust position:** [caller controls X; this code runs with Y]

## Findings

### F1 — [title]   **[SEVERITY]**
**Attack:** [attacker sentence: precondition → action → outcome]
**Evidence:** `path/to/file.ts:44-51`
```[lang]
[the 3-8 lines that matter]
```
**Why it rates [severity]:** [reasoning, referencing exposure and data sensitivity]
**Fix:**
```[lang]
[the change, or the shape of it]
```
[one sentence on why the obvious smaller fix is not enough, if applicable]

## Worth cleaning up
- `file:line` — [not exploitable, but wrong]

## Coverage
Read closely: [...]   Skimmed: [...]   Not read: [...]
```

For a batch, wrap the per-PR reports in a triage table and a cross-PR section:

```markdown
# Security review: [n] pull requests
[date] · [n] findings across [n] of [n] PRs

## Triage
| PR | Title | Touches | Depth | Findings |
|----|-------|---------|-------|----------|
| #142 | [title] | session handling | full | 1 critical, 1 medium |
| #147 | [title] | lockfile only | skim → `dependency-audit` | — |

## Across the set
- **[title]** — #142 and #147 both change `middleware/auth.ts`; merged, [what the combination does].
- **Merge order:** #150 is only safe once #142 lands, because [reason].
- **Pattern:** [defect] appears in #144, #146, #151 — one fix in [shared location].

## PR #142 — [title]
[the single-PR format above: trust position, findings, worth cleaning up, coverage]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Reviewed <scope>: <n> findings, worst is <one clause naming the actual defect>."`
- `kind`: `other`
- `skills_used`: `["dragon-dev-buddy:secure-code-review"]`

One observation per review, not per finding — and one per *batch*, not one per PR in it. Summarise the batch as `"Reviewed <n> PRs: <n> findings, worst is <clause naming the actual defect> in #<pr>."`

Relay the reaction verbatim.

## File output

For a review of more than three findings, write a report to `output.reports_dir` as `YYYY-MM-DD-review-<scope>.md`. For a small diff, deliver in chat. A batch goes to one file, `YYYY-MM-DD-review-prs-<first>-<last>.md`, not one per PR — the cross-PR section is the reason the batch was worth running and it has nowhere to live in split files.

Do not modify source files; this skill finds, it does not fix. Do not post to GitHub, and never approve a PR, without the user asking for it in that turn. Hand confirmed findings to `security-test-writer` to lock them down, or apply fixes explicitly when the user asks.

## Reference library

Load these for depth when the task calls for it:
- `references/review-patterns.md`: per-ecosystem constructs that are almost always wrong, the absent-control checklist, the severity rubric, and the sink list for input tracing.
- `references/codegen-pipeline.md`: reviewing spec-driven codebases (Huma/OpenAPI → generated Go/TS SDKs and Terraform providers) — where the review effort goes, the spec as a security surface, auth handling in generated clients, and spec/client drift.
- `references/pr-review.md`: pull request mechanics — the `gh` and `git` commands, why removed lines are findings, fork PRs as untrusted code, the batch triage ranking, the cross-PR interactions a per-PR review cannot see, and how to post a review back without approving anything.

## Worked example

See `examples/secure-code-review-run.md` for a review of an authentication handler. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- Every finding has an attack sentence with a precondition and an outcome. Zero findings of the form "user input is not sanitized."
- Every finding cites `file:line` and quotes the lines that matter. A reader can verify without searching.
- Severity reasoning references the project's actual exposure and data sensitivity, not a generic CVSS intuition.
- Authorization at object access was checked explicitly and separately, and the report says so even when nothing was found.
- Absent controls were looked for, not just present-but-wrong ones.
- The coverage statement is present and names what was not read.
- Non-exploitable issues are in "worth cleaning up," not inflating the finding count.
- On a PR, removed lines were reviewed as deliberately as added ones, and the review says whether the change matches its description.
- On a batch, the triage was published before the reviews, every PR's depth is stated, repeated defects are collapsed into one pattern, and the cross-PR section exists even when it says "nothing interacts."
