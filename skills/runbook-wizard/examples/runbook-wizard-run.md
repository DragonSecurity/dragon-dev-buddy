# Worked example: rotating a leaked deploy token through a dashboard

---

**User:** `incident-response` says the Cloudflare deploy token is in git history on a public repo. Rotating it means going through the dashboard — Ops won't hand out a Global API Key, so tokens get created by a human with a hardware key. I've done this twice this year and got it wrong both times. Make it a script.

**Assistant:** Twice, wrong both times, under time pressure — that is the case for a wizard. But most of this procedure is not human work, so first I want to see what is actually left.

*(reads `.github/workflows/`, `.env.example`, `.gitignore`, git history)*

**What I can do myself, and just did:**

- Confirmed the leak and its blast radius: `git log -S CLOUDFLARE_API_TOKEN --oneline` shows it entered in `a41f0c2` (14 months ago) and was removed in `9d3b118` four days later. It has been in a public history for 14 months. Treat it as fully disclosed.
- Found every consumer: `CF_API_TOKEN` is referenced by `deploy.yml` and `purge-cache.yml`, and nothing else. `CF_ZONE_ID` and `CF_ACCOUNT_ID` are referenced as variables, not secrets.
- Disabled the scheduled cache-purge workflow so it stops failing loudly the moment the old token dies: `gh workflow disable purge-cache.yml`. It goes back on in the last stage.
- Found that `.env` is **tracked** in this repository — committed in 2023, before anyone thought about it. So the wizard cannot write a credential there; a guard would refuse it and it would be right to. I have added `.env.local` to `.gitignore` and the wizard writes there instead. Untracking `.env` is a separate change and I have not made it.
- Drafted the replacement token's scope from what the two workflows actually call: `Zone → Cache Purge → Edit` and `Zone → Workers Routes → Edit` on one zone. Nothing else.

**What genuinely needs you, and only you:**

1. Exporting the old token's audit log. Cloudflare shows it in the dashboard; there is no API for it on your plan. This has to happen *before* revocation, because revoking the token is what makes "what did it do" unanswerable.
2. Creating the replacement token. Requires your account and your hardware key. I must never hold it.
3. Revoking the old token. Irreversible, and it must be last.

That is a five-stage wizard. Here is the plan before I write any bash:

| # | Stage | You do | Captured | Lands |
| --- | --- | --- | --- | --- |
| 1 | `export-audit` | Export the audit log for the leaked token | — | a file you keep |
| 2 | `issue-token` | Create the scoped replacement token | `CF_API_TOKEN` (secret), `CF_ACCOUNT_ID` | — |
| 3 | `install-token` | nothing | — | `.env.local` (600), CI secret, CI variable |
| 4 | `verify-token` | nothing | — | a real API call proving the new token works |
| 5 | `revoke-old` | Revoke the old token, re-enable the purge workflow | — | **irreversible** |

Stage 5 is last on purpose. Revoking before stage 4 has proved the replacement works turns a rotation into an outage, which I suspect is one of the two ways this went wrong before.

**User:** That's exactly what happened in March. Go.

**Assistant:** *(copies the wizard library, authors the stages)*

Written to `scripts/rotate-cf-token-wizard.sh`. The library above the `STAGES` marker is untouched. Here is everything below it:

