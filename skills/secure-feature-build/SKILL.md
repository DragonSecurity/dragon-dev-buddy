---
name: secure-feature-build
description: Takes a feature from idea to shipped code with the abuse cases written before the implementation. Produces a spec that includes how the feature will be attacked, then builds it with those defenses in place from the first commit. Use when someone says "build a feature that", "add X to the app", "implement login/upload/payments/sharing", "I need a spec for", "build this the right way", or is about to add anything that handles user input, money, files, or access. Security is in the spec, not bolted on after review.
---

# Secure Feature Build

The cheapest time to defend a feature is before it exists. A bug caught in the spec costs a sentence; the same bug caught in review costs a rewrite; caught in production it costs an incident. This skill front-loads the defense: the abuse cases are written alongside the requirements, and the implementation has them built in rather than added after `secure-code-review` finds their absence.

This is the build-time complement to `threat-model`. Threat modelling maps an existing system; this shapes a new feature as it is designed.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `project.stack`, `project.primary_language`, `security.exposure`, `security.data_sensitivity`, `security.auth_model`, `security.trust_boundaries`, `practice.test_command`, `skill_level`.

## Inputs

Ask only for what is missing:
- **What the feature does**, in the user's words. One or two sentences.
- **Who uses it** and at what privilege. Anonymous, authenticated, admin, service.
- **What it touches.** User input, money, files, personal data, other users' data, external services.
- **Where it fits.** The existing code it extends or the boundary it sits on.

Do not ask for a full spec. Draft one and confirm it.

## Workflow

1. **Write the requirement in one paragraph.** What the feature does when everything goes right. Confirm this with the user before going further; building the wrong feature securely is still building the wrong feature.

2. **Enumerate the abuse cases.** For each thing the feature touches, ask how it is misused. Load `references/abuse-case-library.md` for the standard set per feature type — auth, upload, payment, sharing, search, admin action. Write each as an attacker sentence. This is the step that makes the feature secure; do not shorten it.

3. **Turn abuse cases into requirements.** Each abuse case becomes a defense the spec requires, phrased as testable behavior. "An attacker cannot upload an executable disguised as an image" becomes "the upload handler validates content by magic bytes, not extension or client content-type, and rejects anything not in the allowlist." The defense is now a requirement, not an afterthought.

4. **Decide the data and trust shape.** Where does the data live, what is the minimum privilege the feature needs, and where is the trust boundary. Prefer the design that makes the abuse case structurally impossible over the one that checks for it. A tenant column that every query must remember is worse than a row-level policy the database enforces.

5. **Write the spec.** Requirement, abuse cases, defenses-as-requirements, data model, trust boundary, and the acceptance tests — including the negative ones. This is the artifact; the code implements it.

6. **Build it.** Implement to the spec. The defenses go in as the feature is written, not after. Every abuse case from step 2 is either handled in the code or explicitly deferred with a reason. Match the codebase's existing patterns and style.

7. **Write the tests alongside.** The positive path and the negative path. Every abuse case that the spec said is defended gets a test that performs the attack and asserts it fails. Hand these to `security-test-writer` if they need depth, but the feature is not done until the abuse cases are red-then-green.

8. **Self-review against the spec.** Before declaring done, walk the abuse case list one more time against the actual code. Anything unhandled is either fixed or moved to a stated follow-up. Then recommend `secure-code-review` for an adversarial second pass — the author is the worst reviewer of their own defenses.

## Output format

First the spec, then the implementation:

```markdown
# Feature spec: [name]

## Requirement
[one paragraph, happy path]

## Abuse cases
- A1: [attacker sentence] → defended by [R#]
- A2: ...

## Requirements (including defenses)
- R1: [testable behavior]
- R2: [defense phrased as behavior] (closes A1)

## Data and trust
[what's stored, minimum privilege, boundary]

## Acceptance tests
- Positive: [...]
- Negative: [...one per defended abuse case]
```

Then the code, then:

```markdown
## Build notes
Handled: [abuse cases closed in code]
Deferred: [abuse case] — [why, and the follow-up]
Next: secure-code-review on [files]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Built <feature> with <n> abuse cases defended from the spec, <n> negative tests."`
- `kind`: `feature`
- `skills_used`: `["dragon-dev-buddy:secure-feature-build"]`

Relay the reaction verbatim.

## File output

The spec to `output.reports_dir` as `YYYY-MM-DD-spec-<feature>.md`, and the implementation into the codebase where it belongs. Tests alongside the code. Unlike the audit skills, this one writes source — it is a build skill.

## Reference library

Load these for depth when the task calls for it:
- `references/abuse-case-library.md`: the standard abuse cases per feature type, the defense that closes each, and the "make it structurally impossible" alternatives.

## Worked example

See `examples/secure-feature-build-run.md` for a file-sharing feature built from abuse cases. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- Abuse cases were written before the implementation, and each maps to a requirement that closes it.
- Defenses are phrased as testable behavior, not as "validate input."
- The design prefers structural impossibility over runtime checks where it can.
- Every defended abuse case has a negative test that performs the attack.
- The build notes state what was deferred and why, rather than implying full coverage.
- The feature does what the user actually asked for. Secure and wrong is still wrong.
