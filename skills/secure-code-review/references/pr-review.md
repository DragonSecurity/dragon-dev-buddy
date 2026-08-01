# Reviewing pull requests

Everything here is about the difference between reviewing *code* and reviewing *a proposed change to code*. The security questions are the same; what changes is what you can see, what you are allowed to trust, and what you do with the answer.

## Getting the change

```bash
gh pr view <n> --json number,title,body,author,baseRefName,headRefName,isCrossRepository,mergeable,files,additions,deletions
gh pr diff <n>          # base...head — the change itself, not everything base gained since the branch
gh pr checks <n>        # CI status; a failing security job is a finding you did not have to find
```

Without `gh`, or for a PR on a remote you have not authenticated to:

```bash
git fetch origin pull/<n>/head:pr-<n>
git diff $(git merge-base main pr-<n>)..pr-<n>
```

Use the merge base. `git diff main..pr-<n>` shows unrelated changes `main` picked up since the branch started, and you will waste a review on code nobody is proposing.

## The diff is not enough on its own

A diff shows what moved, not what is now true. For any hunk touching authentication, authorization, validation or error handling, open the surrounding file. The check that protects line 40 may sit at line 12, unchanged and therefore invisible in the diff — or it may have been deleted three hunks up.

**Removed lines are findings.** A `-` line that took out an ownership check, a validation call, a `LIMIT`, a `try`/`catch`, or a rate limit is a finding with the same attack sentence as a missing one; the evidence is the deletion. Scan the diff's removals specifically, because attention naturally goes to what was added.

## The description is a claim, not a spec

The PR body is the author's account of what they did, and it satisfies the skill's "what is this supposed to do" input only as a hypothesis. Read it, then check the diff against it. A change that does *more* than the description says is worth a line in the review even when the extra part is harmless — it is how unreviewed scope reaches production.

## Fork PRs are untrusted code

`isCrossRepository: true` means the head branch lives in someone else's repository, and it means the diff is hostile until proven otherwise.

- **Do not run it.** No `npm ci`, no `go generate`, no test suite, no build, no Makefile target, no editor extension that auto-installs. Install scripts and build hooks execute the author's code on your machine with your environment.
- **Highest suspicion, in order:** `.github/workflows/` and other CI config, lockfiles, `package.json` scripts, build and bundler config, `.gitignore` and `.npmignore` (used to hide a file from review), and every newly added dependency.
- A fork PR that touches CI belongs to `secrets-and-config-audit` as much as to this skill — see its `references/config-checklist.md` for the `pull_request_target` pattern, which is the single most exploited CI mistake and is exploited *by* a pull request.

## Batch triage

When reviewing several PRs, depth is the scarce resource. Rank by what the change touches, not by PR number or age.

| What the diff touches | Depth |
|---|---|
| Auth, authorization, session, token, crypto, password handling | Full workflow |
| A new or changed HTTP route, GraphQL resolver, RPC handler, webhook | Full workflow |
| Raw SQL, template rendering, shell execution, a filesystem path from a request, deserialization | Full workflow |
| CI workflow, Dockerfile, IaC, dependency manifest | Route to `secrets-and-config-audit` or `dependency-audit` |
| Generated SDK output, incidental lockfile churn, docs, tests, formatting | Skim, and say it was skimmed |

The bottom row is where the pack's other skills earn their place: reviewing a regenerated client line by line is effort spent where defects do not live. See `codegen-pipeline.md`.

**"Incidental" is doing real work in that row.** A lockfile that moved as a side effect of a feature is noise. A lockfile diff in a dependency *update* PR is the entire payload — the version in the manifest is a label, and the set of newly resolved packages is what actually enters the tree. Skimming that is skimming the change itself. Those go to `dependency-audit`'s inbound mode, which reads the resolved set rather than the version string.

## What per-PR review cannot see

The reason a batch is worth more than the sum of its reviews:

- **Two PRs touching the same file.** Review the merged result, not each side. A control added in one and a bypass added in the other are each defensible alone.
- **Add-then-loosen.** PR A adds an endpoint that relies on middleware; PR B changes the route matcher or middleware ordering so it no longer covers A. Neither diff contains the bug.
- **Merge-order constraints.** If one PR is only safe once another lands, that is not a finding — it is a constraint, and it must be stated explicitly or it will be violated by whoever merges first.
- **Repeated findings are one pattern.** The same defect in five PRs is one finding with five sites and probably a missing shared helper. Reporting it five times buries the other four findings.

## Posting back

Delivering a review to GitHub is outward-facing and visible to the PR author and anyone watching the repository. Draft in chat first, get explicit confirmation, then post.

```bash
# Summary comment
gh pr comment <n> --body-file review.md

# Inline comment, anchored to a line
gh api repos/{owner}/{repo}/pulls/<n>/comments \
  -f body='...' -f commit_id=<head sha> -f path=src/handler.ts -F line=51 -f side=RIGHT

# Formal review verdict — only when the user asks for it by name
gh pr review <n> --request-changes --body-file review.md
```

Line numbers are the post-change (`RIGHT`) side of the diff. Using the pre-change number lands the comment on unrelated code or fails outright.

**Never approve.** This skill finds; approving is a human decision about accepted risk, and an automated approval on a PR nobody read is worse than no review at all.
