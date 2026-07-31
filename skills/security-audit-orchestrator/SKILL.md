---
name: security-audit-orchestrator
description: Runs a full security audit in one pass by chaining this pack's skills in dependency order: threat model, secrets and config, dependencies, targeted code review, then a consolidated report and backlog. Use when someone says "full security audit", "audit this codebase", "run the whole security pass", "we have a pentest next month", "end-to-end security review", "check everything", or wants the complete picture rather than one skill at a time. This is the master orchestrator of the Dragon Dev Buddy pack.
---

# Security Audit Orchestrator

Running one skill at a time is right when you know what you are looking for. This is for when you do not: a new codebase, an audit deadline, a due-diligence request, or the uneasy feeling that you have never actually looked.

It chains the pack in dependency order, because the order matters. Threat modelling first means the code review knows where to look. Secret scanning early means rotation starts while the slower work runs. Dependency triage before code review means you do not spend an hour on a hand-written parser that a known-vulnerable library was going to bypass anyway.

You get a checkpoint after each stage. Redirect there; do not wait for the end.

## First-run check

Read `.dragon-buddy/config.json`. If missing or `security.exposure` is unset, run `buddy-setup` first — this orchestrator depends on a complete profile and every severity it produces is wrong without one. Pull all keys.

If `security.known_risk_areas` is populated, those areas jump the queue in stages 3 and 4.

## Inputs

Ask only for what is missing:
- **Scope.** Whole repo, one service, or one subsystem. Whole-repo is viable up to roughly 50k lines; past that, push for a subsystem and say why.
- **Deadline and depth.** A pentest in a month justifies the full chain. "Something feels wrong" may only justify stages 2 and 4.
- **What must not be touched.** Files, environments, anything under active migration.
- **Whether findings can be written to the repo** or must stay in the report only.

## Workflow

Run each stage, then stop and summarize before starting the next. Never run the whole chain silently and present a wall of findings at the end.

1. **Orient.** Survey the repo: size, entry points, where input enters, where privilege is decided, what talks to the outside world. Produce a one-paragraph statement of what this system is and what an attacker would want from it. Confirm scope with the user. This is five minutes and it prevents the audit from being shaped by whatever file you happened to open first.

2. **Threat model** (`threat-model`). Run it over the confirmed scope. The ranked threat list becomes the targeting data for stages 4 and 5. **Checkpoint:** present the boundaries and the top five threats. Ask whether anything is missing, and whether the ranking matches the user's intuition. A user who disagrees with the ranking usually knows something the code does not show.

3. **Secrets and configuration** (`secrets-and-config-audit`). Run early and in parallel with nothing else, because a live credential changes the shape of the whole engagement. **Checkpoint:** if anything live is found, stop the audit. Rotation is now the only priority, and `incident-response` may be the correct next skill rather than stage 4. Say this plainly rather than logging it as finding number seven.

4. **Dependencies** (`dependency-audit`). Triage what the SCA tool says against what is actually reachable. **Checkpoint:** report the count that is real versus the count the tool claimed, and the upgrade plan. This is usually where the biggest gap between "scanner output" and "risk" appears.

5. **Targeted code review** (`secure-code-review`). Do not review everything. Review, in order: the files named in stage 2's top threats, the `known_risk_areas` from config, every place authorization is decided, and every entry point that takes untrusted input. **Checkpoint:** findings by severity, with the coverage statement — what you reviewed and what you did not.

6. **Consolidate.** Merge findings from stages 2 through 5 into one ranked list. De-duplicate: a threat from the model and a finding from the review that describe the same defect are one item, not two. Re-rank across the whole set, because a Medium from the dependency pass may outrank a High from the review once you can see both.

7. **Produce the deliverables.** Two artifacts:
   - **The report** — for a reader who was not here. Executive summary, method, scope, findings, what was not covered.
   - **The backlog** — for the team. One row per fix, ranked, each naming a file and a change. If the user permits repo writes, offer to open issues or write it to the repo.

   If this is a client engagement with a recorded `engagement.authorization_ref`, hand the report to `pentest-report` for formatting instead of writing it here.

8. **Recommend the next two skills.** After an audit, the highest-value follow-ups are usually `security-test-writer` for the confirmed findings and `hardening-playbook` for the structural gaps. Name the specific findings each would take.

## Depth control

If the user wants less than the full chain:

| Situation | Stages to run |
| --- | --- |
| Pentest or audit incoming | all of 1–8 |
| New codebase, getting oriented | 1, 2, 5 |
| "Something feels wrong" | 1, 3, 5 |
| Pre-release check on a known codebase | 3, 4, 5 |
| Compliance evidence needed | 1, 2, 7 |

State which stages you are running and which you are skipping, before you start.

## Output format

Per checkpoint:

```
Stage [n]: [name] — done
[3-5 lines of what was found]
Next: stage [n+1], [name]. Redirect now if this is going the wrong way.
```

Final:

```markdown
# Security audit: [scope]
[date] · [project] · exposure [level] · data [sensitivity]

## Summary
[what was audited, how, and the honest headline in 3 sentences]

## Findings
| # | Finding | Severity | Source stage | File |

## [one section per finding: attack, evidence, impact, fix]

## Backlog
| # | Change | File | Closes | Effort |

## Coverage
Reviewed: [...]   Not reviewed: [...]   Tool-assisted only: [...]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Call `buddy_observe` once, at the end of the whole chain — not once per stage:
- `summary`: `"Audited <scope>: <n> findings, <n> critical, top one is <one clause>."`
- `kind`: `other`
- `skills_used`: every stage skill that actually ran, e.g. `["dragon-dev-buddy:security-audit-orchestrator", "dragon-dev-buddy:threat-model", "dragon-dev-buddy:secrets-and-config-audit", "dragon-dev-buddy:dependency-audit", "dragon-dev-buddy:secure-code-review"]`

Listing the sub-skills is the point: it is how the buddy learns which skills co-occur. Relay the reaction verbatim.

## File output

One report and one backlog in `output.reports_dir`, named `YYYY-MM-DD-audit-<scope>.md` and `YYYY-MM-DD-audit-<scope>-backlog.md`. Stage skills' individual reports are folded into the consolidated one rather than written separately, unless the user asks for both. No source files are modified.

## Reference library

Load these for depth when the task calls for it:
- `references/audit-orchestration.md`: the dependency map and why the order is what it is, the consolidation and de-duplication rules, the coverage statement template, and the stop conditions that abort the chain.

## Worked example

See `examples/security-audit-orchestrator-run.md` for a full chain with checkpoints. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- Every stage is summarized before the next begins. The user had a real chance to redirect after the threat model.
- Stage 3 aborts the chain on a live credential rather than filing it as a finding.
- Stage 5 reviews what stage 2 pointed at, not whatever was alphabetically first. The link between the two is visible in the report.
- Consolidation actually de-duplicates. The same defect never appears twice under two names.
- The coverage statement names what was **not** reviewed. An audit that implies full coverage it did not achieve is worse than no audit.
- The backlog names a file and a change per row. It can be worked without reading the report.
- One buddy observation for the whole chain, listing every sub-skill that ran.
