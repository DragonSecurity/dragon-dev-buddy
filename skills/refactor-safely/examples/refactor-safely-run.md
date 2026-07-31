# Worked example: decomposing a 200-line handler under a net

---

**User:** `handleCheckout` in src/checkout.ts is 210 lines and nobody wants to touch it. Clean it up.

**Assistant:** "Nobody wants to touch it" plus payments means the first job isn't refactoring — it's building the net, because this is exactly the kind of function where a refactor silently drops a control. Let me look before I touch.

*(reads `src/checkout.ts`, existing `test/checkout.test.ts`)*

Existing coverage: two happy-path tests. That's not a net for a 210-line payment handler. Building one first.

**Security controls I can see in the target, inventoried before changing anything:**

1. `:12` — auth check, rejects if no session
2. `:19` — the cart's line prices are re-fetched from the DB, not taken from the request (server-derived pricing)
3. `:34` — idempotency key checked against prior charges
4. `:41` — amount validated as positive, currency matched to account
5. `:88` — audit write recording who checked out what
6. `:145` — the whole charge-and-fulfil block is in a transaction

Every one of these is a line a "cleanup" could drop. Each gets a test asserting it still happens, before I refactor.

*(writes characterization + control tests)*

```ts
// Characterization: pin current behavior, including a quirk
it('rounds a 3-decimal line price DOWN, as it currently does', async () => {
  // This looks like a bug (should probably round half-up). Pinning it as-is;
  // fixing it is a separate change, not part of this refactor.
  const res = await checkout(cartWith({ price: 1.005, qty: 1 }), sessionA);
  expect(res.charged).toBe(100);   // 1.00, not 1.01 — current behavior
});

// Control tests, one per inventoried item
it('rejects checkout with no session', async () => {
  await expect(checkout(cartA, null)).rejects.toThrow(Unauthorized);
});
it('ignores client-supplied prices and uses DB prices', async () => {
  const res = await checkout(cartWith({ price: 0.01, qty: 1, dbPrice: 5000 }), sessionA);
  expect(res.charged).toBe(5000);   // client said 1 cent; server charged the real price
});
it('rejects a replayed idempotency key', async () => {
  await checkout(cartA, sessionA, { idem: 'k1' });
  await expect(checkout(cartA, sessionA, { idem: 'k1' })).rejects.toThrow(Duplicate);
});
it('writes an audit record on success', async () => {
  await checkout(cartA, sessionA);
  expect(await auditCount({ action: 'checkout', actor: sessionA.userId })).toBe(1);
});
// ...currency validation, transaction rollback-on-failure
```

Net: 6 control tests + 9 characterization tests. **All green against the current code** — confirmed before touching anything.

**Refactor, small steps, net after each:**

1. Extract `:12-18` → `requireSession(req)`. Net green.
2. Extract `:19-33` → `priceCart(cart)` — the server-side pricing. Net green. *(The control test for DB pricing is what makes this extraction safe — it proves `priceCart` still ignores client prices.)*
3. Extract `:34-40` → `assertNotReplayed(idemKey)`. Net green.
4. Extract `:41-52` → `validateAmount(total, account)`. Net green.
5. Extract `:145-200` → `chargeAndFulfil(...)`, keeping the transaction boundary *inside* the extracted function. Net green — the rollback test is what confirms the transaction survived the move.
6. `handleCheckout` is now 38 lines that read as a sequence of named steps.

**Behavior held:** 15 tests green, same as before. All 6 controls still asserted and passing — critically, the transaction boundary and the server-pricing check survived their extractions, which the control tests prove rather than my eyeballing. `npm run build` clean, `npm run lint` clean.

**Goal met:** the 210-line function is 38 lines; each concern is a named, independently testable function. The next person can touch `priceCart` without reading the charge logic.

**Deliberately not done:** the 3-decimal rounding quirk (pinned in a characterization test, looks like a bug, belongs to `debug-and-fix` as its own change — I didn't fix it mid-refactor because a rounding change in a payment path needs its own review, not a free ride on a "cleanup"). Also left: `chargeAndFulfil` is still 55 lines and could decompose further, but that's a second refactor, not this one.

---

## What this run got right

- Recognized that the real first task was the net, and refused to refactor a payment handler over two happy-path tests.
- **Inventoried the security controls before touching anything** and gave each a test — the step that turns "I think it still works" into proof.
- Pinned a suspected bug as current behavior rather than fixing it mid-refactor, and said why.
- Small steps, net after each, so any red would point at one transformation.
- Called out that the control tests are specifically what made the risky extractions (server-pricing, transaction boundary) safe.
- Met a concrete goal (210 → 38 lines, testable units) and named what it deliberately left for separate changes.
