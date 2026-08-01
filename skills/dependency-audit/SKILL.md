---
name: dependency-audit
description: Triages dependency and supply-chain risk: which advisories are actually reachable, which are noise, and a staged upgrade plan that will not break the build. Use when someone says "npm audit says", "check my dependencies", "we have 200 vulnerabilities", "dependency audit", "is this package safe", "supply chain review", "should I upgrade X", or when a scanner opens a pile of pull requests nobody understands. Turns scanner output into a short list of things that matter.
---

# Dependency Audit

Scanner output is not a finding list. It is a list of advisories that exist somewhere in a dependency tree, most of which describe code paths your application never executes. Teams respond by either upgrading everything, which breaks things, or ignoring all of it, which eventually does not work out.

This skill does the part in between: work out which advisories are reachable from your code, rank those, and produce an upgrade plan ordered by risk rather than by whatever the tool listed first.

It also looks at the supply chain itself, which scanners do not: unmaintained packages, install scripts, typosquats, and whether your lockfile means anything.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `project.primary_language`, `project.stack`, `project.runtime`, `security.exposure`, `security.data_sensitivity`, `practice.sca_tool`, `practice.test_command`, `practice.ci`, `output.reports_dir`.

## Inputs

Ask only for what is missing:
- **Existing scanner output**, if they have it. Otherwise run the ecosystem's native tool.
- **Appetite for breaking changes.** "Patch only until the release" and "clean it all up" produce different plans.
- **Anything pinned deliberately.** A version held back for a reason you do not know about will look like negligence.
- **Whether there is already a queue of automated update PRs.** If so, run the inbound mode below instead of opening more.

## Workflow

1. **Establish the inventory.** Read the manifest and the lockfile. Record: direct dependency count, total resolved count, whether a lockfile exists and is committed, and whether the manifest uses ranges that let CI resolve differently from a developer's machine. A missing or uncommitted lockfile is a finding in itself and it invalidates everything else in this report.

2. **Run the scanner.** Use `practice.sca_tool` if set, otherwise the ecosystem default (`npm audit --json`, `pip-audit`, `govulncheck`, `cargo audit`, `bundle audit`, `mvn dependency-check`). Capture the raw count. Prefer `govulncheck` and similar reachability-aware tools where available, because they do step 3 for you.

3. **Triage for reachability.** This is the work. For each advisory, determine which of these it is:

   | Verdict | Means |
   | --- | --- |
   | **Reachable** | Your code calls the affected function, or a dependency you call does, on a path that can see untrusted input. |
   | **Present but unreachable** | The package is installed, the vulnerable code path is never executed. |
   | **Dev-only** | Build tooling, test runner, linter. Never processes untrusted input at runtime. Matters if CI runs untrusted PRs. |
   | **Not applicable** | The advisory describes a configuration or platform you do not use. |

   Say how you determined reachability. Grep for the affected import and call, check whether the calling path takes external input, or state honestly that you could not determine it and defaulted to treating it as reachable.

4. **Rank what is reachable.** Severity is the advisory's rating adjusted by *your* context: where in the request path it sits, whether the input reaching it is attacker-controlled, and the project's exposure and data sensitivity. A Critical advisory in a code path nothing reaches outranks nothing. A Medium on the authentication path outranks a Critical in a build script.

5. **Check the supply chain, not just the advisories.** Load `references/supply-chain.md` and work the list: packages with no release in two years, packages with a single maintainer and high download counts, install scripts (`postinstall`), recently transferred ownership, names one character from a popular package, and dependencies added in the last 90 days that nobody remembers approving.

   If the codebase generates code from a spec — openapi-generator, `@hey-api/openapi-ts`, a generated Terraform provider — also load `references/codegen-pipeline.md`. The generators, their pinned versions, and the provenance of the vendored generated code are a supply-chain surface a manifest scan misses entirely.

6. **Build the upgrade plan in stages.** Order by risk, then by blast radius:
   - **Stage 1 — patch and minor, no API change.** Batchable, one PR, verified by the existing test suite.
   - **Stage 2 — major upgrades on reachable advisories.** One per PR, each with the specific breaking changes named and the migration steps listed.
   - **Stage 3 — everything else.** Scheduled, not urgent, explicitly deferred.

   For each entry name the current version, the target, and what will break. "Upgrade to latest" is not a plan.

7. **State what does not get fixed.** Advisories with no patched version, packages that are abandoned, transitive dependencies you cannot move without the parent moving first. For each, give the compensating control: a runtime guard, a network restriction, an input constraint, or a decision to accept and revisit.

8. **Close the loop.** If `practice.sca_tool` is unset, recommend one and the CI wiring for it. If it is set but the noise is the reason nobody reads it, recommend the configuration change that would make the output actionable.

## Inbound mode: a queue of update PRs that already exists

The main workflow runs outward — read the tree, find what matters, propose PRs. Sometimes the PRs are already there and the question is the reverse: a bot has opened some number of them and nobody knows which to merge. Run this instead of steps 1–2, then rejoin at step 3, because the triage is the same work in the other direction.

The queue is not the unit of work. Risk is. Never process these in the order the bot opened them.

1. **List the queue with its state, not just its titles.** For each PR: what changed, direct or transitive, patch/minor/major, whether CI passed, and how old it is. On GitHub, `gh pr list --json number,title,headRefName,createdAt,statusCheckRollup` and the diff for each. A title is the bot's summary of its own change and is not evidence.

