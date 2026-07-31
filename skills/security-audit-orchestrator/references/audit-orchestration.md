# Audit orchestration reference

## The dependency map

```
1 orient
     │
     ├──> 2 threat-model ────────────┐
     │        produces targeting     │
     │                               ▼
     ├──> 3 secrets-and-config ──> [STOP CONDITION]
     │        may abort the chain    │
     │                               ▼
     ├──> 4 dependency-audit ───> 5 secure-code-review
     │        removes false leads         │
     │                                    ▼
     └────────────────────────────> 6 consolidate ──> 7 deliverables ──> 8 next steps
```

## Why this order

**Threat model before code review.** A review without targeting reads files in whatever order they appear and finds whatever is on the surface. A review pointed at "the three places tenancy is enforced" finds the bug. The single largest quality difference between a good audit and a bad one is whether stage 5 knew where to look.

**Secrets before everything slow.** A live credential is not a finding, it is an incident. Discovering it in hour four means it was live for four more hours than it needed to be. This stage is cheap and fast; run it early even though it feels out of order.

**Dependencies before code review.** Two reasons. First, a known-vulnerable library in a hot path reframes what the hand-written code around it is worth reviewing. Second, SCA output is mostly noise, and triaging it early tells you how much of the remaining time you actually have.

**Consolidate before writing.** Findings arrive in four different vocabularies from four different stages. Merging them last means the report has one voice and one ranking, rather than four appendices.

## Stop conditions

Abort the chain and change skills when any of these appear. Say why, out loud, rather than continuing and mentioning it later.

| Condition | Switch to | Why |
| --- | --- | --- |
| A credential that is live in a real system | `incident-response` | Rotation and blast-radius assessment outrank the rest of the audit. |
| Evidence of prior compromise: unexplained accounts, unfamiliar scheduled tasks, log gaps | `incident-response` | You are no longer auditing, you are responding. |
| A Critical finding with an active public exploit against an exposed surface | `vuln-triage` then patch | Finish the fix, then resume. |
| Scope turns out to be 10× larger than stated | back to the user | Re-scope rather than producing shallow coverage across everything. |

## Consolidation rules

**De-duplicate by defect, not by description.** A stage 2 threat ("cross-tenant read on reports") and a stage 5 finding ("`findUnique` with no tenant predicate at `reports.ts:44`") are one item. Keep the code-level evidence, keep the model's risk reasoning, merge into one entry that has both.

**Re-rank across the whole set.** Severity assigned inside a stage is relative to that stage. An outdated crypto library rated Medium by the dependency pass may outrank a High from the review once you can see that it sits on the unauthenticated path the model flagged. Re-rank once, at the end, with everything visible.

**Promote chains.** Two Mediums that compose into a full compromise are one Critical. Say which two, and how they chain. Auditors and engineers both consistently miss these because the stages that found them ran separately.

**Demote what is unreachable.** A vulnerable code path that nothing calls is a maintenance issue, not a security finding. Move it to a "worth cleaning up" section rather than inflating the count. An audit judged on finding count is an audit optimizing for the wrong thing.

## The coverage statement

Non-negotiable, and the part most likely to be quietly dropped. Three lists:

- **Reviewed** — read by a human (or by you, carefully) with intent. Name the directories or modules.
- **Not reviewed** — in scope but not reached, with the reason: time, size, or a deliberate decision.
- **Tool-assisted only** — a scanner ran, nobody read the output line by line. This is a real coverage level and pretending otherwise is how findings get missed.

Without this, a reader assumes complete coverage. That assumption is the mechanism by which audits cause harm.

## Effort estimates for the backlog

Coarse is fine; precision is not the point. The purpose is to let someone sequence the work.

| Band | Means |
| --- | --- |
| **S** | One file, one change, existing test covers it. Under an hour. |
| **M** | Several files or a new test. Half a day. |
| **L** | Structural: a new abstraction, a migration, a dependency swap. Multiple days. |
| **XL** | Needs a design decision before it can be estimated. Say what the decision is. |

A Critical rated XL should be flagged explicitly in the summary, because it means the risk will be live for a while and someone senior needs to know that now rather than in three weeks.

## What the orchestrator must not do

- Do not run all stages silently and present everything at the end. The checkpoints are the feature.
- Do not review the whole codebase because the scope said "the repo." Review what the model pointed at and state the coverage honestly.
- Do not pad the finding count with style issues, missing comments, or unreachable code. Every low-value finding costs the reader attention that the Critical needed.
- Do not produce a report and a backlog that disagree. They are two views of one list.
