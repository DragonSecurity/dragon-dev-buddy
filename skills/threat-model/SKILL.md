---
name: threat-model
description: Builds a threat model for a system, service or feature: data flows, trust boundaries, STRIDE analysis, ranked threats, and mitigations that point at real files. Use when someone says "threat model this", "what could go wrong here", "what's the attack surface", "STRIDE", "security design review", "we need a threat model for the audit", "how would someone attack this", or before building anything that touches money, credentials or personal data. Produces a document you can hand to an auditor and a backlog you can actually work.
---

# Threat Model

Most security work is spent auditing whatever is easiest to audit. A threat model is how you find out what is worth auditing. It maps what the system does, where trust changes hands, and what an attacker gets for breaking each part, then ranks the results so the first thing you fix is the first thing worth fixing.

This is a design-level skill. It does not read every line of code. It reads enough structure to draw the boundaries correctly, then reasons about what crosses them.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `project.what_it_is`, `project.stack`, `project.runtime`, `security.exposure`, `security.data_sensitivity`, `security.auth_model`, `security.trust_boundaries`, `security.compliance`, `output.reports_dir`.

If `security.trust_boundaries` is populated, start from it rather than deriving from scratch, and say so. If it is empty, derive boundaries in step 2 and offer to write them back to config at the end.

## Inputs

Ask only for what is missing:
- **Scope.** The whole system, one service, or one feature? A model of everything is a model of nothing; push for a bounded scope.
- **What an attacker wants here.** Money, data, compute, disruption, or a foothold into something else. If the user does not know, derive it from `data_sensitivity` and confirm.
- **Anything already known to be weak.** Saves you rediscovering it.

Do not ask for a diagram. Build one from the code.

## Workflow

1. **Map the system.** Identify the actors (human and machine), the processes, the data stores, and the flows between them. Read entry points, route definitions, service clients and schema. Keep it to the level of detail where every element could plausibly be attacked separately. Ten elements is a useful model; sixty is a diagram nobody will read.

2. **Draw the trust boundaries.** A boundary is any place where the level of trust in the data or the caller changes: browser to API, API to database, service to third party, tenant to tenant, unauthenticated to authenticated, user to admin. Render the result as a Mermaid diagram with boundaries as subgraphs. Every boundary you draw is a place you will enumerate threats.

3. **Enumerate threats per element, using STRIDE.** For each element that touches a boundary, walk the six categories: Spoofing, Tampering, Repudiation, Information disclosure, Denial of service, Elevation of privilege. Not every category applies to every element; say so rather than inventing filler. Use the prompts in `references/stride-prompts.md` to avoid the generic-threat trap.

   Write each threat as an attacker sentence with a concrete precondition and a concrete outcome: "An authenticated tenant A user calls `GET /reports/:id` with tenant B's report id and receives B's financial data, because the handler filters by id but not by tenant." A threat you cannot write that way is not yet a threat, it is a category.

4. **Rank by risk.** Score each threat on likelihood and impact using the rubric in `references/risk-ranking.md`. Impact is scaled by `security.data_sensitivity`; likelihood is scaled by `security.exposure`. State the score and the reason for it. A ranking with no visible reasoning is a ranking nobody will trust or revisit.

5. **Assign a response to each threat.** One of: mitigate, transfer, accept, eliminate. Most will be mitigate. Naming an accepted risk explicitly, with who accepted it, is the part teams skip and auditors ask about.

6. **Write mitigations that point at code.** Each mitigation names the file or module it lands in, and the shape of the change. "Add authorization" is not a mitigation. "In `src/routes/reports.ts`, scope the Prisma query by `session.tenantId` and add a repository-level guard so a future handler cannot skip it" is a mitigation.

7. **Produce the backlog.** Convert every `mitigate` into a work item with a title, the threat it closes, the file, and the ranking. This is the output the team will actually use; the document is what the auditor will read.

8. **Name what you did not model.** Scope boundaries, components you could not see, assumptions you made about infrastructure. An honest gap list is worth more than false completeness, and it tells the next person where to start.

## Output format

Write the report to `output.reports_dir`:

```markdown
# Threat model: [scope]
[date] · [project] · exposure [level] · data [sensitivity]

## Scope
In: [...]   Out: [...]   Assumed: [...]

## System and trust boundaries
[mermaid diagram]
[one paragraph per boundary: what changes hands and why it matters]

## Threats
### T1 — [short title]   `[element]` · STRIDE: [letter]
**Attack:** [attacker sentence with precondition and outcome]
**Likelihood:** [H/M/L] — [reason]   **Impact:** [H/M/L] — [reason]   **Risk:** [score]
**Response:** [mitigate | transfer | accept | eliminate]
**Mitigation:** [concrete change, with file]

## Backlog
| # | Threat | Risk | Change | File |

## Not modelled
- [gap] — [why, and what would close it]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Threat modelled <scope>: <n> threats, <n> high risk, top one is <one clause>."`
- `kind`: `docs`
- `skills_used`: `["dragon-dev-buddy:threat-model"]`

Relay the reaction verbatim. If the server is not connected, skip silently.

## File output

One markdown report in `output.reports_dir`, named `YYYY-MM-DD-threat-model-<scope>.md`. Offer to write derived trust boundaries back into `.dragon-buddy/config.json` so later skills inherit them. No source files are modified.

## Reference library

Load these for depth when the task calls for it:
- `references/stride-prompts.md`: per-category question prompts by element type, plus the threat patterns that recur in web, API, multi-tenant and queue-based systems.
- `references/risk-ranking.md`: the likelihood and impact rubric, how exposure and data sensitivity scale each, and worked scoring examples.

## Worked example

See `examples/threat-model-run.md` for a complete model of a multi-tenant reporting API. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- Scope is bounded and the out-of-scope list is non-empty. A model of the whole company is not a model.
- The diagram shows trust boundaries as boundaries, not just boxes and arrows.
- Every threat is an attacker sentence with a precondition and an outcome. No entries of the form "SQL injection risk."
- Every risk score carries its reasoning inline. A reader can disagree with a specific score without discarding the model.
- Every mitigation names a file or module. Zero instances of "add validation" with no location.
- Accepted risks are labelled as accepted, not quietly omitted.
- The "Not modelled" section is populated. Every model has gaps; hiding them is the failure mode.
