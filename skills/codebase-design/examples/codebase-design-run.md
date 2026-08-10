# Worked example: a tenant predicate copied nineteen times, put behind one interface

---

**User:** `src/db/index.ts` is basically a wrapper around the pg pool. Someone said it should be a repository. Is that worth doing?

**Assistant:** "Should be a repository" is a shape with no argument attached, so let me get the argument first. Two things decide it: what the current interface really is, and what disappears if I delete the module.

*(reads `src/db/index.ts`, greps its callers)*

**The current interface, written out in full.** The signatures are small — `query(sql, params)`, `tx(fn)`, `pool`. The interface is not.

- `query` takes raw SQL. Every caller must know the schema.
- Every caller must add `AND tenant_id = $n` itself, and must know that the tenant id comes from `req.session.tenantId` and not from anything in the request body.
- Every caller must know that `tx` does not nest, and that calling `query` inside a `tx` callback uses the pool, not the transaction — so it silently escapes the transaction.
- `pool` is exported, so the two constraints above are optional.

That is four paragraphs a caller has to carry, three of which no compiler checks. This is a shallow module: the interface is nearly as large as the implementation, and it is a large interface written in prose instead of in types.

**Deletion test.** Delete `src/db/index.ts` and rewrite the callers against `pg` directly. What vanishes: eleven lines of pool configuration. What reappears: nothing, because none of the four constraints are actually enforced here — they are enforced at the callers already. It is a pass-through. The module is not earning its keep, which is a stronger result than "it should be a repository": there is no depth to preserve, only depth to add.

**Scatter audit — grepped, not estimated.**

```
$ grep -rn "FROM \(documents\|shares\|comments\|audit_log\)" src/ | wc -l
19
$ grep -rLn "tenant_id" $(grep -rl "FROM documents" src/)
src/jobs/reindex.ts
```

Nineteen query sites carry the tenant predicate by hand. **One does not.** `src/jobs/reindex.ts:44` selects every document in the table to rebuild the search index, and writes each into a per-document index entry without a tenant field. That is a cross-tenant read, and downstream it is a cross-tenant search result.

That is a finding, not a design detail, and I am not fixing it quietly inside a design change where it would land in a diff nobody reviews as a security fix. Handing it to `vuln-triage` to get a severity and a reachability answer, and to `security-test-writer` for a regression test, before this design lands. The design is the reason it will not happen a twentieth time; the triage is what deals with the nineteenth.

**Dependencies.** One: Postgres. Category 2, local-substitutable — the suite already starts a real Postgres in a container for the migration tests. That decides the testing strategy: no port, no mock repository, tests run against the real engine. It matters here specifically. A tenant predicate verified against an in-memory fake repository proves only that my fake honors my fake; the property I need is that the real engine returns zero rows, and only the real engine can say that.

**Seam.** At the repository: one interface between application code and SQL, and no other route to the pool.

It is worth being precise that this seam is **not** the trust boundary. The trust boundary is at the request edge, where a session cookie becomes a claim about who the caller is — `threat-model` already draws it there. What sits at the repository seam is the *consequence* of that decision, carried down to where it can still be forgotten. The seam is where I put the control; the boundary is where the trust changed. Putting a validation at the repository would be ceremony; putting the tenant scope there is the whole point.

**Three designs, in parallel, then a recommendation.**

1. *Minimal.* `repo.documents.list(scope)`, `.get(scope, id)`, `.write(scope, doc)`. Three entry points, `scope` passed explicitly. Deep, but `scope` is a plain string, so `list("")` compiles.
2. *Flexible.* A query builder with a `.forTenant()` method. Supports every existing call site unchanged. Also supports omitting `.forTenant()`, which is the bug we are here to remove — this design's flexibility is precisely the flexibility that caused the problem.
3. *Unsafe call unrepresentable.* A `TenantScope` branded type that only `requireSession()` at the request edge can mint. Every repository function takes one. There is no other constructor and no export of the pool.

Design 3, taking design 1's three entry points. The comparison that decides it: on depth they are similar, on locality all three concentrate the predicate in one place, but only design 3 closes the path back out. Under 1 and 2 the control is enforced by review, which is exactly the mechanism that produced `reindex.ts`.

`reindex.ts` is the interesting case for design 3, because it is a background job with no session to mint a scope from. That is the design working: the job genuinely does need to cross tenants, and it should have to say so. It gets `forEachTenant(scope => ...)`, which iterates and mints a scope per tenant — the cross-tenant read becomes nineteen scoped reads and the index entries come out tenant-tagged. The escape hatch is named `unsafeSystemScope`, it lives in one file, and CI greps for it outside that file.

**Wrong path closed by:** `pool` is no longer exported and the `pg` import is banned outside `src/db/` by an ESLint `no-restricted-imports` rule. Checked by CI, not by review. `TenantScope`'s brand is checked by the compiler.

**Test surface:** tests call the repository interface against the containerized Postgres. The central one is a call with tenant A's scope for a row id belonging to tenant B, asserting zero rows — one test, at the interface, covering what nineteen hand-written predicates used to cover individually. The four existing tests in `test/db.test.ts` that assert `query()` passes SQL through are deleted: they test the shape being removed.

**Handoff:** moving the nineteen call sites onto the new interface without changing behavior is `refactor-safely`'s job, under its net, and the tenant tests above are part of that net before the first call site moves.

---

## What this run got right

- Answered "is it worth doing" with the deletion test and a number, and got the more useful answer — a pass-through with no depth to preserve — instead of agreeing that it should be a repository.
- Wrote the current interface out in full, which is what exposed a four-paragraph interface hiding behind three small signatures.
- Counted the scatter with grep and found the call site that had already lost the control, then routed it as a finding instead of folding a security fix into a design change.
- Filed the dependency in a category and let that decide the test strategy, with the reason a mock would have been worthless for this particular property.
- Named the seam and said explicitly that it is not the trust boundary, and what each one is for.
- Designed the interface three ways and recommended one in its own voice, on the grounds that only that design removes review from the enforcement path.
- Treated the awkward caller as information about the design rather than an exception to it, and gave the escape hatch a name that costs something to type and a CI check that finds it.
- Said which mechanism holds the wrong path shut and who enforces it — compiler and CI, not habit.
- Named the tests being deleted, and handed the code motion on.
