# Abuse case library

For each feature type: how it is attacked, the defense that closes each attack, and where possible the design that makes the attack structurally impossible rather than merely checked.

The last column is the important one. A check can be forgotten by the next developer; a structure cannot.

## Authentication

| Abuse | Defense | Structural version |
| --- | --- | --- |
| Credential stuffing | Rate limit per account and per IP; detect breached passwords | — |
| Username enumeration via error messages | Identical response and timing for "no such user" and "wrong password" | Single generic auth failure path, no branch on which factor failed |
| Session fixation | Regenerate session id on login | Framework session middleware that does this by default |
| Token forgery | Verify signature with explicit algorithm, issuer, audience, expiry | Asymmetric signing so the verifier cannot mint tokens |
| Password reset hijack via Host header | Build reset URLs from server config, never the request Host | — |
| No logout / revocation | Server-side session invalidation | Short-lived tokens + refresh, so revocation is bounded by TTL |

## File upload

| Abuse | Defense | Structural version |
| --- | --- | --- |
| Executable disguised as image | Validate by magic bytes; allowlist types; never trust extension or client content-type | Process through a library that re-encodes, so the output is provably the claimed type |
| Path traversal via filename | Generate the stored name server-side; never use the client filename in the path | Store by generated UUID; keep the original name as metadata only |
| Zip bomb / decompression DoS | Cap size before buffering; cap decompressed size | Stream with a hard byte ceiling that aborts |
| Stored XSS via SVG or HTML | Serve uploads from a separate origin; `Content-Disposition: attachment`; CSP | Dedicated upload domain with no auth cookies scoped to it |
| Overwrite another user's file | Namespace storage by owner; never a shared flat path | Object key includes tenant + owner, enforced at write |
| Malware served to other users | Scan on upload; serve with correct non-executable type | — |

## Payments / money movement

| Abuse | Defense | Structural version |
| --- | --- | --- |
| Client sets the price | Server derives amount from server-side product data | Never accept an amount field from the client at all |
| Double-spend via replay | Idempotency key on every mutating call | Unique constraint on the operation id in the database |
| Negative / overflow amounts | Validate range and type; use decimal, not float | Money type that rejects invalid values at construction |
| Race on balance check | Transactional check-and-debit, row lock or atomic decrement | Ledger append with a balance constraint the DB enforces |
| Currency confusion | Explicit currency on every amount, validated against the account | Amount and currency as one inseparable value object |
| Webhook forgery | Constant-time signature over raw body; verify before parsing | — |

## Sharing / access grants

| Abuse | Defense | Structural version |
| --- | --- | --- |
| Guess a share link | Unguessable token, not a sequential id | 128-bit random token |
| Share link never expires | Expiry on every link | TTL enforced at the storage layer |
| Escalate a view grant to edit | Server checks the grant's level per action | Capability encodes the level; cannot be widened client-side |
| Access after revocation | Check revocation at use, not only at grant | Short TTL so revocation is bounded |
| Enumerate what's shared with whom | No listing endpoint that leaks others' grants | — |
| Reshare beyond intent | Explicit reshare permission, off by default | Grants that cannot themselves grant |

## Search / query

| Abuse | Defense | Structural version |
| --- | --- | --- |
| Injection via the query | Parameterize; allowlist any identifier (sort column, table) | Query builder that cannot emit raw identifiers |
| Cross-tenant results | Tenant predicate on every query | Row-level security or a scoped accessor the raw client can't bypass |
| ReDoS via a regex search | Bound input length; avoid backtracking patterns | Non-backtracking engine (RE2) |
| Enumerate records via result counts | Consistent responses; rate limit | — |
| Expensive unbounded query | Mandatory pagination and `LIMIT` | Query layer that refuses an unbounded fetch |

## Admin / privileged action

| Abuse | Defense | Structural version |
| --- | --- | --- |
| Regular user reaches admin route | Authorize every admin action server-side, not via UI hiding | Separate admin service or a role check in shared middleware |
| No audit trail | Log actor, target, before/after, timestamp on every privileged action | Audit write in the same transaction as the change |
| Impersonation hides the real actor | Log both the admin and the impersonated user | — |
| Self-granted privilege | Role and tier are never client-settable | Role changes only through an audited admin path |
| CSRF on an admin action | Anti-CSRF token; `SameSite` cookies | — |

## How to use this during a build

1. Identify the feature's type(s) from the list. Many features are two: a "share a file" feature is *sharing* plus *upload*.
2. Take every row for those types as a candidate abuse case.
3. Discard the ones that genuinely do not apply, and say why — an internal tool may not need share-link expiry.
4. For each that remains, prefer the structural column. If you cannot make it structural this sprint, take the defense column and note the structural version as a follow-up.
5. Each surviving row becomes a spec requirement and a negative test.

The goal is not to implement every row. It is to have *considered* every row and made a stated decision on each.
