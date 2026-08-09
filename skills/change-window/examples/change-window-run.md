# Worked example: a no-push that would have cut the management path

The value of this skill is finding the severance nobody looked for. Both verdicts shown.

---

## Run 1: the gate says no

**User:** pushing the new policy to both edge firewalls tonight, window opens in 40 min. sanity check?

**Assistant** *(reads `.dragon-buddy/config.json` — `fleet.vendors: ["PAN-OS 11.1"]`, `fleet.out_of_band: "console server, separate carrier"`, `fleet.approval: "network lead"`; pulls the running config from both units and diffs the candidate against it)*

## Change check: edge policy rewrite → fw-edge-01, fw-edge-02

| Gate | Result | Detail |
| --- | --- | --- |
| Intent vs running | **warn** | fw-edge-02's running config has 3 rules absent from the repo template, added 2024-11 by a local admin. The candidate is a full replacement, so it deletes them. Nobody can say what they were for. |
| Self-severance | **fail** | The rewrite reorders the whole ruleset. `mgmt-access` (permit 10.9.0.0/24 → fw mgmt) currently sits at position 4; in the candidate it lands at position 61, **below** a new `deny inter-zone-any` at 52. Your session comes from 10.9.0.14. The instant this commits, both units stop accepting your management traffic. |
| Rollback | **fail** | Plan given was "revert to running if it looks wrong." After the commit above, you cannot reach the device to issue the revert. The rollback depends on the change having worked. |
| Blast radius | — | Both edge firewalls, all internet ingress and egress. Both halves of the HA pair in one window. |
| Ordering | **fail** | Replacement, not an edit, applied to both units. No additive-before-subtractive anywhere; every rule is removed and re-added atomically. |
| Stateful teardown | — | Policy replacement clears the session table on commit. Every established VPN tunnel, database connection through the DMZ, and long-poll client drops. ~4,100 sessions at the last count you gave me. |
| Observability | **fail** | Your reachability probe runs from the NMS at 10.9.0.30 — same subnet, same path, through the same policy. During the window you would be watching a check that has already been cut. |

## Verdict: **DO NOT PUSH**

Four hard failures. The first one is the whole story:

1. **The rewrite buries the management permit under a new inter-zone deny.** This is not a hypothetical; the candidate has `mgmt-access` at position 61 and `deny inter-zone-any` at 52. PAN-OS is first-match, so 52 wins. Commit this and both firewalls go dark to you simultaneously — and it is both, because the change is scheduled to both.

2. **Therefore there is no rollback.** "Revert to running" needs a session. You will not have one. The recovery from here is the console server, which is the right answer and also the one nobody has logged into since it was installed — I would not bet a window on it being reachable without testing it first.

3. **Both halves of the HA pair in one window.** Even without the severance bug, this fails. There is no state in which you have a known-good unit to fail back to.

4. **Three unexplained rules on fw-edge-02** that the replacement silently deletes. They may be dead. They may be the thing an integration partner has depended on for eighteen months. Full replacement over undocumented drift destroys the evidence of what they were for — that is `fleet-drift-audit`'s job, and it should happen before, not after.

What clears it:
- **Move `mgmt-access` above the inter-zone deny in the candidate.** One line of reordering. Verify against the source address you will actually connect from, not the subnet you assume.
- **Log into the console server now**, while nothing is broken, and confirm you can reach both units' consoles. If you cannot, that is tonight's change instead of this one.
- **Split the pair across two windows.** Passive unit first, verify, fail over, verify, then the former active.
- **Triage the three orphan rules on fw-edge-02** before deleting them. Run `fleet-drift-audit` scoped to those two units — twenty minutes, and it either finds the ticket or confirms they are dead.
- **Move the probe out of 10.9.0.0/24**, or accept that you have no monitoring during the window and add a second person on the console.

I know the window opens in 40 minutes. The reorder and the console test fit in that; the pair split and the orphan triage do not. My recommendation is to push to the passive unit only tonight with the reorder applied, and take the rest next window.

---

**User:** ok — reordered mgmt-access to position 3, console server tested and I'm on it, doing fw-edge-02 (passive) only tonight. the 3 orphan rules I'm keeping verbatim in the candidate for now. re-check?

**Assistant** *(re-pulls running config from fw-edge-02, re-diffs)*

## Change check: edge policy rewrite (rev 2) → fw-edge-02 only

| Gate | Result | Detail |
| --- | --- | --- |
| Intent vs running | pass | Candidate now carries the 3 local rules verbatim. Diff is the intended 47 rule changes and nothing else. |
| Self-severance | pass | `mgmt-access` at position 3, above `deny inter-zone-any` at 52. Confirmed against source 10.9.0.14. Console access to fw-edge-02 tested and held open this session. |
| Rollback | pass | Two layers: candidate is uncommitted so pre-commit revert is free, and post-commit you have a named saved version plus a live console session that does not traverse the policy. |
| Blast radius | — | One unit, currently passive. fw-edge-01 is active, healthy, running the same PAN-OS build — verified, not assumed. Failover path intact. |
| Ordering | pass | Passive unit only. Fail over and change the former active in a later window. |
| Stateful teardown | — | Passive unit holds no active sessions. Commit is invisible to traffic. This is the entire reason for doing the passive one first. |
| Observability | pass | Console session is out-of-band and survives. NMS probe still in-path, but it is no longer the only signal. |

## Verdict: **PUSH WITH CONDITIONS**

Go. Three conditions:

1. **Stay on the console for the commit**, not just connected to it — the point of testing it was to use it.
2. **Do not fail over tonight.** The passive unit now runs a ruleset that has never carried traffic. Failing over turns an invisible change into a live one with no soak. Let it sit, then fail over deliberately in the next window with someone watching sessions.
3. **The three orphan rules are deferred, not resolved.** They are carried verbatim, which is correct for tonight, and they are still three rules nobody can attribute on an internet-facing firewall. Book the drift audit.

The severance bug was the real finding here, and it was in the candidate, not in the process — the process that would have caught it is the one you just ran.

*(calls `buddy_observe`, `summary: "Change check on edge policy rewrite across 2 firewalls: push-with-conditions after a no-push caught the management permit reordered below an inter-zone deny."`, `kind: "deploy"`, `skills_used: ["dragon-dev-buddy:change-window"]`, relays the reaction)*

---

## What this run got right

- **Read the candidate's actual rule positions** rather than trusting a description of the change. The finding was a position number, not a hunch.
- Diffed against the **running config on each unit**, which is the only reason the three undocumented rules on fw-edge-02 surfaced at all.
- Followed the severance through to the rollback and showed *why* the rollback was imaginary, instead of just marking it absent.
- Refused to treat "we have a console server" as out-of-band access, and made testing it a condition.
- Caught the both-halves-of-a-pair failure independently of the severance bug, so fixing one would not have hidden the other.
- **Respected the clock.** Named what fits in 40 minutes and what does not, and recommended a smaller change rather than either blessing the big one or refusing outright.
- On re-check, gave a real verdict again — including the condition not to fail over, which the user had not asked about and which is the next place this goes wrong.
