# Worked example: downgrading a reported Critical

---

**User:** bug bounty report came in, rated Critical — "SQL injection in report search". Here's their writeup: sending `name=' OR '1'='1` to `GET /reports?name=...` returns more results than expected. They want $5k.

**Assistant:** Own code, own repo, local build — authorization gate is satisfied, so I can write a PoC against your own environment if it comes to that.

**Claim, precisely:** an authenticated user passes SQL metacharacters in the `name` query parameter and alters the query's WHERE clause, potentially reading rows they shouldn't.

Let me find the actual code path before I believe or dismiss it.

*(reads `src/routes/reports.ts`, follows `name` to the query builder in `src/db/search.ts`)*

**Code path:** `src/routes/reports.ts:31` takes `req.query.name` and passes it to `searchReports()` at `src/db/search.ts:18`:

```ts
export function searchReports(tenantId: string, name: string) {
  return prisma.$queryRaw`
    SELECT id, title, amount FROM reports
    WHERE tenant_id = ${tenantId} AND title ILIKE ${'%' + name + '%'}
  `;
}
```

This is `$queryRaw` as a **tagged template**, not string concatenation. Prisma parameterizes every `${}` interpolation in a tagged template — `name` becomes a bound parameter, not query text. The `' OR '1'='1` reaches the database as the literal four-character string, wrapped in `%...%`, and matches any title containing that string.

**Refutation attempted, and it holds:** I ran the reporter's exact payload against a local build seeded with two tenants.

```
GET /reports?name=' OR '1'='1
→ 200, 0 rows for tenant A (no report title contains that literal string)
```

Then I confirmed the parameterization directly:

```
GET /reports?name=%25
→ returns tenant A's reports whose title contains "%", correctly, and none of tenant B's
```

The "more results than expected" the reporter saw is almost certainly the `%` wildcard behavior of `ILIKE` — a `%` in their input is a wildcard, so `name=%` matches everything *within their own tenant*. That is a mild surprise, not an injection, and not a cross-tenant read.

**Verification:** reproduced locally, both payloads, two tenants. High confidence. This is not SQL injection.

**Severity here: Informational**, versus the reported Critical. The gap is that the reporter saw a result-count change and inferred injection without confirming the mechanism — a reasonable thing to report, an incorrect thing to rate Critical.

## But — the refutation surfaced something real

While confirming tenant isolation, I checked whether `name` is the *only* thing that reaches `$queryRaw` in this file. It is not. `src/db/search.ts:41`, `sortReports()`:

```ts
export function sortReports(tenantId: string, column: string, dir: string) {
  return prisma.$queryRawUnsafe(
    `SELECT * FROM reports WHERE tenant_id = $1 ORDER BY ${column} ${dir}`, tenantId
  );
}
```

`$queryRawUnsafe` with `column` and `dir` interpolated as **raw string**, and `column` comes from `req.query.sort` at `reports.ts:34` with no allowlist. `tenant_id` is parameterized, but the `ORDER BY` is not — and you cannot parameterize an identifier, so a bound parameter is not the fix here.

**This** is the SQL injection. Not the one reported. Reachable by any authenticated user via `?sort=`, and while `ORDER BY` injection is more constrained than a WHERE injection, it reaches subqueries: `?sort=(SELECT CASE WHEN (SELECT amount FROM reports WHERE tenant_id!=$1 LIMIT 1)>0 THEN title ELSE id END)` is a boolean oracle across tenant boundaries. Financial data, cross-tenant. **That's the Critical**, and it was two functions away from where anyone was looking.

## Fix

**Now** — allowlist the identifier, since it cannot be parameterized:

```ts
const SORTABLE = { title: 'title', amount: 'amount', created: 'created_at' } as const;
const DIRS = { asc: 'ASC', desc: 'DESC' } as const;

export function sortReports(tenantId: string, column: string, dir: string) {
  const col = SORTABLE[column as keyof typeof SORTABLE] ?? 'created_at';
  const d = DIRS[dir as keyof typeof DIRS] ?? 'DESC';
  return prisma.$queryRawUnsafe(
    `SELECT * FROM reports WHERE tenant_id = $1 ORDER BY ${col} ${d}`, tenantId
  );
}
```

**Then** — grep the codebase for every `$queryRawUnsafe` and every `${` inside a `$queryRaw`, because a team that reached for `Unsafe` once reached for it in a pattern. This is a class, not an instance. Handing the class sweep to a fresh `secure-code-review` scoped to raw queries.

Regression test going to `security-test-writer`: the boolean-oracle payload must return the same rows as a plain sort, and must never return a different tenant's data.

## Reply to reporter

> Thanks for this. We dug into it carefully. The specific payload you sent isn't SQL injection — that endpoint uses parameterized queries, so `' OR '1'='1` reaches the database as a literal string. The extra results you saw are the `%` in ILIKE acting as a wildcard within your own account. Here's the reproduction showing tenant isolation holds: [attached].
>
> **That said** — your report sent us looking, and two functions over we found a genuine SQL injection in the `sort` parameter that we'd missed. That one's real, it's Critical, and it's being fixed today. We think your report materially led us to it, so we're awarding the bounty. Credit as you'd like it.

---

## What this run got right

- Restated the claim precisely, then went to the code before believing or dismissing it.
- Identified the tagged-template-versus-`Unsafe` distinction that is the whole ballgame, and **verified** it by running both payloads rather than reasoning about it.
- The refutation of the reported bug is what uncovered the real one. That is the entire argument for step 4 existing.
- Downgraded the report to Informational and **upgraded the actual finding to Critical**, both with evidence — the two directions the rubric warns about, in one triage.
- The fix names why parameterization can't apply to an identifier, gives the allowlist, and escalates to a class sweep.
- The reporter reply is honest about the downgrade, credits the real chain of causation, and pays the bounty for the assist rather than on a technicality. That reporter comes back.
