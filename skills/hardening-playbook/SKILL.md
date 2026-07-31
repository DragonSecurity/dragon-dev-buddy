---
name: hardening-playbook
description: Closes the gap between code that works and code that is defensible, in priority order. Produces a ranked hardening plan across the app, its runtime, its pipeline and its dependencies, then applies the changes. Use when someone says "harden this", "security best practices", "lock this down", "prod-ready checklist", "what should I do before launch", "improve our security posture", or "we passed the audit, now what". Ranked by risk reduction per unit of effort, not a checklist to grind top to bottom.
---

# Hardening Playbook

Hardening is everything that is not a specific bug: the defense-in-depth, the secure defaults, the controls that would have contained a breach you have not had yet. The failure mode is a 200-item checklist that gets abandoned at item 30, having done the easy 30 and none of the important ones. This skill ranks by risk reduction per unit of effort and does the high-leverage work first, so stopping early still leaves you meaningfully safer.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `project.runtime`, `project.deploy_target`, `security.exposure`, `security.data_sensitivity`, `security.compliance`, `security.auth_model`, `practice.ci`, `practice.sca_tool`, `output.reports_dir`.

## Inputs

Ask only for what is missing:
- **The trigger.** Pre-launch, post-audit, post-incident, or general unease. Each front-loads different areas.
- **What is already in place**, so the plan does not recommend what exists.
- **Constraints.** What cannot change — a runtime you are stuck with, a dependency you cannot drop, a deploy process you do not own.
- **Effort appetite.** An afternoon, a sprint, or ongoing.

## Workflow

1. **Assess across the four layers.** Walk each, recording current state versus defensible state. Load `references/hardening-layers.md` for the per-layer checklist.
   - **Application** — authn, authz, input handling, output encoding, session, crypto, secrets handling, error and log hygiene.
   - **Runtime** — container/host user, network exposure, resource limits, filesystem permissions, the platform's own security features.
   - **Pipeline** — CI trust model, dependency controls, artifact integrity, deploy credentials, branch protection.
   - **Data** — encryption at rest and in transit, access scope, backup security, retention.

2. **Rank by risk reduction per effort.** For each gap, estimate how much risk closing it removes (using `security.exposure` and `data_sensitivity`) against how much effort it takes. A one-line change that closes an internet-facing hole beats a week-long project that hardens something already behind three other controls. Order the whole plan this way. `references/hardening-layers.md` has the scoring approach.

3. **Separate the structural from the additive.** Some hardening removes a class of bug (row-level security, an allowlist-based query layer, OIDC instead of static deploy keys). Some adds a layer that helps only if something else already failed (a CSP, a WAF rule). Structural changes rank higher at equal effort because they prevent rather than mitigate. Say which each is.

4. **Present the ranked plan before doing the work.** The user sees the order and the reasoning and can veto, reprioritize, or cap it at "just the top five." Hardening is where scope creep lives; the ranked plan is the control on it.

5. **Apply, highest leverage first.** Make the changes in priority order, each verified — the control works, and nothing legitimate broke. Test after each; a hardening change that breaks a real workflow gets reverted and noted, not forced through. Stop wherever the user's appetite runs out, having done the most valuable work, not the easiest.

6. **Leave the rest as a ranked backlog.** Everything not done stays in the plan, ordered, so the next session or the next person continues from where the value is highest rather than restarting the assessment.

7. **Wire in what makes hardening stick.** A CI check that fails on a reintroduced weakness is worth more than fixing it once by hand. Where a gap can be guarded automatically — a linter rule, a CI gate, a policy check — set that up so the improvement does not decay.

## Output format

```markdown
# Hardening plan: [project]
[date] · exposure [level] · data [sensitivity] · trigger [reason]

## Ranked plan
| # | Gap | Layer | Risk reduced | Effort | Type | Status |
|   | [gap] | app/runtime/pipeline/data | H/M/L | S/M/L | structural/additive | done/backlog |

## Applied
### [gap] — [what changed]
[the change, verified — control works, nothing legitimate broke]

## Backlog (ranked, not yet done)
[remaining items in priority order, so the next pass starts at the top]

## Made durable
[the CI checks / lint rules / policies wired in so this doesn't decay]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Hardened <project>: applied top <n> of <n> gaps, worst closed was <one clause>."`
- `kind`: `config`
- `skills_used`: `["dragon-dev-buddy:hardening-playbook"]`

Relay the reaction verbatim.

## File output

The plan to `output.reports_dir` as `YYYY-MM-DD-hardening-plan.md`, kept current as items move from backlog to applied. Config, IaC, CI and code changes go into the codebase. This skill modifies source and infrastructure files.

## Reference library

Load these for depth when the task calls for it:
- `references/hardening-layers.md`: the per-layer checklist (application, runtime, pipeline, data), the risk-per-effort scoring method, the structural-versus-additive distinction with examples, and the durability mechanisms per ecosystem.

## Worked example

See `examples/hardening-playbook-run.md` for a pre-launch hardening pass. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- The plan is ranked by risk reduction per effort, not presented as a flat checklist.
- The ranking reasoning is visible, so the user can disagree with a specific item's placement.
- Structural fixes are distinguished from additive ones, and the reason structural ranks higher is stated.
- The ranked plan was shown before the work, giving the user control over scope.
- Applied changes are verified — the control works and legitimate use still works.
- Stopping early left the highest-value work done, and the backlog is ranked for the next pass.
- At least one durability mechanism was wired in, so the hardening does not silently decay.