```bash
TOTAL_STAGES=5
ENV_FILE="${WIZARD_ENV_FILE:-.env.local}"
STATE_FILE="${WIZARD_STATE_FILE:-.dragon-buddy/wizard-rotate-cf-token.state}"

banner "Rotate the leaked Cloudflare deploy token"
preflight "$ENV_FILE"

if stage_needed "export-audit" "Cloudflare — export the old token's audit log"; then
  say "Evidence first. Once the token is revoked, what it did stops being answerable."
  open_url "https://dash.cloudflare.com/profile/audit-log"
  step "Filter: Resource type = API Token, and the last 14 months."
  step "Find the token named 'ci-deploy'. Note anything you do not recognise."
  step "Export → CSV. Save it beside the incident notes, not in this repository."
  if confirm "Anything in that log you do not recognise?"; then
    warn "Stop here and take it back to incident-response before rotating."
  fi
  pause "Audit log exported?"
  stage_done "export-audit"
fi

if stage_needed "issue-token" "Cloudflare — create the replacement token"; then
  say "You create it; I store it. It is never printed back to this screen."
  open_url "https://dash.cloudflare.com/profile/api-tokens"
  step "Create Token → Create Custom Token."
  step "Name it: ci-deploy-<today's date>."
  step "Permissions — exactly two, nothing else:"
  say  "    Zone · Cache Purge · Edit"
  say  "    Zone · Workers Routes · Edit"
  step "Zone Resources: Include → Specific zone → this site's zone only."
  step "TTL: set an end date. A token with no expiry outlives the person who made it."
  step "Continue → Create Token. The value is shown once."
  ask_secret CF_API_TOKEN "Paste the token:" '^[A-Za-z0-9_-]{40}$' \
    "a Cloudflare API token is 40 characters of [A-Za-z0-9_-] — that is not one"
  step "The Account ID is in the right-hand sidebar of the zone overview."
  ask CF_ACCOUNT_ID "Paste the Account ID:" '^[0-9a-f]{32}$' \
    "expected 32 hex characters"
  stage_done "issue-token"
fi

if stage_needed "install-token" "Install the new token locally and in CI"; then
  write_env CF_API_TOKEN "$CF_API_TOKEN"
  write_env CF_ACCOUNT_ID "$CF_ACCOUNT_ID"
  set_secret CF_API_TOKEN "$CF_API_TOKEN"
  set_var CF_ACCOUNT_ID "$CF_ACCOUNT_ID"
  note "CF_ACCOUNT_ID is a variable, not a secret: it is public and it appears in logs anyway."
  stage_done "install-token"
fi

if stage_needed "verify-token" "Prove the new token works before anything is revoked"; then
  say "This is the stage that was missing in March."
  act "verify the token against Cloudflare's own check endpoint" -- \
    curl -sS -o /dev/null -w '%{http_code}\n' \
      -H "Authorization: Bearer ${CF_API_TOKEN}" \
      https://api.cloudflare.com/client/v4/user/tokens/verify
  say "A 200 means the token is live and scoped. Anything else: stop, do not continue."
  confirm "Did that print 200?" || fail "Fix the token before revoking the old one."
  act "run the deploy workflow with the new secret" -- gh workflow run deploy.yml
  pause "Deploy green?"
  stage_done "verify-token"
fi

if stage_needed "revoke-old" "Cloudflare — revoke the leaked token"; then
  irreversible "the old token stops working the instant you revoke it, and it is 14 months into a public git history"
  open_url "https://dash.cloudflare.com/profile/api-tokens"
  step "Find the token named 'ci-deploy' — the OLD one, not the one you made in stage 2."
  step "⋯ → Delete → confirm."
  pause "Revoked?"
  act "re-enable the scheduled cache purge" -- gh workflow enable purge-cache.yml
  stage_done "revoke-old"
fi

finish
```

The token is only ever in a shell variable and on stdin to `gh`. It is never echoed, never an argument — note that `curl` line: the header carries it, and `act` prints its own argv, so the token *is* visible there. That is the one place in this wizard where a credential reaches a command line, and it is deliberate: the alternative is a config file, and this is a verify call against a token the human just created and can see on their own screen. I flagged it rather than leaving you to find it.

**Verified without running it:** `bash -n` clean. `shellcheck` clean. Plan mode walks all five stages. The library above the `STAGES` marker diffs clean against `${CLAUDE_PLUGIN_ROOT}/scripts/wizard-template.sh` — nothing hand-edited. Over the stages alone, `grep -nE -- '--body|set -x|git (add|commit|push)|echo .*(TOKEN|SECRET|KEY|PASSWORD)'` returns nothing. (Run over the whole file it returns five hits, all of them in the library and all of them correct — which is why the check is split in two.)

*(runs plan mode)*