2. **If automerge is configured and the queue exists anyway, that is the finding.** An update bot with automerge is a control; a queue that has stopped draining means the control is broken, not that there is a backlog. Diagnose why before triaging a single PR — a required check that never runs, a failing test nobody looked at, rules the automerge conditions do not satisfy. Fixing that clears the queue and every future one. Triaging by hand clears this queue only.

3. **Separate security-driven from drift-driven.** Only the first is urgent:
   - **Security-driven** — closes an advisory. Verify the claim rather than trusting the label; run it through step 3 of the main workflow to establish whether the advisory is reachable here. A bot does not know your call graph.
   - **Drift-driven** — the version simply moved. This is hygiene. Valuable, not urgent, and safe to batch.

4. **Read the lockfile, not the version number.** A dependency PR is a supply-chain event: someone else's code entering yours. The meaningful diff is the set of newly resolved packages, not the number in the manifest. A patch bump that pulls in three new transitive packages deserves more attention than a major bump that pulls in none. Any *new* package name in the lockfile gets the `references/supply-chain.md` checks — age, maintainer count, install scripts, name similarity — because that is a dependency nobody chose.

   This is the deliberate exception to `secure-code-review`'s batch-triage table, which skims lockfile churn. That is right when a lockfile moved as a side effect of someone else's feature and there is nothing in it that anybody chose to add. It is exactly wrong here, where the resolved set is the whole of what the PR does.

5. **Establish that CI means something before treating it as evidence.** A green check on an update PR is only worth what the pipeline runs. If the workflow does not execute the test suite, or the tests do not exercise the upgraded package, green means "it installed". Say which of the two you have.

6. **Order the merges.** Batched patch and minor updates first: they are the least likely to break and merging them proves the pipeline is healthy before anything risky rides on it. Then majors, one at a time, each with its breaking changes named. A major merged alongside anything else makes the bisect ambiguous when something fails next week.

7. **Close, do not merge, the superseded.** A PR old enough that a later one supersedes it should be closed. Merging both replays history and can resurrect an intermediate version. Stale queue entries are also the reason the queue reads as noise.

## Output format

For inbound mode:

```markdown
# Update queue triage: [project]
[date] · [n] open PRs · [n] security-driven, [n] drift · automerge [on/off/broken]

## Control health
[If automerge is configured but the queue is not draining, why — this outranks the queue itself.]

## Merge now
| PR | Change | Why now | CI |

## Batch together
| PR | Change | Why safe to batch |

## Hold
| PR | Change | What must happen first |

## Close
| PR | Superseded by / reason |

## New transitive packages entering the tree
| Package | From which PR | Age / maintainers / install scripts |
```

For a standard audit:

```markdown
# Dependency audit: [project]
[date] · [n] direct, [n] resolved · scanner reported [n], reachable [n]

## Inventory
Lockfile: [present/absent, committed?]   Manifest ranges: [pinned/caret/wildcard]

## Reachable
### D1 — [package]@[version] — [advisory id]   **[SEVERITY]**
**Advisory:** [what it is, one sentence]
**Why it's reachable here:** [the call path, with a file reference]
**Severity here:** [rating] — [reason, referencing exposure and the input that reaches it]
**Fix:** [current] → [target]. [What breaks, or "no API change"]

## Dismissed ([n])
| Package | Advisory | Verdict | Why |

## Supply chain
- [finding]

## Upgrade plan
**Stage 1** (one PR, existing tests cover it): [list]
**Stage 2** (one PR each): [list with migration notes]
**Stage 3** (scheduled): [list]

## No fix available
| Package | Advisory | Compensating control |
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Audited <n> dependencies: <n> of <n> advisories reachable, worst is <package> on <path>."`
- `kind`: `config`
- `skills_used`: `["dragon-dev-buddy:dependency-audit"]`

Relay the reaction verbatim.

## File output

One report in `output.reports_dir` as `YYYY-MM-DD-dependency-audit.md`. Do not run upgrades as part of the audit. Offer to execute stage 1 as a separate, explicit step, with the test suite run afterwards.

## Reference library

Load these for depth when the task calls for it:
- `references/supply-chain.md`: the reachability determination method per ecosystem, the supply-chain risk checklist, lockfile and pinning guidance, and how to read an advisory without over-trusting its severity rating.
- `references/codegen-pipeline.md`: the codegen toolchain as a supply-chain surface — pinning generators, provenance and reproducibility of vendored SDKs, reachability through generated code, and the Terraform provider as the highest-stakes generated artifact.

## Worked example

See `examples/dependency-audit-run.md` for a triage of a noisy `npm audit`. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- The headline is the reachable count against the reported count. That gap is the value this skill adds.
- In inbound mode: a stalled automerge was diagnosed as a broken control before the queue was triaged by hand, PRs were ordered by risk rather than by age, and every package newly entering the lockfile was named — a transitive addition nobody chose is the point of reading an update PR at all.
- Every reachable finding names the call path that makes it reachable, with a file reference.
- Every dismissal has a stated reason. "Probably fine" is not a verdict.
- Severity is re-rated for this project, not copied from the advisory.
- The upgrade plan names what breaks per major version. Nothing says "upgrade to latest."
- Supply-chain risks that no scanner reports were looked for and reported.
- Advisories with no fix have a compensating control, not silence.
- No upgrades were performed during the audit.
