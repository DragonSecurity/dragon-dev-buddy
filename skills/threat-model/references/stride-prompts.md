# STRIDE prompts by element type

STRIDE fails when applied as six words. It works when applied as questions specific to the element in front of you. Use these; skip the categories that genuinely do not apply and say you skipped them.

## Which categories apply to which element

| Element | S | T | R | I | D | E |
| --- | --- | --- | --- | --- | --- | --- |
| External actor | ✓ | | ✓ | | | |
| Process (service, handler, job) | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Data store | | ✓ | ✓ | ✓ | ✓ | |
| Data flow | | ✓ | | ✓ | ✓ | |

An element crossing a trust boundary gets all six regardless. The boundary is the point.

## Spoofing — becoming someone you are not

- What proves the caller is who they claim? Where is that proof verified, and can any path skip the check?
- Are tokens verified for signature, issuer, audience, and expiry, or only decoded?
- Can a token issued for one tenant, environment or scope be replayed against another?
- For machine-to-machine flows: is it mTLS, a shared secret in an env var, or nothing because it is "internal"?
- Webhook receivers: is the signature verified with a constant-time comparison against the raw body, before parsing?

## Tampering — changing data you should not

- Which fields does the client send that the server should be deriving? Prices, roles, user ids, totals, timestamps.
- Is the object built from a whitelist of fields, or does it mass-assign whatever arrived?
- Can an integer overflow, negative number, or scientific-notation string reach an amount field?
- Are files written to a path built from user input? Can `../` escape it?
- Is anything deserialized that came from outside? Which library, and does it construct arbitrary types?

## Repudiation — denying you did it

- Is there an audit record for privileged actions: role change, data export, refund, deletion, impersonation?
- Does the record capture actor, target, before/after, timestamp and source address?
- Can the actor edit or delete their own audit trail?
- If an admin impersonates a user, does the log show the admin or the user? Both is the right answer.

## Information disclosure — seeing what you should not

- Does the response include fields the caller has no right to? Password hashes, internal ids, other tenants' names, full objects where a projection was intended.
- Do error messages differ between "no such user" and "wrong password"? Between "not found" and "not yours"?
- Do timing differences leak the same thing the messages were careful not to?
- What lands in logs: tokens, request bodies, PII, full URLs with secrets in the query string?
- Is anything cached that is user-specific but served with a shared cache key?
- Directory listings, source maps, `.git`, `/debug`, `/metrics`, stack traces in production.

## Denial of service — making it unavailable

- Is there any input where cost grows faster than size? Nested queries, regex on user input, zip or image decompression, recursive JSON.
- Are there unbounded reads: no pagination, no `LIMIT`, `SELECT *` on a table that grows forever?
- What is the rate limit, and is it per-account or per-IP? Per-IP alone is defeated by a rotating proxy; per-account alone lets registration be the attack.
- Can one tenant's load starve another? Shared connection pool, shared queue, shared worker.
- Retry storms: does a failing dependency cause the system to attack it harder?

## Elevation of privilege — doing what you should not

- Where is authorization decided? Once, centrally, or in each handler independently? The second is where the bugs live.
- Is any object accessed by an id from the request without checking ownership? Enumerate every such route; this is the highest-yield question in the whole model.
- Can a user set their own role, tier or feature flags at signup or profile update?
- Do admin routes rely on a hidden UI, an unlinked path, or an `isAdmin` claim the client can influence?
- Does a background job run with more privilege than the user who triggered it, using input the user controlled?

## Recurring patterns worth checking explicitly

**Multi-tenant.** Tenant scoping enforced per-query is a defect waiting to happen; the tenth developer will forget it. Look for enforcement at the repository, middleware or row-level-security layer. Ask what happens on a query written by someone who has never heard of the tenancy rule.

**Public API.** Object ids that are sequential integers turn a missing authorization check into a full data export. This is the single most common critical finding in real applications.

**Queue and worker.** The message is input. Is it authenticated? Can a message be replayed, reordered, or poisoned? Does a poison message block the queue or spin forever?

**Third-party callbacks.** Unauthenticated by nature, and usually trusted by default because "it comes from Stripe." Verify the signature and treat the payload as a hint to go and ask the third party for the truth, not as the truth.

**File upload.** Content type from the client is a suggestion. Check magic bytes, cap size before buffering, store outside the web root, never serve from the same origin as the app, and never construct the path from the filename.

**Password reset and email change.** The classic chain: request a reset for the victim's address, receive it at yours because the flow trusts a `Host` header or an unvalidated `redirect_to`. Model the full sequence, not each endpoint alone.
