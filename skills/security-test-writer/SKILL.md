---
name: security-test-writer
description: Turns a confirmed vulnerability or finding into an automated regression test that fails on the vulnerable code and passes once it is fixed. Use when someone says "write a test for this bug", "make sure this doesn't come back", "regression test for the finding", "add a security test", "prove the fix works", or after any finding from review or triage. A finding without a test comes back on the next refactor.
---

# Security Test Writer

A fix without a test is a fix with an expiry date. The next refactor, the next dependency bump, the next person who does not know why that check is there — any of them can quietly reopen it, and nothing will notice until someone external does.

This skill writes the test that notices. The discipline is the same as test-driven development, aimed at an attacker: the test must fail against the vulnerable code, for the right reason, and pass against the fix. A security test that passes against the vulnerable code is not testing the vulnerability.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `project.primary_language`, `project.stack`, `practice.test_command`, `output.reports_dir`.

Detect the test framework from the repo — the manifest, the existing `test/` directory, the config file — rather than asking. Match the style of the tests already there.

## Inputs

Ask only for what is missing:
- **The finding.** From `secure-code-review`, `vuln-triage`, a report, or a description. The attack sentence is what you are encoding.
- **The fix**, if it exists yet. You can write the test before the fix (it should fail now) or after (it should pass now, and you verify it would have failed before).
- **The affected code path**, so the test targets the real function, not a mock of it.

## Workflow

1. **Extract the security property.** Every finding asserts a property that should hold and does not. "A user cannot read another tenant's report." "A token signed with the wrong algorithm is rejected." "An upload path cannot escape the upload directory." Write it as a sentence. This is what the test asserts; everything else is setup.

2. **Find the real seam.** Test at the lowest layer where the property actually lives. An authorization property tested through the full HTTP stack is slow and flaky; the same property tested at the handler or the repository function is fast and precise. But do not test below the bug: if the vulnerability is in the routing layer, a unit test of the query function will pass while the app stays exploitable. Match the layer to where the defect is. Name the seam and confirm it before writing test code rather than after — `codebase-design` has the vocabulary if the right boundary is itself in question.

3. **Write the negative test — the attack.** The primary test performs the attack and asserts it fails. Tenant A authenticates and requests tenant B's object; assert 404 or 403, and assert the response body does not contain B's data. This test must **fail against the vulnerable code**. If it passes before the fix, it is asserting the wrong thing.

4. **Write the positive test — the legitimate path.** The control that blocks the attack must not block real use. Tenant A requests tenant A's own object; assert success. Without this, the "fix" of denying everyone would pass the suite. Security tests that only check the negative case incentivize breaking the feature.

5. **Write the boundary cases.** The variations that a naive fix misses: the empty value, the null, the encoded payload (`%2e%2e%2f` as well as `../`), the case-varied header, the almost-right token, the off-by-one on the boundary. Load `references/security-test-patterns.md` for the set that matters per vulnerability class.

6. **Prove the test discriminates, and that it could have disagreed.** State how you confirmed it fails on the vulnerable code and passes on the fix — if the fix is already in, revert it on a scratch branch and confirm the test goes red. A test never seen to fail is not yet a test. Then check the other half: where the expected value came from. If the assertion recomputes it the way the code does — signing the token with the verifier's own helper, comparing output to `sanitise(input)`, blessing a snapshot of current behaviour — it passes by construction and can never contradict the implementation, however cleanly it discriminated. Expected values come from an independent source: a hand-written literal, the exploit payload, the spec. `references/security-test-patterns.md` covers this and the other two structural anti-patterns.

7. **Make it durable.** No dependence on record ids that a seed reshuffle will change, no sleeps for timing, no reliance on test execution order. It runs in CI, on someone else's machine, in two years. Name it so a failure is self-explaining: `rejects_cross_tenant_report_read`, not `test_security_3`.

8. **Wire it in.** Put it where the suite will run it, in the style of the existing tests. If the project has a security-test grouping, use it; if not, the test lives next to the code it guards. Confirm `practice.test_command` picks it up.

## Output format

The test code, ready to commit, plus:

```markdown
## Security test: [property]
**Finding:** [the attack sentence this encodes]
**Property asserted:** [one sentence]
**Test layer:** [unit / integration / e2e] — [why this layer]
**Discrimination proven:** [how you confirmed red-on-vulnerable, green-on-fixed]

**Tests written:**
- `[name]` — the attack, must fail on vulnerable code
- `[name]` — the legitimate path, guards against over-correction
- `[name]`, `[name]` — boundary cases: [what each covers]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Wrote regression tests for <finding>: <n> cases, red on vulnerable and green on fixed."`
- `kind`: `test`
- `skills_used`: `["dragon-dev-buddy:security-test-writer"]`

Relay the reaction verbatim.

## File output

Test files in the project's test location, matching existing conventions. This skill writes tests, not fixes and not reports. If the fix is not yet in, say clearly that the test is expected to fail until it lands — a red test is the correct state, not a mistake.

## Reference library

Load these for depth when the task calls for it:
- `references/security-test-patterns.md`: the boundary-case set per vulnerability class, agreeing seams before writing, framework-specific patterns for auth and multi-tenant tests, how to test negative properties cleanly, how to prove a test discriminates, and the three structural anti-patterns — tautological, implementation-coupled, horizontal slicing.

## Worked example

See `examples/security-test-writer-run.md` for tests written for an IDOR fix. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- The primary test performs the actual attack and was confirmed to fail on the vulnerable code. This is stated, not assumed.
- A positive test guards the legitimate path, so "deny everyone" cannot pass the suite.
- Boundary cases cover the encodings and edge values a naive fix misses.
- The test is written at the layer where the defect lives, not above or below it.
- Discrimination is proven and the method is stated. No test that was never seen to fail.
- No assertion recomputes its expected value the way the code does. Expected values come from an independent source of truth, so the test could have disagreed with the implementation.
- Names are self-explaining. A CI failure tells you what broke without opening the file.
- The test is durable: no ordering, timing, or hardcoded-id dependence.
