# Security test patterns

## The shape of every security test

Three parts, always:

1. **The attack** — does the malicious thing, asserts it is stopped. Must be red on vulnerable code.
2. **The legitimate path** — does the real thing, asserts it works. Guards against fixing by breaking.
3. **The boundaries** — the variations a lazy fix misses.

A suite with only part 1 rewards a fix that denies everyone. A suite with only part 2 is not a security test at all. Both, every time.

## Proving discrimination

A security test is only meaningful if it has been seen to fail on the vulnerable code. Ways to prove it, best first:

- **Write the test before the fix.** It goes red. Then apply the fix. It goes green. This is the cleanest and it is the default when the fix is not in yet.
- **Revert on a scratch branch.** Fix already merged: `git stash` the fix or check out the parent commit, run the test, confirm red, restore. State that you did this.
- **Break it deliberately, once.** Temporarily reintroduce the bug in a local edit, confirm red, revert. Least clean; use only when the other two are impractical.

Never ship a security test whose red state you have not observed. A green test proves nothing about a bug you cannot make it catch.

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

- Testing a mock of the vulnerable function instead of the function. The mock is not the code that ships.
- Asserting only the status code when the body is where the data leaks.
- A single test that seeds one tenant — passes on broken isolation.
- Sleeps for timing-dependent tests. Use fake timers or assert on the mechanism, not the clock.
- Hardcoded record ids that a seed change invalidates.
- A test named `test_security` that reveals nothing on failure.
- Committing a green security test whose red state was never observed.