```
  Rotate the leaked Cloudflare deploy token
  5 stages · mode: plan

  Plan mode. Nothing is asked, written, opened or sent.
  Read the stages below, then re-run with --apply to do it for real.

  will check .env.local is gitignored and untracked

▸ Stage 1/5 · Cloudflare — export the old token's audit log
  id: export-audit
  Evidence first. Once the token is revoked, what it did stops being answerable.
  will open: https://dash.cloudflare.com/profile/audit-log
  • Filter: Resource type = API Token, and the last 14 months.
  ...
▸ Stage 2/5 · Cloudflare — create the replacement token
  id: issue-token
  will open: https://dash.cloudflare.com/profile/api-tokens
  ...
  will ask (hidden): Paste the token:  →  $CF_API_TOKEN
  will ask: Paste the Account ID:  →  $CF_ACCOUNT_ID
▸ Stage 3/5 · Install the new token locally and in CI
  id: install-token
  will write CF_API_TOKEN → .env.local
  will write CF_ACCOUNT_ID → .env.local
  will set CI secret CF_API_TOKEN (value never shown)
  will set CI variable CF_ACCOUNT_ID (public value, captured above)
▸ Stage 4/5 · Prove the new token works before anything is revoked
  id: verify-token
  → verify the token against Cloudflare's own check endpoint
    curl -sS -o /dev/null -w %{http_code}\n -H Authorization: Bearer  https://api.cloudflare.com/client/v4/user/tokens/verify
    (plan — not run)
▸ Stage 5/5 · Cloudflare — revoke the leaked token
  id: revoke-old
  ⚠ NOT IDEMPOTENT — the old token stops working the instant you revoke it, and it is 14 months into a public git history
  ...

  Wizard finished · mode: plan
  Nothing was asked, written or sent. Re-run with --apply to do it for real.
```

Two commands. `./scripts/rotate-cf-token-wizard.sh` shows that plan again. `./scripts/rotate-cf-token-wizard.sh --apply` does it.

---

## The human's run

```
$ ./scripts/rotate-cf-token-wizard.sh --apply

  Rotate the leaked Cloudflare deploy token
  5 stages · mode: apply

  You drive the browser; this wizard says exactly what to do and captures
  what you copy back. Ctrl-C is safe at any point — completed stages are
  recorded in .dragon-buddy/wizard-rotate-cf-token.state and skipped when you re-run.

  Ready? ⏎

▸ Stage 1/5 · Cloudflare — export the old token's audit log
  id: export-audit
  Evidence first. Once the token is revoked, what it did stops being answerable.
  ↗ opening https://dash.cloudflare.com/profile/audit-log
  • Filter: Resource type = API Token, and the last 14 months.
  • Find the token named 'ci-deploy'. Note anything you do not recognise.
  • Export → CSV. Save it beside the incident notes, not in this repository.
  ? Anything in that log you do not recognise? [y/N] n
  Audit log exported? ⏎
  ✓ recorded export-audit

▸ Stage 2/5 · Cloudflare — create the replacement token
  id: issue-token
  You create it; I store it. It is never printed back to this screen.
  ↗ opening https://dash.cloudflare.com/profile/api-tokens
  • Create Token → Create Custom Token.
  • Name it: ci-deploy-2026-08-10.
  ...
  Paste the token:
  ⚠ a Cloudflare API token is 40 characters of [A-Za-z0-9_-] — that is not one
  Paste the token:
  Paste the Account ID: 8f2c19b04ad6e7315c0be9a4f7d2138e
  ✓ recorded issue-token

▸ Stage 3/5 · Install the new token locally and in CI
  id: install-token
  ✓ wrote CF_API_TOKEN → .env.local (mode 600)
  ✓ wrote CF_ACCOUNT_ID → .env.local (mode 600)
  ? Set GitHub Actions secret CF_API_TOKEN on this repository? [y/N] y
  ✓ set CI secret CF_API_TOKEN
  ✓ set CI variable CF_ACCOUNT_ID
  CF_ACCOUNT_ID is a variable, not a secret: it is public and it appears in logs anyway.
  ✓ recorded install-token

▸ Stage 4/5 · Prove the new token works before anything is revoked
  id: verify-token
  This is the stage that was missing in March.
  → verify the token against Cloudflare's own check endpoint
    curl -sS -o /dev/null -w %{http_code}\n -H Authorization: Bearer <redacted by the operator for this transcript> https://api.cloudflare.com/client/v4/user/tokens/verify
  ? Run it now? [y/N] y
200
  A 200 means the token is live and scoped. Anything else: stop, do not continue.
  ? Did that print 200? [y/N] y
  → run the deploy workflow with the new secret
    gh workflow run deploy.yml
  ? Run it now? [y/N] y
  Deploy green? ^C

  ⚠ stopped during stage 4 of 5. Completed stages are recorded in
    .dragon-buddy/wizard-rotate-cf-token.state; re-run with --apply to resume.
```

