# Risk ranking rubric

Score likelihood and impact independently, then combine. Always state the reason inline; a bare score is not reviewable.

## Likelihood

| Level | Means |
| --- | --- |
| **High** | Reachable by an unauthenticated attacker, or by any authenticated user, with no special conditions. Requires only tools that already exist. |
| **Medium** | Needs a precondition an attacker can usually arrange: a valid account, a specific role, a race window, a user to click something, or knowledge of an id that is guessable. |
| **Low** | Needs something genuinely hard: a compromised insider, physical access, a leaked secret, or a race window measured in microseconds. |

Scale by `security.exposure`:

- `public` — no adjustment. The internet finds everything.
- `authenticated` — drop one level for anything that needs no account; keep as-is for anything that does. An authenticated-only attack is not less likely just because there is a login page.
- `internal` — drop one level, once. Do not drop below Low. Internal is a delay, not a defence, and the moment the boundary moves the score is wrong.

## Impact

| Level | Means |
| --- | --- |
| **High** | Loss or exposure of the sensitive data class, unauthorized movement of money, full account takeover, remote code execution, or a foothold into another system. |
| **Medium** | Partial data exposure, one account compromised, integrity damage that is recoverable, sustained outage of a core function. |
| **Low** | Information that is annoying but not exploitable on its own, brief degradation, a defect that needs to be chained with something else to matter. |

Scale by `security.data_sensitivity`:

- `credentials` — raise one level. Compromise here cascades into systems you do not control and cannot clean up.
- `financial`, `phi` — raise one level for confidentiality or integrity threats. Both carry mandatory notification, which turns a technical finding into a legal one.
- `pii` — raise one level for bulk exposure only. One record and a hundred thousand records are different events.
- `none` — no adjustment.

Never lower impact because "the data is public anyway" without checking integrity and availability. Public data still needs to be correct and present.

## Combining

|  | Impact Low | Impact Med | Impact High |
| --- | --- | --- | --- |
| **Likelihood High** | Medium | High | **Critical** |
| **Likelihood Med** | Low | Medium | High |
| **Likelihood Low** | Low | Low | Medium |

Critical means stop and fix. High means fix this sprint. Medium goes in the backlog with a date. Low is a candidate for explicit acceptance.

## Worked examples

**Multi-tenant IDOR on a reporting endpoint.** Public API, financial data. Any authenticated user, sequential ids, no special conditions → Likelihood High. Bulk exposure of another tenant's financial records → Impact High, raised from Medium by `financial`. → **Critical.**

**Timing difference in a login handler.** Public, PII. Measurable but noisy over the internet, needs many requests → Likelihood Medium. Reveals which addresses are registered, exploitable only in combination → Impact Low. → **Low.** Worth recording, worth accepting if the fix is expensive.

**Unbounded query on an internal admin export.** Internal exposure drops likelihood from High to Medium. Impact is a sustained outage of a core function → Medium. → **Medium.** Note explicitly that this becomes High the day the admin panel is exposed to a VPN-less workforce.

**Homegrown webhook signature verification with a non-constant-time compare.** Public, credentials-grade. Remote, unauthenticated, but the attack needs many thousands of requests and a stable network path → Likelihood Medium. Forged settlement callbacks move money → Impact High. → **High.**

## Writing the reason

One clause, naming the specific fact that drove the score.

Good: "Likelihood High — `GET /reports/:id` takes a sequential integer and the handler has no tenant filter, so enumeration works from any account."

Bad: "Likelihood High — this is a common vulnerability."

The second tells the reader nothing they can check, argue with, or re-evaluate in six months.
