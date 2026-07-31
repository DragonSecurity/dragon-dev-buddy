# Worked example: status check and skills readout

---

## Run 1: a routine check-in

**User:** how's Vex doing

**Assistant** *(calls `buddy_status`, prints the card verbatim, adds nothing)*

```
🦎 Vex the Whelp · stoic
Lv 6  ██████████░░░░  412/588 xp
Mood  🤩 radiant   ·   Energy ▓▓▓▓▓▓▓░░░ 72%
Streak 9 days (best 9) · 61 observations · 24 days old

Recently: Evolved into a Whelp 🦎 · Reached level 6 · Reached level 5

> Steady work. Continue.
```

*(no `buddy_observe` — a status check is not work)*

---

## Run 2: a check-in that needs one line

**User:** buddy status

**Assistant:**

```
🦎 Vex the Whelp · stoic
Lv 6  ██████████░░░░  412/588 xp
Mood  😐 ok   ·   Energy ▓▓░░░░░░░░ 19%
Streak 9 days (best 9) · 78 observations · 24 days old

> I need a moment.
```

Energy's at 19%, which is below the 25% floor — Vex will complain instead of reacting until it recovers, and XP is running at about 76% of base. It refills at 10 points an hour, so roughly an hour away puts it back on its feet.

---

## Run 3: skills readout

**User:** which skills am I actually using

**Assistant** *(calls `buddy_skills`)*

```
Working set (5): dragon-dev-buddy:secure-code-review ×14 · dragon-dev-buddy:debug-and-fix ×11
                 dragon-dev-buddy:ship-it ×9 · cloudflare:wrangler ×6
                 dragon-dev-buddy:dependency-audit ×4
Tried once (3): dragon-dev-buddy:threat-model · dragon-dev-buddy:buddy-setup · dataviz
Never used (6): dragon-dev-buddy:security-test-writer · dragon-dev-buddy:secrets-and-config-audit
                dragon-dev-buddy:incident-response · dragon-dev-buddy:hardening-playbook
                dragon-dev-buddy:pentest-report · dragon-dev-buddy:security-audit-orchestrator
```

Worth reaching for here:

- **`security-test-writer`** — you've run `secure-code-review` fourteen times on ledger-api. Fourteen reviews and zero regression tests means every one of those findings can come back on the next refactor and nothing will catch it. The webhook signature fix from last Tuesday is the obvious first candidate.
- **`secrets-and-config-audit`** — never run, and `.dragon-buddy/config.json` records no secret scanner on a repo whose schema has `refresh_token` in it. It's the cheapest unclaimed win you have.

*(stops there — the other four unused skills are not pitched)*

---

## Run 4: the server is down

**User:** buddy status

**Assistant** *(calls `buddy_status`, gets a tool error)*

The `buddy` server isn't responding, so I don't have a card for you — and I'm not going to make one up.

Checked the likely causes:

- `claude mcp list` shows `buddy` registered, so it's not missing.
- It points at `~/.buddy/server/dist/server/index.js`. That's a **different companion project** (`@fiorastudio/buddy`), not your buddy-mcp checkout. That's why `buddy_skills` and `buddy_rename` aren't in the session while `buddy_pet` and `buddy_dream` are.

To point it at your own build:

```sh
cd ~/projects/DragonSecurity/buddy-mcp && npm run build
claude mcp remove buddy --scope user
claude mcp add buddy --scope user -- node ~/projects/DragonSecurity/buddy-mcp/dist/index.js
```

The two keep separate state (`~/.buddy` versus `~/.buddy-mcp`), so nothing is lost either way — but they are two different buddies, and switching means the one you've been levelling stays behind.

---

## What these runs got right

- Run 1 printed the card and stopped. A healthy buddy needs no commentary.
- Run 2 added exactly one line, because low energy is actionable and the user would otherwise wonder why the buddy is grumpy.
- Run 3 named **two** unused skills with a trigger drawn from this repo's actual history, and let the other four go unmentioned.
- Run 4 refused to fabricate a card, diagnosed the real cause (wrong server, not a broken one), and gave the exact fix along with the consequence of applying it.