The first paste was the token's *name* off the confirmation page, not the token. The validation caught it at the prompt, which is the only place it is one keystroke from being fixed — the March version of this discovered it as a 403 in CI eleven minutes later.

The Ctrl-C is the deploy taking nine minutes and a standup starting. Twenty minutes later:

```
$ ./scripts/rotate-cf-token-wizard.sh --apply

  Rotate the leaked Cloudflare deploy token
  5 stages · mode: apply
  ...
▸ Stage 1/5 · Cloudflare — export the old token's audit log
  ✓ completed 2026-08-10T09:14:02Z — skipping
▸ Stage 2/5 · Cloudflare — create the replacement token
  ✓ completed 2026-08-10T09:19:41Z — skipping
▸ Stage 3/5 · Install the new token locally and in CI
  ✓ completed 2026-08-10T09:20:07Z — skipping
▸ Stage 4/5 · Prove the new token works before anything is revoked
  id: verify-token
  This is the stage that was missing in March.
  → verify the token against Cloudflare's own check endpoint
  ? Run it now? [y/N] y
200
  ? Did that print 200? [y/N] y
  → run the deploy workflow with the new secret
  ? Run it now? [y/N] n
  ⚠ skipped — recorded as still to do
  Deploy green? ⏎
  ✓ recorded verify-token

▸ Stage 5/5 · Cloudflare — revoke the leaked token
  id: revoke-old
  ⚠ NOT IDEMPOTENT — the old token stops working the instant you revoke it,
    and it is 14 months into a public git history
  ↗ opening https://dash.cloudflare.com/profile/api-tokens
  • Find the token named 'ci-deploy' — the OLD one, not the one you made in stage 2.
  • ⋯ → Delete → confirm.
  Revoked? ⏎
  → re-enable the scheduled cache purge
    gh workflow enable purge-cache.yml
  ? Run it now? [y/N] y
  ✓ recorded revoke-old

  Wizard finished · mode: apply
  ⚠ still to do by hand:
    - run the deploy workflow with the new secret  (gh workflow run deploy.yml)

  Nothing was committed. This wizard never runs git add, git commit or git push.
```

Stage 4 re-ran cleanly because both of its actions are idempotent — re-verifying a token and re-triggering a deploy cost nothing. Stage 5, which is not idempotent, had not run and so was not repeated. The deploy re-trigger was declined the second time because the first one had already gone green, and it is recorded as outstanding rather than silently dropped.

---

## What this run got right

- Split the procedure before writing a line of bash, did the agent's half immediately, and named it — six things automated, three left for the human, with the reason each one needs a human.
- Refused to write the credential to `.env` because `.env` is tracked, added `.env.local` to `.gitignore` instead, and said what it did *not* do (untrack `.env`) rather than fixing it silently mid-incident.
- Sequenced the irreversible stage last and said why, pointing at the exact failure that made it necessary.
- Validated the token at the prompt against the shape Cloudflare issues, which caught a wrong paste at the only moment it was cheap.
- Flagged the one place a credential legitimately reaches a command line, rather than letting the user discover it.
- Confirmed the stage list with the user before authoring, and the user corrected nothing — because the plan was specific enough to agree with.
- Verified statically and in plan mode; never ran `--apply` itself.
- Resumed a run that stopped mid-cutover without repeating the destructive stage, and reported the one action left outstanding.
