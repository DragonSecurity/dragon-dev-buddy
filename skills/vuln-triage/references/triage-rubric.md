# Triage rubric

## Severity for this system

| Severity | Means here |
| --- | --- |
| **Critical** | Unauthenticated attacker reaches the sensitive data class, moves money, executes code, or takes over accounts. Or any authenticated user reaches all other users' data. Fix now; consider whether it is already an incident. |
| **High** | Authenticated attacker escalates, reads others' data in bounded quantity, or forges a trusted message. Or unauthenticated sustained outage. Fix this sprint. |
| **Medium** | Needs a precondition an attacker can usually arrange, or the outcome is partial. Backlog with a date. |
| **Low** | Real but weak alone. Information leaks, missing defence in depth. Candidate for explicit acceptance. |
| **Informational** | Not a vulnerability. Hardening opportunity or a difference of opinion about defaults. Say so directly. |

## Calibrating against the reporter

Reporters are systematically miscalibrated in both directions, for structural reasons.

**Over-rated, commonly:**
- Self-XSS ("paste this in your console"). No attacker-controlled delivery path, so no vulnerability.
- Missing security headers with no demonstrated impact. Informational unless chained to something real.
- Findings on a staging or demo host with no real data. Rate on the data, not the hostname.
- CSRF on an endpoint that changes nothing, or that is already protected by `SameSite` defaults.
- "Version disclosure" and banner grabbing. Informational.
- Rate limiting absent on an endpoint that costs nothing to call.
- CVSS scored for a hypothetical worst-case deployment rather than yours.

**Under-rated, commonly:**
- An IDOR reported on one endpoint that is actually a systemic pattern across forty. The reporter found one; you should look for the class.
- An information leak the reporter did not realize completes a chain they did not see.
- A "low impact" bug on an endpoint that turns out to sit on the money path.
- Anything the reporter could not fully exploit because they stopped at the ethical boundary. They were right to stop; you should not assume the limit is technical.

Always state when you are moving the severity, and why. A reporter who sees reasoning will argue productively. A reporter who sees a silent downgrade goes public.

## Reachability preconditions

Enumerate each one explicitly. Each is a place the finding could die.

- [ ] **Authentication** — none, any account, or a specific one?
- [ ] **Authorization** — any role, or a privileged one? How hard is that role to obtain?
- [ ] **Configuration** — does it need a non-default setting? Is that setting on in production?
- [ ] **Feature flag** — on, off, or partially rolled out?
- [ ] **Timing** — a race window? How wide, in practice, not in theory?
- [ ] **Prior knowledge** — an id, a token, an email address? Guessable, enumerable, or genuinely secret?
- [ ] **Network position** — internet, VPN, same host, same cluster?
- [ ] **User interaction** — does a victim have to click, and would they plausibly?
- [ ] **Volume** — one request or ten million? Ten million against a rate-limited endpoint is a different finding.

A finding with five preconditions is not automatically low. A finding with zero is automatically high.

## False-positive patterns by class

**SQL injection.** The ORM parameterizes; the concatenation is on an identifier that comes from an in-code allowlist; the input is coerced to an integer before it reaches the query. Check whether the "injectable" value is actually attacker-controlled at that point, or derived from something the server set.

**XSS.** The framework escapes by default and this call site does not opt out; the sink is `textContent`, not `innerHTML`; the CSP has no `unsafe-inline` and the payload cannot execute. Verify by actually loading it, not by reading the string.

**SSRF.** The URL host comes from a fixed allowlist; the HTTP client does not follow redirects; the network has no egress. Check the redirect behavior specifically — it is the most common way a "fixed" SSRF is still live.

**Path traversal.** The path is resolved and then checked for containment *after* resolution; the framework normalizes before the handler sees it. `..` appearing in a string is not traversal; escaping the root is.

**Deserialization.** The parser is a safe loader; the type allowlist is genuine; the input is JSON, not a pickle. But: check the JSON path for prototype pollution separately — different bug, same report.

**IDOR.** There is an ownership check, just not where the reporter looked — in middleware, in a repository layer, in a database policy. Find it before dismissing. And if you find it in only one of three places, the report is under-rated, not wrong.

**Timing attack.** Real in theory; over the internet, jitter usually exceeds the signal. Rate it on whether the attacker is local or remote, and whether the secret is short enough to recover before rotation.

## Reporter reply templates

Adapt; do not send verbatim. The pattern that matters: verdict, evidence, timeline, and no defensiveness.

**Confirmed:**
> Thanks — this is real and we've confirmed it. [One sentence of what we verified.] We've rated it [severity] because [reasoning]. Fix is going out [timeline]. We'll let you know when it's deployed. [Bounty/credit note if applicable.]

**Confirmed, lower severity:**
> Confirmed, thank you. We're rating this [severity] rather than [their rating], because [specific reason — e.g. the endpoint requires an admin role, and admin accounts are provisioned manually]. It's still going to be fixed, on [timeline]. If we've got the reachability wrong, tell us what we're missing — happy to re-rate.

**Not exploitable:**
> We looked at this closely. The code path you found is there, but [specific blocker, named]. Here's what we checked: [evidence]. That said, [the blocker] is the only thing stopping it, so we're [tracking it / hardening it] separately. If you can get past [the blocker], we'd very much like to see it.

**Not applicable:**
> Thanks for looking. We're not affected: [specific evidence — version outside the affected range, the feature isn't enabled, the code was removed in a previous release]. Here's how to verify: [reproducible check].

**Duplicate:**
> This matches [reference], reported [date] and currently [status]. [Bounty policy note.] Your report described [the thing they found that the original didn't], which is useful — we've added it to the ticket.
