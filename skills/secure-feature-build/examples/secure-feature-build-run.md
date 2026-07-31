# Worked example: a file-sharing feature, built from abuse cases

Condensed — the spec in full, the code abbreviated. The point is that the defenses were in the spec before the first line.

---

**User:** add a feature where users can share a report with someone outside their org via a link

**Assistant:** That's two feature types at once — *sharing* and, since a report can be exported, *file access*. So I'm pulling abuse cases from both. Let me confirm the requirement first, then write the spec before any code.

**Requirement:** A user generates a link to one of their reports. Anyone with the link can view that report, read-only, without an account, until the link expires or is revoked.

Right? And two questions the design turns on: should the recipient see it in the browser or download it, and is there a maximum sensible lifetime — a day, a week?

---

**User:** view in browser, and default 7 days but let them pick up to 30

**Assistant:** Here's the spec. Note the abuse cases are the middle of it, not an appendix.

```markdown
# Feature spec: external report share links

## Requirement
A report owner generates a share link. Anyone holding the link views that one
report, read-only, unauthenticated, until it expires (default 7d, max 30d) or
the owner revokes it.

## Abuse cases
- A1: Attacker guesses or enumerates a share link → reads arbitrary reports
- A2: Link is scoped to report X but is manipulated to read report Y
- A3: Link works forever; a leaked link is permanent access to financial data
- A4: Owner revokes, recipient keeps reading
- A5: Recipient escalates read-only to export/edit
- A6: Report is served from the app origin → SVG/HTML in report content runs
      as the app, with the viewer's cookies in scope
- A7: The share link itself becomes an oracle: valid-but-expired vs never-existed
      reveals which report ids exist
- A8: Anyone can enumerate who a report is shared with

## Requirements (defenses as behavior)
- R1: Link token is 128-bit random, not derived from the report id (closes A1)
- R2: The token maps server-side to exactly one report id; the client sends only
      the token, never a report id (closes A2)
- R3: Every link row has expires_at; the viewer checks it at read time, not just
      at creation; max 30d enforced server-side (closes A3, A4 via short window)
- R4: Revocation sets revoked_at; the read path checks it every time (closes A4)
- R5: The share read path is a distinct handler that can only read, and only the
      one report the token maps to; it shares no code with the authenticated
      report routes that can export (closes A5)
- R6: Shared reports render on a separate origin (share.ledger.example) with no
      auth cookies scoped to it and a strict CSP (closes A6)
- R7: Expired, revoked, and never-existed tokens return an identical response
      (closes A7)
- R8: No endpoint lists shares for a report to an unauthenticated caller (A8)

## Data and trust
share_links(token PK, report_id FK, created_by, expires_at, revoked_at, created_at)
Trust boundary: the share read path treats the token as the *only* input and the
report_id as derived. The owner's identity never travels with the link.
Minimum privilege: the share handler's DB role can SELECT reports and share_links
and nothing else — it cannot export, write, or read other tables.

## Acceptance tests
- Positive: owner creates link → recipient reads the correct report, no auth
- Negative:
  - random token → identical 404 to an expired one (R1, R7)
  - a token for report X, with ?report=Y appended → still returns X (R2)
  - expired link → 404, no data (R3)
  - revoked link → 404 immediately after revocation (R4)
  - POST/export/edit against the share path → 405/404, never succeeds (R5)
  - a report whose title contains <script> → served escaped, on the share
    origin, with CSP blocking inline execution (R6)
```

Confirm this and I'll build it. The one I want you to look at is R5 — I'm making the share path a physically separate handler rather than reusing the report route with a "isShare" flag, because a shared flag is one `if` away from leaking export. Costs a little duplication, removes a whole class of bug.

---

**User:** yeah that's right, R5 especially. build it

**Assistant** *(implements to spec — the token generation, the separate read-only handler, the expiry/revocation checks, the separate-origin serving; then the tests)*

```ts
// src/share/create.ts
export async function createShareLink(ownerId: string, reportId: string, days: number) {
  const ttl = Math.min(Math.max(days, 1), 30);              // R3: server clamps
  const token = crypto.randomBytes(16).toString('base64url'); // R1: 128-bit random
  const report = await scopedReport(ownerId).findUnique({ where: { id: reportId } });
  if (!report) throw new NotFound();                         // owner must own it to share it
  const expiresAt = new Date(Date.now() + ttl * 86_400_000);
  await db.shareLink.create({ data: { token, reportId, createdBy: ownerId, expiresAt } });
  return `https://share.ledger.example/s/${token}`;          // R6: separate origin
}

// src/share/read.ts — physically separate, read-only, its own DB role (R5)
export async function readShared(token: string) {
  const link = await shareDb.shareLink.findUnique({ where: { token } });
  // R7: one indistinguishable failure for missing / expired / revoked
  if (!link || link.revokedAt || link.expiresAt < new Date()) throw new NotFound();
  return shareDb.report.findUnique({           // R2: report id is derived, never from the caller
    where: { id: link.reportId },
    select: { title: true, amount: true, rows: true },      // projection, never the whole row
  });
}
```

## Build notes

**Handled:** A1–A7. Each has a negative test that goes red if you remove the corresponding line — I checked R7 specifically by making expired and missing return different codes and watching the test fail.

**Deferred:** A8 (share enumeration) — there is no share-listing endpoint at all yet, so there is nothing to leak. When you add an owner-facing "manage shares" view, it needs its own authorization and this abuse case comes back. Noted in the spec's follow-ups.

**One thing I want a second opinion on:** R6 depends on `share.ledger.example` genuinely having no auth cookies scoped to it. That's a DNS and cookie-domain configuration fact I asserted in code but cannot verify from here. It's exactly the kind of assumption that's true today and silently false after someone sets a cookie on `.ledger.example` (note the leading dot). Flagging it for `hardening-playbook` on the cookie config, and I'd run `secure-code-review` over the share handler as an adversarial pass — I wrote these defenses, so I'm the wrong person to certify them.

---

## What this run got right

- The abuse cases were written **before** the code, and every requirement traces to one.
- It recognized the feature as two types (sharing + file access) and pulled from both.
- R5 chose structural impossibility (a separate handler) over a runtime flag, and explained the tradeoff.
- Defenses are phrased as behavior and each has a negative test whose red state was checked.
- The deferred abuse case (A8) was deferred with a *reason it's currently safe* and a trigger for when it won't be.
- It named an assumption it couldn't verify (the cookie scope on the share origin) and routed it, rather than asserting the feature was airtight.
- It asked for an adversarial review of its own work.
