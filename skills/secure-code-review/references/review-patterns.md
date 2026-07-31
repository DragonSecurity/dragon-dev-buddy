# Review patterns

## Sinks worth tracing to

A sink is where attacker-controlled data does something. Trace every input to every sink.

| Sink | The question |
| --- | --- |
| SQL / ORM raw query | Is it parameterized, or concatenated? ORMs have raw escape hatches; find them. |
| NoSQL query | Can an object arrive where a string was expected? `{"$ne": null}` in a JSON body. |
| Shell / `exec` | Is there a shell at all? `spawn` with an array is safe; `exec` with a template string is not. |
| Filesystem path | Can `../` escape? Is the result checked to still be inside the intended root **after** resolution? |
| HTTP client URL | SSRF: can the host be chosen by the caller? Are redirects followed? Is the metadata endpoint reachable? |
| Template render | Is it escaped by default, and does this call site opt out? `dangerouslySetInnerHTML`, `\|safe`, `v-html`. |
| Redirect target | Open redirect, and the phishing chain it enables. |
| Deserializer | `pickle`, `yaml.load`, Java serialization, `unserialize`. Any of these on untrusted input is Critical by default. |
| Comparison deciding access | Is it constant-time where it needs to be? Does `==` coerce? |
| Regex | Can the pattern or the input cause catastrophic backtracking? |

## Constructs that are almost always wrong

### JavaScript / TypeScript

- `jwt.decode()` where `jwt.verify()` was meant. Decode does not check the signature.
- `jwt.verify()` without an explicit `algorithms` array. Algorithm confusion lives here.
- `==` on a secret, token or HMAC. Use `crypto.timingSafeEqual` on equal-length buffers.
- `JSON.parse` on a body then spread into a model: mass assignment. Look for `{...req.body}`.
- Prisma/TypeORM/Sequelize `findUnique`/`findOne` by an id from the request with no ownership predicate.
- `res.redirect(req.query.next)`.
- `child_process.exec` with a template literal.
- `Math.random()` for tokens, ids, or anything an attacker benefits from predicting.
- Express: error handler that returns `err.stack`. Route-level auth middleware applied to some routes and not others.
- `cors({ origin: true, credentials: true })` — reflects any origin *with* cookies.

### Python

- `yaml.load` without `SafeLoader`. `pickle.loads` on anything from outside.
- f-strings or `%` inside `cursor.execute`.
- Django: `.raw()`, `.extra()`, `mark_safe`, a `ModelForm` with `fields = '__all__'`.
- Flask: `render_template_string` with user input (SSTI), `send_file` with a user-supplied path.
- `subprocess` with `shell=True`.
- `assert` used for an access check. Stripped under `-O`.
- `requests` with `verify=False`.
- Comparing secrets with `==` instead of `hmac.compare_digest`.

### Go

- `fmt.Sprintf` into a query instead of placeholders.
- Ignored errors on anything security-relevant: `_ = json.Unmarshal(...)`, an unchecked error from a verify call.
- `math/rand` where `crypto/rand` is required.
- `http.Client` with no `Timeout`, enabling resource exhaustion.
- `filepath.Join` on user input without a subsequent containment check.
- Middleware ordering that puts logging before auth and logs the credential.

### Java

- Any use of `ObjectInputStream` on untrusted bytes.
- String concatenation into `Statement` instead of `PreparedStatement`.
- XML parsers without `FEATURE_SECURE_PROCESSING` or with external entities enabled (XXE).
- Spring: `@PreAuthorize` on some methods and not others in the same service. `permitAll()` on a path prefix that matches more than intended.

### SQL and data layer, any language

- A query built from an identifier (table or column name) that came from the request. Placeholders do not cover identifiers; use an allowlist.
- `LIKE '%' || input || '%'` with no length cap on a large table.
- Missing `LIMIT` on anything that returns a list.
- A migration that adds a column with a default containing a secret.

## Absent-control checklist

Findings of omission, in rough order of how often they matter.

- [ ] **Ownership check** on every object fetched or mutated by a request-supplied id.
- [ ] **Rate limit** on login, password reset, token exchange, signup, and any endpoint that sends email or SMS.
- [ ] **Audit record** on role change, permission grant, data export, deletion, refund, impersonation.
- [ ] **Revocation path** for every long-lived credential the system issues.
- [ ] **Pagination or `LIMIT`** on every list endpoint.
- [ ] **Size cap** applied *before* buffering an upload or decompressing anything.
- [ ] **Expiry** on tokens, sessions, presigned URLs, and invitations.
- [ ] **CSRF protection** on state-changing routes that accept cookie auth.
- [ ] **Idempotency key** on anything that moves money or sends a message.
- [ ] **Lockout or backoff** after repeated authentication failures.
- [ ] **Tenant predicate** at a layer a new handler cannot bypass.

## Severity rubric

Assign from the attack sentence, then adjust for the project profile.

| Severity | Means |
| --- | --- |
| **Critical** | Unauthenticated attacker reaches the sensitive data class, moves money, executes code, or takes over accounts. Or: any authenticated user reaches every other user's data. |
| **High** | Authenticated attacker escalates privilege, reads data belonging to others in bounded quantity, or forges a trusted message. Or an unauthenticated attacker causes sustained outage. |
| **Medium** | Requires a precondition an attacker can usually arrange, or the outcome is partial. Chainable with something else. |
| **Low** | Real defect, weak outcome alone. Information leaks, missing defence-in-depth, timing signals. |

Adjustments:
- `exposure: internal` — drop one level, once, never below Low, and note what happens when the boundary moves.
- `data_sensitivity: credentials` — raise one level. Cascades into systems outside your control.
- `data_sensitivity: financial` or `phi` — raise one for confidentiality and integrity findings; notification obligations apply.
- Already shipped to production — do **not** raise severity. Raise urgency, in the recommendation, and say so separately.

## The chain check

Before finalizing, ask once: do any two Mediums or Lows compose into something worse? The recurring pairs:

- Verbose errors distinguishing "not found" from "not yours" **+** sequential ids = enumeration becomes a directory listing.
- Open redirect **+** OAuth callback = token theft.
- Any file write with a controllable path **+** a directory the runtime executes from = code execution.
- Username enumeration **+** no rate limit on login = credential stuffing at scale.
- SSRF **+** a cloud metadata endpoint = credential theft.

Report a chain as one finding at the combined severity, naming both halves.
