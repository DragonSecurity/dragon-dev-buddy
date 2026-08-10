# Worked example: interviewing a document-sharing feature into a threat model

---

**User:** We want to let people share a document with someone outside the org. Grill me — I don't want to find out in review that I'd assumed something stupid.

**Assistant:** Sharing across an org boundary is a decision tree, not a feature, so I'll work it in rounds. Reading first — nothing goes in a question that the repo can answer.

*(reads `.dragon-buddy/config.json` — `security.exposure: "public"`, `security.data_sensitivity: "pii"`, `security.auth_model: "session cookie + org membership"`; reads `src/documents/`, `src/auth/`, the Prisma schema)*

What the code already settles, so I won't ask: documents are org-scoped through a `orgId` column enforced in a Prisma middleware; there is no existing concept of a user outside an org; uploads go to S3 with server-side encryption and are served through a signed-URL endpoint, not directly.

## Round 1 — the shape of a share

**Q1 — Who is a share addressed to?**

An email address, a link that anyone holding it can open, or both. This is the root of the tree: almost every question below it changes meaning depending on the answer. A link-only model has no recipient identity at all, which removes the accountability questions and adds the leakage ones.

**Recommended:** Both, but ship the emailed share first. Your `data_sensitivity` is `pii` and these documents are where customers put it, so the question you will eventually be asked is whose data went to whom — and a link-only share has no recipient identity with which to answer it.

**Q2 — Does the recipient end up with an account?**

Full account in your org, a lightweight external identity scoped to the documents shared with them, or no identity at all (the link is the credential).

**Recommended:** A lightweight external identity. A full account puts a non-employee inside your org boundary, which your entire authorization model — a middleware that filters by `orgId` — currently assumes cannot happen.

**Q3 — What can a share grant?**

View, view plus comment, or edit.

**Recommended:** View only for v1. Edit means an external principal writing into org-scoped data, and that is a second trust boundary you don't need to open in the same release.

**Q4 — Can a share be revoked, and does revocation have to be immediate?**

"Immediate" is a real constraint: it rules out anything where access is carried in a token the server does not check against state at request time.

**Recommended:** Yes, and immediate. The realistic trigger is a recipient's mailbox being compromised, and at that point "access stops within 24 hours" is not a remediation, it is a notification you have to make anyway.

---

**User:** 1 — both, and yes email first. 2 — lightweight external identity, agreed, definitely not full accounts. 3 — view plus comment actually, comments are the whole point. 4 — yes immediate.

**Assistant:** Q3 going to comment reopens a branch I'd pruned: an external principal now writes a row into org-scoped data, so authorship, visibility and retention of that row are all decisions, and they weren't reachable a round ago.

I need one fact before I can recommend anything about the file itself, and I'm not going to ask you for it. Dispatching a subagent to establish how the signed-URL endpoint scopes access today — whether the signature is bound to a session or is a bare time-limited S3 URL. Two decisions hang off that answer — how long a share link should live, and whether an external recipient can download at all — so they stay unnumbered and out of this round. The rest of the frontier goes now.

## Round 2 — identity, comments, revocation

*Settled in round 1: shares are addressed by email with links to follow; recipients get a scoped external identity; view + comment; revocation must be immediate.*
*Dispatched: how the document-serving endpoint scopes access today. Link lifetime and download-versus-stream wait on it.*

**Q5 — How does an external recipient prove they are the addressed person?**

The invite email carries a one-time token that establishes the external identity, and thereafter they have a session. Or every access re-verifies by emailing a code. Or the token in the link *is* the credential, forever.

**Recommended:** One-time token that mints a scoped session, token single-use and 72-hour expiry. Reusable invite links get forwarded, and a forwarded credential is indistinguishable from the intended recipient.

**Q6 — Is a comment from an external recipient visible to other external recipients of the same document?**

Only to org members, or to everyone on the share.

**Recommended:** Only to org members in v1. Two external parties on the same document who don't know about each other is an information disclosure you'd be shipping without noticing — the comment author's name and email is the disclosure, not the comment.

**Q7 — What happens to comments when the share is revoked?**

Comments stay and remain attributed, comments stay but are anonymised, or comments are removed.

**Recommended:** Stay, attributed. Revocation removes future access, not the record — and an audit trail that deletes itself when access is cut is not an audit trail.

**Q8 — Does revoking a share invalidate the recipient's existing session immediately?**

This is where "immediate" gets expensive. A stateless session survives revocation until it expires.

**Recommended:** Yes — external sessions carry a share reference, checked against share state on every request. External sessions are low-volume, so the per-request check costs nothing that matters.

---

**User:** 5 — agreed, but 72 hours is too long, make it 24. 6 — agreed. 7 — agreed. 8 — yes, do the per-request check.

**Assistant:** *(subagent reports: `GET /documents/:id/content` issues a 15-minute pre-signed S3 URL and redirects to it. The signature is not bound to the session — anyone holding the S3 URL within those 15 minutes gets the file, no authentication involved.)*

