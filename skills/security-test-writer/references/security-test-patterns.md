# Security test patterns

## The shape of every security test

Three parts, always:

1. **The attack** — does the malicious thing, asserts it is stopped. Must be red on vulnerable code.
2. **The legitimate path** — does the real thing, asserts it works. Guards against fixing by breaking.
3. **The boundaries** — the variations a lazy fix misses.

A suite with only part 1 rewards a fix that denies everyone. A suite with only part 2 is not a security test at all. Both, every time.

## Agree the seams before writing

A **seam** is the boundary you test at: the interface where the property is observable without reaching inside the implementation. Security tests live at seams, never against internals.

Write down which seams are under test and confirm them before writing a line of test code. You cannot test everything, and agreeing the seams up front is how the effort lands on the paths that carry the risk — the auth check, the tenancy filter, the upload sink — instead of spreading evenly across the codebase. An unconfirmed seam is a guess about where the property lives, and a test at the wrong layer passes while the application stays exploitable.

For the vocabulary — seam, module, interface, depth — and for what to do when the property has no seam to be tested at, load `codebase-design`.

## Proving discrimination

A security test is only meaningful if it has been seen to fail on the vulnerable code. Ways to prove it, best first:

- **Write the test before the fix.** It goes red. Then apply the fix. It goes green. This is the cleanest and it is the default when the fix is not in yet.
- **Revert on a scratch branch.** Fix already merged: `git stash` the fix or check out the parent commit, run the test, confirm red, restore. State that you did this.
- **Break it deliberately, once.** Temporarily reintroduce the bug in a local edit, confirm red, revert. Least clean; use only when the other two are impractical.

Never ship a security test whose red state you have not observed. A green test proves nothing about a bug you cannot make it catch.

Discrimination is necessary and not sufficient. Seeing a test go red on vulnerable code proves it can disagree with *that* version of the code; it does not prove the assertion has an opinion of its own. A tautological assertion (below) can still go red on the vulnerable code and green on the fix while being incapable of ever disagreeing with the code as written. Check both: it discriminated, and its expected value came from somewhere other than the implementation.

## Boundary cases by vulnerability class

### Broken access control / IDOR

- Attack: user A requests user B's object by id → assert 403/404 **and** assert B's data is absent from the body (a 200 with an empty body and a 403 are different; test the data, not just the status).
- Legitimate: user A requests A's own object → assert success.
- Boundaries:
  - Nonexistent id → same response as unauthorized id (no distinguishing 404-vs-403 oracle).
  - The object at a list endpoint, not just the detail endpoint.
  - Every verb: can A read, update, delete, or export B's object?
  - Nested resources: A accessing `/orgs/B/reports/valid-A-report-id`.
  - Role boundary: a regular user hitting an admin-only route.

### Authentication / token handling

- Attack: token signed with `none`, with HS256 against the public key, with the wrong key, expired, wrong issuer, wrong audience → each rejected.
- Legitimate: a correctly signed, current token → accepted.
- Boundaries:
  - Expiry exactly at the boundary second.
  - `alg: none` with and without a trailing dot.
  - A valid token for a different environment (staging token on prod).
  - A revoked token, if revocation exists — if it does not, that is a finding, not a test gap.

### Injection (SQL, command, template)

- Attack: the metacharacter payload → asserts it is treated as data (the literal string is searched for, or the input is rejected), not executed.
- Legitimate: input that legitimately contains the metacharacter (`O'Brien`, a title with `%`, a filename with a space) → works correctly.
- Boundaries:
  - Encoded payloads: URL-encoded, double-encoded, unicode variants.
  - The payload in every field that reaches the sink, not just the reported one.
  - Identifier positions (ORDER BY, table names) separately — parameterization does not cover them.

### Path traversal

- Attack: `../`, `%2e%2e%2f`, `..\\`, an absolute path, a symlink → assert contained.
- Legitimate: a valid filename, including one with dots in the name (`my.report.2024.pdf`) → works.
- Boundaries: the payload after URL decoding, after unicode normalization, and a path that is valid until the last segment escapes.

### SSRF

