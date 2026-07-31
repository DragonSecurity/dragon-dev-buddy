# Worked example: from stack trace to regression test

---

**User:** intermittent 500 on `POST /reports/export`, maybe 1 in 50. Stack trace points at `export.ts:58`, `TypeError: Cannot read properties of undefined (reading 'amount')`. Can't reproduce it locally.

**Assistant:** "1 in 50" plus "can't reproduce locally" reads like a data-dependent or race bug, not a logic bug — a pure logic bug at that line would fail every time. Let me reproduce it before touching anything.

*(reads `export.ts:40-65`)*

```ts
const transfers = await db.transfer.findMany({ where: { reportId } });   // :52
const rows = transfers.map((t) => ({ id: t.id, amount: t.amount.toFixed(2) })); // :58
```

`t.amount.toFixed` — so `t.amount` is undefined for some transfers, not all. That's the data dependency. `amount` is defined `Decimal?` in the schema — nullable. So the bug is: **some transfers have a null amount, and 1-in-50 is just how often an export includes one.**

Let me force it rather than wait for production.

*(writes a failing test with a null-amount transfer)*

```ts
it('exports a report containing a transfer with no amount', async () => {
  const report = await seed.report(tenantA);
  await seed.transfer(report, { amount: 500 });
  await seed.transfer(report, { amount: null });   // the 1-in-50 case, forced
  const res = await app.inject({ method: 'POST', url: '/reports/export',
    payload: { reportId: report.id }, headers: authA });
  expect(res.statusCode).toBe(200);
});
```

Run: **red**, same TypeError at `:58`. Reproduced on demand. The "intermittent" bug is 100% reproducible once you know the trigger is a null amount.

**Root cause:** `export.ts:58` calls `.toFixed()` on `t.amount` without handling that `amount` is nullable in the schema, so a transfer with a null amount (which the pending state legitimately produces) crashes the export.

**Class check:** grepped for `.amount` across the codebase.

```
src/reports/export.ts:58    t.amount.toFixed(2)      ← the reported crash
src/reports/summary.ts:31   sum += t.amount          ← null → NaN, silent
src/jobs/reconcile.ts:44    t.amount.gt(0)           ← same crash, different job
```

Three instances of the same class. `export.ts:58` crashes loudly. `reconcile.ts:44` crashes the same way in a background job — probably logged and ignored. `summary.ts:31` is worse: it doesn't crash, it silently produces `NaN`, so **report totals are wrong whenever a pending transfer is included** and nobody gets an error. The reported bug was the least harmful of the three.

**Fix:** decide what a null amount means in each context. It means "not yet settled," so:

```ts
// export.ts:58 — show pending explicitly, don't crash or fake a zero
amount: t.amount == null ? 'pending' : t.amount.toFixed(2)

// summary.ts:31 — exclude pending from a settled-total, don't NaN it
sum += t.amount ?? new Decimal(0)

// reconcile.ts:44 — skip pending, it's not reconcilable yet
if (t.amount != null && t.amount.gt(0)) { ... }
```

Each is the *right* handling for that context, not a blanket `?? 0` that would have made the summary silently wrong in a new way.

**Proof:** the export test passes. Added one test per instance — a summary test asserting a pending transfer doesn't corrupt the total (this one was green-on-broken until I asserted the actual total, then red, confirming the NaN bug), and a reconcile test for the skip. Full suite: 87 pass, 0 fail. `npm run build` clean.

**Security dimension:** the `summary.ts` NaN is the one that matters. It's an integrity bug in financial reporting reachable by any user who has a pending transfer at report time — which is normal, not adversarial, but it means totals shown to customers can be silently wrong. Not an exploit, but it's the kind of correctness-is-integrity issue worth flagging. Handing the summary case to `security-test-writer` for the boundary cases (all-pending, mixed, exactly-one-pending).

---

## What this run got right

- Read the "intermittent, can't reproduce" symptom correctly as data-dependent, and **made it 100% reproducible** by identifying the trigger — a null amount.
- Wrote the failing test first and confirmed it was red for the right reason.
- Stated the root cause in one sentence, at a line.
- **Checked the class** and found two more instances — including a silent data-corruption bug worse than the reported crash.
- Fixed each instance with context-appropriate handling, and explicitly rejected the lazy blanket `?? 0`.
- Proved it with the suite actually run, and noted that the summary test was green-on-broken until the assertion was strengthened.
- Flagged the integrity dimension of the silent-NaN bug without overstating it as an exploit.
