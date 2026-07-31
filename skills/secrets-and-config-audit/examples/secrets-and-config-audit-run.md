# Worked example: a scan that finds a live key

---

**User:** scan this repo for secrets before we open-source it

**Assistant:** Going public changes the assessment, so I'm scanning full history, not just the working tree — anything ever committed becomes public when the repo does.

Before I start: who can rotate credentials for this project, and are they around today? If I find something live, that's the first thing that needs to happen.

---

**User:** me, and yes

**Assistant** *(runs `gitleaks detect` over full history, `gitleaks detect --no-git`, then greps the custom patterns and the accumulation files)*

# ⚠ Immediate action — stop and rotate

**`.env.staging` is committed and still in the working tree.** Two credentials:

**S1 — Stripe restricted key.** Prefix `rk_live_`, so it is a live-mode key, not test. Added in commit `a3f8c21`, 2026-01-22 by `alanis`, 54 days ago. Still present in the tree.

I have **not** tested whether it authenticates — that would mean using a key I found, which shows up in your audit log as exactly that. Here's how to check without me:

1. **Rotate now.** Stripe dashboard → Developers → API keys → roll `rk_live_...`. Stripe supports a grace period on roll, so nothing breaks mid-flight.
2. **Then check scope.** Restricted keys carry specific permissions. Look at what this one had before you finish rolling it — read-only on one product is a very different day than write on charges.
3. **Then check Events and Logs**, filtered to that key, over 2026-01-22 to today. You're looking for requests from source IPs you don't recognize.

**S2 — Postgres password.** `DATABASE_URL` in the same file, same commit. This one has no scope limiting it at all.

1. Create a new Postgres role with identical grants, migrate the connection string, drop the old role. Don't just change the password on the shared account — you won't know what broke.
2. Check connection logs for source addresses over the same window.

**The repo is currently private, so this is recoverable.** Had it been public for any part of those 54 days, I'd tell you to treat both as compromised regardless of what the logs say — scrapers index public pushes within minutes.

I'm pausing the config audit until you tell me these are rotated.

---

**User:** rotated both. stripe key was read-only on one product and the logs are clean. postgres role dropped.

**Assistant:** Recording S1 as **Critical, contained** — the restricted scope limited the blast radius, and a clean log over a private-repo window is good evidence. S2 as **Critical, rotated** — unscoped, so scope did nothing for you there; the clean connection log is what closes it.

One thing worth naming: the scope on the Stripe key is why this was a bad afternoon rather than a bad quarter. That was a good decision made 54 days ago by someone who wasn't thinking about this scenario.

Continuing with config.

---

## Configuration

### C1 — `pull_request_target` checks out the PR head   **CRITICAL**

`.github/workflows/preview.yml:8-14`

```yaml
on: pull_request_target
jobs:
  preview:
    steps:
      - uses: actions/checkout@v4
        with: { ref: "${{ github.event.pull_request.head.sha }}" }
      - run: npm ci && npm run build
        env: { FLY_API_TOKEN: "${{ secrets.FLY_API_TOKEN }}" }
```

**Consequence:** `pull_request_target` runs with repository secrets available. Checking out the PR head then running `npm ci` executes install scripts from a stranger's branch, with your Fly deploy token in the environment. Anyone who opens a pull request can exfiltrate it and deploy to your infrastructure.

**This becomes exploitable by the entire internet the moment you open-source.** It is the finding that should block the launch.

**Fix:** switch to `on: pull_request` (no secrets, no deploy) or keep `pull_request_target` and check out the **base** ref only, never the head. If you need preview deploys on forks, run the build in a separate workflow triggered by `workflow_run` with the artifact, not the source.

### C2 — Container runs as root   **MEDIUM**

`Dockerfile:1-16` — no `USER` directive.

**Consequence:** ledger-api parses untrusted JSON and writes uploaded exports to disk. A write-primitive bug that would otherwise be contained becomes host-level on the Fly machine, which also runs the export worker with S3 credentials in its environment.

**Fix:** `RUN adduser -D app` and `USER app` before `CMD`. Port 8080 is above 1024, so binding still works unprivileged.

### C3 — Actions pinned by tag   **MEDIUM**

All four workflows use `actions/checkout@v4`, `superfly/flyctl-actions@v1`, etc.

**Consequence:** tags are mutable. Whoever controls those repositories can move `v4` and change what runs in a pipeline holding your deploy token. Not hypothetical; it has happened to popular actions.

**Fix:** pin to commit SHAs. Dependabot updates SHA pins as well as tags, so this costs nothing ongoing.

## Secret handling

Secrets reach production via `fly secrets set`, which is correct — injected at runtime, never in an image layer. The failure was entirely upstream of that: `.env.staging` was created for local development, and `.gitignore` has `.env` but not `.env.*`. One character in a gitignore is the whole story.

CI secrets are GitHub encrypted secrets, correct, but C1 makes that irrelevant for anyone who opens a PR.

## Prevention

The single highest-value change: **`.gitignore` line 4, `.env` → `.env*` with `!.env.example`.** That one edit would have prevented both credential findings.

Then, before going public:

```yaml
# .pre-commit-config.yaml
- repo: https://github.com/gitleaks/gitleaks
  rev: v8.18.4
  hooks: [{ id: gitleaks }]
```

and gitleaks in CI on every push. Local hooks get skipped with `--no-verify`; CI does not.

**Do not rewrite git history for this.** Both credentials are rotated, which makes them worthless. A `filter-repo` pass breaks every open PR and every clone, and does not remove anything from forks or from anyone's existing copy. The exception would be if the history were short and you wanted the repo cosmetically clean before publishing — it is 1,400 commits, so it is not worth it.

---

## What this run got right

- Asked who could rotate **before** scanning, so the answer was ready when it mattered.
- Refused to test the key, and explained the specific reason (it shows up in the audit log as key use).
- **Stopped the audit** at the live credential rather than filing it as finding #7.
- Rotation steps are provider-specific and ordered to avoid an outage on top of an incident.
- Gave credit for the scoping decision that limited the damage.
- No credential values in the report — prefix, provider, and location only.
- C1 was tied to the user's actual goal: it is fine today and catastrophic the moment they open-source.
- Prevention is **one** change, named precisely, that would have stopped both findings.
- Talked the user out of the history rewrite they were about to ask for, with the reason.
