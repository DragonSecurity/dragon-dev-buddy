# Worked example: regression tests for an IDOR fix

Following the cross-tenant read (F1 / T1) from the review and threat-model examples.

---

**User:** write a test for the cross-tenant report read we just fixed in the repository layer

**Assistant:** Framework is vitest (found `vitest.config.ts` and `test/*.test.ts`), and there's a `test/helpers/seed.ts` already. I'll match that.

**Property to assert:** an authenticated user cannot read a report belonging to another tenant, through any route that fetches a report by id.

**Test layer:** integration, at the route level. The bug lived in the handler's call to the repository, and the fix (a `scopedReport(tenantId)` accessor plus a lint guard) has to be exercised through the route to prove the handler actually uses it. A unit test of `scopedReport` would pass even if a handler bypassed it — which is the exact failure mode we're guarding against.

**Discrimination:** the fix is already merged, so I checked out `HEAD~1` (the vulnerable commit), ran these tests, and confirmed all three attack cases go red there; on `HEAD` they're green. Method noted below the code.

```ts
// test/security/report-tenant-isolation.test.ts
import { describe, it, expect, beforeEach } from 'vitest';
import { buildApp } from '../../src/app.js';
import { seedTenant, tokenFor } from '../helpers/seed.js';

describe('report tenant isolation', () => {
  let app, tenantA, tenantB, reportB;

  beforeEach(async () => {
    app = await buildApp();
    // Two tenants, each with a report. Both must exist, or the attack has
    // nothing to steal and the test passes on broken isolation.
    tenantA = await seedTenant({ name: 'A' });
    tenantB = await seedTenant({ name: 'B' });
    reportB = await seedTenant.report(tenantB, { title: 'B quarterly', amount: 90000 });
  });

  // --- The attack: must be red on the vulnerable commit ---
  it('rejects_cross_tenant_report_read_by_id', async () => {
    const res = await app.inject({
      method: 'GET',
      url: `/reports/${reportB.id}`,
      headers: { authorization: `Bearer ${tokenFor(tenantA)}` },
    });
    expect(res.statusCode).toBe(404);
    // Assert on the data, not only the status — a 200 with B's body and a
    // 404 are worlds apart, and status alone would miss a partial leak.
    expect(res.body).not.toContain('B quarterly');
    expect(res.body).not.toContain('90000');
  });

  it('rejects_cross_tenant_report_read_via_export', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/reports/export',
      headers: { authorization: `Bearer ${tokenFor(tenantA)}` },
      payload: { reportId: reportB.id },
    });
    expect(res.statusCode).toBe(404);
    expect(res.body).not.toContain('90000');
  });

  // --- The legitimate path: guards against "deny everyone" passing ---
  it('allows_own_tenant_report_read', async () => {
    const reportA = await seedTenant.report(tenantA, { title: 'A quarterly', amount: 10000 });
    const res = await app.inject({
      method: 'GET',
      url: `/reports/${reportA.id}`,
      headers: { authorization: `Bearer ${tokenFor(tenantA)}` },
    });
    expect(res.statusCode).toBe(200);
    expect(res.body).toContain('A quarterly');
  });

  // --- Boundaries ---

  // No 404-vs-403 oracle: a nonexistent id and another tenant's id must be
  // indistinguishable, or enumeration leaks which ids exist.
  it('nonexistent_and_unauthorized_ids_are_indistinguishable', async () => {
    const auth = { authorization: `Bearer ${tokenFor(tenantA)}` };
    const missing = await app.inject({ method: 'GET', url: `/reports/99999999`, headers: auth });
    const forbidden = await app.inject({ method: 'GET', url: `/reports/${reportB.id}`, headers: auth });
    expect(missing.statusCode).toBe(forbidden.statusCode);
    expect(missing.body).toBe(forbidden.body);
  });

  // The list endpoint must scope too — the fix was on the detail route, and
  // list is exactly where "we fixed it" quietly doesn't cover.
  it('list_excludes_other_tenant_reports', async () => {
    const res = await app.inject({
      method: 'GET', url: '/reports',
      headers: { authorization: `Bearer ${tokenFor(tenantA)}` },
    });
    expect(res.body).not.toContain('B quarterly');
  });
});
```

## Security test: report tenant isolation

**Finding:** An authenticated tenant A user requests `GET /reports/:id` for a tenant B report and receives it; the handler filtered by id but not by tenant.

**Property asserted:** no user can read another tenant's report through any by-id fetch.

**Test layer:** integration (route level) — the defect was a handler bypassing tenant scope, so it must be tested through the handler.

**Discrimination proven:** checked out `HEAD~1`, ran the file. The three negative/boundary attack tests (`rejects_*`, `nonexistent_*`, `list_*`) went red; `allows_own_tenant_report_read` stayed green. On `HEAD`, all green. So the suite catches the bug and does not just pass vacuously.

**Tests written:**
- `rejects_cross_tenant_report_read_by_id` — the reported attack. Red on vulnerable.
- `rejects_cross_tenant_report_read_via_export` — same defect, the export path from T2. Red on vulnerable.
- `allows_own_tenant_report_read` — legitimate path; blocks a deny-all "fix."
- `nonexistent_and_unauthorized_ids_are_indistinguishable` — the anti-oracle property, which a future error-message change is likely to regress.
- `list_excludes_other_tenant_reports` — the endpoint the detail-route fix silently doesn't cover.

Two of these — the anti-oracle and the list case — go red on `HEAD` too, actually. The detail fix didn't cover them. So this test file just found two live variants of the bug the "fix" left open. Handing those back for a repository-layer fix rather than a per-route one.

---

## What this run got right

- Chose the integration layer with an explicit reason tied to where the bug lives, and rejected the faster unit test with a reason.
- Seeded **two** tenants — the setup detail that determines whether the test can detect the bug at all.
- Asserted on the leaked data, not just the status code.
- Included the legitimate-path test so "deny everyone" can't pass.
- Encoded the anti-oracle property as its own test, because that is precisely what a future change regresses.
- **Proved discrimination** by checking out the parent commit, and stated the method.
- The boundary tests found two variants the original fix missed — which is the entire reason boundary cases exist.
