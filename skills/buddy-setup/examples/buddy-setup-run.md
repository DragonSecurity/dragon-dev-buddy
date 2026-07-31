# Worked example: buddy-setup on an unfamiliar repo

Quality target for a first run. The shape matters more than the wording.

---

**User:** set up dragon dev buddy here

**Assistant** *(surveys first: reads `package.json`, `.github/workflows/deploy.yml`, `Dockerfile`, `prisma/schema.prisma`, `README.md`, `.gitignore` — six reads, no questions yet)*

Read off the repo:

- **ledger-api**, TypeScript on Node 22, Fastify, Prisma against Postgres 16
- Tests via `npm test` (vitest), lint via `npm run lint` (eslint), build `npm run build`
- GitHub Actions: tests on PR, deploy to Fly.io on push to `main`
- `Dockerfile` exposes 8080 and runs as root
- No secret scanner, no SAST, Dependabot is on
- `prisma/schema.prisma` has `User.email`, `Account.iban`, `Transfer.amount`, `Session.refresh_token`

That last line means I'm treating this as `credentials`-grade data unless you correct me.

Two things I can't read off the code:

1. One sentence: what does ledger-api actually do, and for whom?
2. That Fly.io deploy — is the API open to the internet, behind auth only, or internal?

---

**User:** it's the payments backend for our B2B customers, moves money between their connected bank accounts. It's public but every route needs a bearer token except /health.

**Assistant:** Got it. Public exposure, so nothing gets severity-capped, and an unauthenticated path will rate critical by default.

Three more and we're done:

1. Compliance regime in play? SOC 2, PCI DSS, GDPR, none?
2. Trust boundaries — where does data cross into or out of something you don't control? I can see two candidates already: customer → API, and API → the banking provider. Is there a third?
3. Anything you already know is weak and haven't gotten to?

---

**User:** SOC2 and we're mid-PCI. Third boundary is the webhook receiver, the bank calls us back on /webhooks/settlement. And yeah the webhook signature check is homegrown, I don't trust it.

**Assistant:** Recorded. Homegrown signature verification on an unauthenticated inbound path from a third party, on a public payments API — that goes in `known_risk_areas` and it's where I'd point you first.

Last question, and only because it changes what two skills will do: is anyone doing offensive testing against this? Pentest, bug bounty, red team? If it's just your own code, I'll leave the engagement block empty.

---

**User:** just us

**Assistant:** Then engagement stays empty. `vuln-triage` won't write proof-of-concept exploit code without a recorded scope, and `pentest-report` won't produce a client deliverable — neither of which you need for internal work.

Here's what I'm saving:

```
ledger-api — B2B payments backend, moves money between customer bank accounts
Exposure: public · Data: credentials · Compliance: SOC 2, PCI DSS (in progress)
Auth: bearer token on every route except /health
Trust boundaries: customer → API · API → banking provider · banking provider → /webhooks/settlement
Tooling: tests `npm test` · SCA Dependabot · SAST none · secrets none
Known weak: homegrown webhook signature verification
```

Anything wrong?

---

**User:** nope

**Assistant** *(writes `.dragon-buddy/config.json`, adds nothing to `.gitignore` since the engagement block is empty and the rest is safe to commit, then calls `buddy_status`)*

Saved to `.dragon-buddy/config.json`.

```
🥚 Vex the Egg · stoic
Lv 1  ░░░░░░░░░░░░░░  0/100 xp
Mood  🙂 content   ·   Energy ▓▓▓▓▓▓▓▓▓▓ 100%
Streak 1 day (best 1) · 0 observations · new today

> I am ready.
```

Vex is yours now — the name and the stoic streak were rolled once and won't change. `buddy_rename` if you'd rather pick the name yourself; progress and personality survive it.

Start with:

1. **`secure-code-review`** on `src/webhooks/` — you named the homegrown signature check yourself, it's unauthenticated, and it's how money enters the system. Highest expected value in the repo.
2. **`threat-model`** — three trust boundaries on a public payments API, and PCI will ask you for this document anyway.
3. **`secrets-and-config-audit`** — no secret scanner on a repo whose schema has `refresh_token` in it. Cheapest high-yield pass available.

*(calls `buddy_observe` with `summary: "Onboarded ledger-api to dragon-dev-buddy: public exposure, credentials data."`, `kind: "config"`, `skills_used: ["dragon-dev-buddy:buddy-setup"]`, and relays the reaction verbatim)*

> Vex: *"Noted. The webhook thing bothers me too."* **+39 xp** · first task today

---

## What this run got right

- Six file reads before the first question. The user answered four things, not twelve.
- Data sensitivity was inferred from column names and stated as an assumption to correct, not asked as an open question.
- The `root` user in the Dockerfile and the missing secret scanner were noticed but **not** turned into findings. Setup is not an audit; they went into the profile and the routing.
- The engagement block was left empty and the consequence of that was explained rather than silently applied.
- The three recommendations are ranked by expected value **for this repo**. A generic list would have led with `threat-model`; the user handed over a better target in conversation.