That fact creates a question I wouldn't have known to ask, and rewrites one I was already holding. The new one: Q8's per-request check does not cover the file bytes at all — once the redirect is issued, revocation cannot reach the URL that was handed out. The rewritten one: "how long should a share link live" was the wrong question, because the lifetime that actually bounds access is the 15 minutes on the S3 signature, which is not per-share and not yours to set at share time. Good thing it never went out with a number on it. Frontier now:

## Round 3 — the file bytes

*Settled in round 2: single-use 24-hour invite token minting a scoped session; comments visible to org members only; comments survive revocation, attributed; external sessions checked against share state per request.*
*Fact established: content is served as a 15-minute unauthenticated pre-signed S3 URL — the per-request revocation check in Q8 stops at the redirect.*

**Q9 — Is a 15-minute unrevocable window on the file bytes acceptable for external recipients?**

Accept it and document it, shorten the signature to something like 60 seconds, or proxy the bytes through the app so every range request passes the revocation check.

**Recommended:** Proxy for external recipients, keep pre-signed URLs for org members. You have already decided revocation must be immediate; a 15-minute hole in it makes that decision false rather than approximate. Org members keep the fast path because the threat is different.

**Q10 — Can an external recipient download, or only view in-browser?**

Download is a decision, not a technical detail: it is the point at which your control over the document ends.

**Recommended:** Allow it, and log it. Blocking download against a determined recipient does not work, and pretending otherwise means you neither prevent nor record the copy. A download event on the share's audit trail is worth more than a disabled button.

**Q11 — How long does a share live if nobody sets an expiry?**

Forever until revoked, or a default expiry.

**Recommended:** 90-day default, extendable. Shares that never expire accumulate silently and become the access nobody remembers granting — the thing every access review finds.

---

**User:** 9 — proxy, yes, I hadn't thought about the redirect at all. 10 — agreed, log it. 11 — 90 days, but make it configurable per org.

**Assistant:** Frontier is empty. Walking the settled decisions once more for anything they made reachable: the per-org expiry default in Q11 is a new org-level setting, but who may change it is an existing decision — your org admin role already governs org settings, so it inherits and does not open a branch.

## Settled design: external document sharing

**Decisions:**
- Shares are addressed by email; a link-only mode follows later. Confidential data gets an addressed recipient first.
- Recipients get a lightweight external identity scoped to shared documents, never an org account — the `orgId` middleware is the whole authorization model and an external principal must not be inside it.
- View plus comment. No edit in v1.
- Invite is a single-use token, 24-hour expiry, minting a scoped session. Reusable links get forwarded and a forwarded credential is indistinguishable from the recipient.
- Revocation is immediate: external sessions carry a share reference checked against share state on every request, and content is proxied for external recipients so the check covers the bytes.
- External comments are visible to org members only. Two external parties on one document must not learn about each other.
- Comments survive revocation, attributed. The record is not access.
- Download is allowed and logged. Shares default to 90 days, per-org configurable.

**Facts established:** org scoping is enforced by Prisma middleware on `orgId`; there is no pre-existing external-user concept; content is served as a 15-minute unauthenticated pre-signed S3 URL — found by subagent, not asked, and it is what forced Q9.

**Ruled out:** full org accounts for recipients (breaks the authorization model); edit access in v1 (second trust boundary, later release); cross-visible external comments (discloses recipients to each other); blocking download (unenforceable, and it costs the log entry).

**Open by choice:** the anonymous-link mode — deferred, and it reopens Q6 and Q11 when it lands, because a link has no addressee to scope visibility or expiry to.

**Assumptions still standing:** that comment bodies are not themselves confidential enough to need a separate retention decision. Unverified — flag it if comments will carry customer data.

Confirm this matches your understanding and I'll hand it to `threat-model`, with the org/external boundary and the proxy decision as named trust boundaries rather than ones it has to infer from code.

**User:** Confirmed. Go.

---

## What this run got right

- Read the config and the code before the first question, and said out loud what that removed from the interview.
- Asked the whole frontier in each round, numbered, every question with a recommendation and a reason — so the user settled four decisions in one reply, three times.
- Kept each round internally independent. The comment-visibility question (Q6) waited for the identity model (Q2) to settle, because "visible to whom" is meaningless before you know what a recipient is.
- Dispatched an environment fact to a subagent instead of asking, and did not block on it: Q5 through Q8 went out while it ran.
- Let the fact reshape the tree. The pre-signed URL discovery created Q9, which nobody would have asked, and it is the finding that made the revocation decision actually true rather than merely stated.
- Kept the waiting questions unnumbered until the fact landed. That is what let the discovery rewrite one of them, instead of contradicting a number already sitting in front of the user.
- Named the branch that reopened when Q3 moved from view-only to view-plus-comment, rather than quietly carrying the old pruning forward.
- Walked the settled decisions once more before declaring the frontier empty, and said why the one new item did not open a branch.
- Stated the surviving assumption instead of letting it pass as settled, and handed the boundaries forward with their reasoning attached.
