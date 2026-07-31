# Worked example: review of a webhook receiver

Two findings shown of five. Note the trust position, the chain, and the coverage statement.

---

# Security review: `src/webhooks/settlement.ts`
2026-03-16 · 5 findings · 1 critical, 1 high, 2 medium, 1 low

**Trust position:** The caller controls the entire HTTP request — body, headers, timing, and how many times it is sent. Nothing about the caller is authenticated by the transport. This handler runs with full database write access and can mark transfers as settled, which releases funds.

That gap is the whole review: an anonymous internet caller reaching a function that moves money.

## Findings

### F1 — Signature comparison is not constant-time, and runs on the parsed body   **CRITICAL**

**Attack:** An anonymous attacker posts to `/webhooks/settlement` with a forged body and a guessed signature. `===` on strings short-circuits at the first differing byte, so response time correlates with how many leading bytes are correct. Roughly 64 × 16 requests recovers the signature one byte at a time. With a valid signature, the attacker posts `{"transfer_id": "...", "status": "settled"}` and the system releases funds for a transfer that never settled.

The second half is worse than the timing leak: the HMAC is computed over `JSON.stringify(req.body)`, not the raw bytes. Fastify has already parsed and re-serialized by then, so key order, unicode escaping and number formatting can all differ from what the bank signed. An attacker who can make two different byte sequences parse to the same object gets a signature that validates for a body the bank never sent.

**Evidence:** `src/webhooks/settlement.ts:18-27`

```ts
const expected = crypto
  .createHmac('sha256', process.env.BANK_WEBHOOK_SECRET!)
  .update(JSON.stringify(req.body))          // re-serialized, not raw
  .digest('hex');

if (req.headers['x-bank-signature'] !== expected) {   // not constant-time
  return reply.code(401).send({ error: 'bad signature' });
}
```

**Why it rates Critical:** Unauthenticated, remote, and the outcome is unauthorized movement of money. `exposure: public` means no capping, and `data_sensitivity: credentials` would raise it if it were not already at the top.

**Fix:** Two changes, both required.

```ts
// 1. Capture raw bytes before parsing — Fastify needs to be told.
fastify.addContentTypeParser('application/json', { parseAs: 'buffer' },
  (req, body, done) => { req.rawBody = body; done(null, JSON.parse(body.toString())); });

// 2. Verify over raw bytes, in constant time, with a length guard.
const sig = Buffer.from(String(req.headers['x-bank-signature'] ?? ''), 'hex');
const expected = crypto
  .createHmac('sha256', process.env.BANK_WEBHOOK_SECRET!)
  .update(req.rawBody)
  .digest();

if (sig.length !== expected.length || !crypto.timingSafeEqual(sig, expected)) {
  return reply.code(401).send({ error: 'bad signature' });
}
```

`timingSafeEqual` throws on length mismatch rather than returning false, which is why the length check comes first — without it, a short signature is a 500 instead of a 401, which is its own oracle.

Swapping only the comparison is not enough. The re-serialization bug survives a constant-time compare, and it is the half that does not need thousands of requests.

---

### F3 — Settlement events are replayable   **HIGH**

**Attack:** Any party who observes one valid webhook — a proxy, a log aggregator, anyone with read access to the request logs this service writes at `settlement.ts:41` — can resend it verbatim. The signature is still valid, because it covers the body and the body has not changed. `markSettled` at `:58` is not idempotent: it appends a credit row each time. Ten replays credit the customer ten times.

**Evidence:** `src/webhooks/settlement.ts:52-61`

```ts
const { transfer_id, amount } = req.body;
await db.transfer.update({ where: { id: transfer_id }, data: { status: 'settled' } });
await db.ledgerEntry.create({ data: { transferId: transfer_id, amount, type: 'credit' } });
```

No check on current status, no event id, no timestamp window.

**Why it rates High:** Needs the attacker to have seen one legitimate request, which is a real precondition — but this service logs full request bodies, so the population who has seen one is larger than it looks. Outcome is unauthorized credit, so impact is High.

**Fix:** Persist the bank's event id with a unique constraint and make the whole thing one transaction:

```ts
await db.$transaction(async (tx) => {
  await tx.webhookEvent.create({ data: { id: req.body.event_id, source: 'bank' } }); // unique PK
  await tx.transfer.update({
    where: { id: transfer_id, status: 'pending' },   // no-op if already settled
    data: { status: 'settled' },
  });
  await tx.ledgerEntry.create({ data: { transferId: transfer_id, amount, type: 'credit' } });
});
```

A duplicate event id violates the primary key, the transaction rolls back, and the handler returns 200 — which is what the bank needs to stop retrying. Also reject events whose timestamp is more than five minutes old, so a captured request does not stay useful indefinitely.

---

## The chain

**F1 + F3 together are worse than either alone.** F1 gets an attacker one valid signature. F3 means one valid signature is reusable forever, on any amount they have seen. The pair turns a single successful forgery into unlimited credits. Fixing F1 without F3 leaves the replay path open to anyone who reads a log; fixing F3 without F1 leaves forgery available. Ship them together.

## Worth cleaning up

- `settlement.ts:41` — logs the full request body including the signature header. Not exploitable on its own, but it is what makes F3's precondition realistic. Redact the header and the amount.
- `settlement.ts:12` — `BANK_WEBHOOK_SECRET!` with a non-null assertion. If the env var is missing, the HMAC is computed over `undefined` and every request fails closed, which is the right direction, but the error will be baffling. Fail at boot instead.

## Coverage

**Read closely:** `src/webhooks/settlement.ts`, `src/db/ledger.ts`, the Fastify content-type parser registration in `src/server.ts`.

**Skimmed:** `src/jobs/reconcile.ts` — it consumes the same ledger entries and may have its own idempotency assumptions. Worth its own pass.

**Not read:** the bank's own signing documentation. I have assumed HMAC-SHA256 over the raw body with a hex-encoded header, because that is what the code implements. **Verify this against their docs before shipping the fix** — if they sign a canonical form or include the timestamp in the signed payload, F1's fix is wrong in its details.

---

## What this run got right

- The trust position is stated first and every finding refers back to it.
- F1 found the *second*, worse bug (re-serialization) while investigating the first, and explained why the obvious one-line fix is insufficient.
- The fix code includes the non-obvious detail — `timingSafeEqual` throwing on length mismatch — rather than a snippet that would introduce a new oracle.
- F3's severity reasoning names the specific reason the precondition is realistic *in this codebase*: it logs bodies.
- The chain is called out explicitly with a shipping instruction.
- The coverage statement admits an assumption that could invalidate the recommended fix, and says to check it.