- Attack: URL pointing at `169.254.169.254`, `localhost`, an internal IP, a `file://` scheme, a redirect to any of those → blocked.
- Legitimate: an allowed external URL → works.
- Boundaries: redirect chains (the first hop allowed, the second internal), DNS names that resolve to internal IPs, IPv6 and decimal-encoded IP forms.

### XSS

- Attack: the payload rendered → assert it appears escaped in the output, or load it in a real DOM and assert no script execution.
- Legitimate: content that contains `<`, `>`, `&` legitimately → rendered correctly, not double-escaped.
- Boundaries: the payload in each context — HTML body, attribute, JS string, URL, CSS. Each escapes differently.

### Rate limiting / resource bounds

- Attack: N+1 requests in the window → the N+1th is rejected.
- Legitimate: N requests → all succeed; the window resets correctly after.
- Boundaries: the limit is per-account not per-IP (or both); a slow-drip that stays under the limit; the reset boundary.

## Framework patterns

**Multi-tenant setup.** Seed at least two tenants with at least one object each. The single most common bug in tenant-isolation tests is seeding one tenant and asserting they see their own data — which passes even when isolation is completely broken. You need B's data to exist for A's attack to have something to steal.

**Auth in integration tests.** Prefer minting a real token via the app's own signing path over hand-crafting one, so the test exercises the real verification. Keep one hand-crafted-token test specifically for the forgery cases the real path would never produce.

**Negative assertions cleanly.** Assert on the specific outcome, not merely "an error." `expect(res.status).toBe(403)` plus `expect(res.body).not.toContain(tenantB.secretField)`. A test that passes on any non-200 will pass on a 500 caused by a typo, and you will not notice the control is gone.

**No oracles in the test's own responses.** If the fix makes "not found" and "not yours" return the same thing, the test should assert they are the same thing — identical status and identical body. Encode the anti-oracle property, because it is exactly what a future change will regress.

## Anti-patterns

Three of these are structural — the test is worthless on the day it is written, and no amount of running it will reveal that.

### Tautological

The assertion recomputes the expected value the way the code computes it, so it passes by construction and can never disagree with the implementation. In security tests this hides well: asserting that the signature check accepts a token you signed by calling the same signing helper the verifier trusts; asserting the sanitiser's output equals `sanitise(input)`; asserting the tenant filter returns the rows that `scopedQuery()` returns; a snapshot generated from the current output and then blessed.

Expected values must come from an independent source of truth — a literal you wrote by hand, a payload from the actual exploit, a worked example from the spec or the CVE, a fixture captured from a system that is known good. If the only way to know the expected value is to ask the code, the test is asking the accused to testify.

### Implementation-coupled

The test mocks internal collaborators, reaches into private methods, or verifies through a side channel — querying the database directly to confirm the row was not written, rather than asking the interface for it. The tell is a test that breaks on refactor while behaviour is unchanged.

This is more acute for security tests than for ordinary ones, because a security test is meant to outlive several refactors of the code it guards. That is its whole job: it is there for the change nobody remembers making, two years out. A test coupled to today's internals gets deleted or "updated" during the first restructure, by someone who has no idea what it was defending.

### Horizontal slicing

Writing every test first and then all the implementation. Bulk-written tests verify imagined behaviour — the shape you pictured rather than what the system does — and they go insensitive to real changes because you committed to the test structure before you understood the code.

Work in vertical slices instead: one test, one piece of implementation, then the next test chosen in light of what the last one taught you. Each test is a tracer bullet. With security tests this matters twice over, because writing the attack usually teaches you something about the real attack surface — a second parameter that reaches the same sink, an encoding the framework already decodes — and a batch of tests written up front cannot respond to that.

### Smaller tells

- Testing a mock of the vulnerable function instead of the function. The mock is not the code that ships.
- Asserting only the status code when the body is where the data leaks.
- A single test that seeds one tenant — passes on broken isolation.
- Sleeps for timing-dependent tests. Use fake timers or assert on the mechanism, not the clock.
- Hardcoded record ids that a seed change invalidates.
- A test named `test_security` that reveals nothing on failure.
- Committing a green security test whose red state was never observed.
