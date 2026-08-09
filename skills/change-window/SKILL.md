---
name: change-window
description: The pre-change gate for network devices and infrastructure. Diffs intent against what is actually running, checks whether the change can sever the path it is delivered over, verifies a rollback you can still reach, and gives a clear push or no-push. Use when someone says "pushing config to the switches", "change window", "maintenance window", "can I apply this", "pre-change check", "am I going to lock myself out", "is this firewall change safe", "staged rollout order", "rollback plan for this change", or before any change to a router, switch, firewall, load balancer, DNS zone or routing policy. A change you cannot back out of is not a change, it is a commitment.
---

# Change Window

Application deploys are reversible because the thing you deploy to is not the thing you deploy over. Infrastructure changes are different: the config you push can cut the session you pushed it from, the peer you need for the rollback, or the monitoring that would have told you it broke. This skill is the gate for that class of change. Its two jobs are to find the paths a change can sever, and to make you prove a rollback you can still reach after the change has gone wrong.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `fleet.inventory_source`, `fleet.config_backup`, `fleet.device_classes`, `fleet.vendors`, `fleet.out_of_band`, `fleet.change_window`, `fleet.approval`, `security.exposure`, `output.reports_dir`.

If the `fleet` block is empty and the change is a device or network change, ask the three questions that matter here — vendor and platform, what out-of-band access exists, and who approves — and offer to record them via `buddy-setup` so the next change does not ask again.

## Inputs

Ask only for what is missing:
- **What is changing.** The candidate config, the template diff, the plan output, or the commands. If the user has only described it in prose, get the actual text before gating anything.
- **What it is going onto.** Device names or roles, and how many. "The edge firewalls" is enough to start; "all of them" is not.
- **How you are connected to it.** In-band over the network being changed, out-of-band via console or a separate path, or through an orchestrator. This single answer decides how hard step 2 has to work.
- **Whether a window is open** and who is on it. A change with no window and no second person is a different risk profile, not a blocker.

## Workflow

Run the gates. Each is pass, warn, or fail. Report all of them — the user needs the whole picture, not the first failure.

1. **Diff intent against running, not against git.** Pull the running config from the target and diff the candidate against *that*. The repo is what someone intended; the device is what is true, and the gap between them is drift you are about to overwrite silently. If the running config differs from the reference in ways this change does not explain, stop and say so — that is `fleet-drift-audit`'s work, and pushing over unexplained drift destroys the evidence of how it got there.

2. **Self-severance check.** This is the gate that has no analogue in software. Walk the change line by line and ask whether it touches anything the change itself travels over or depends on: the management VLAN or interface, the ACL permitting your source address, the route to the jump host, the firewall policy allowing your session, the AAA server that authorizes the next command, DNS the orchestrator resolves through, or the interface carrying all of the above. `references/change-gates.md` has the catalogue by device class.

   If the change touches any of them, two things become mandatory rather than advisable: a **timer-based rollback armed before the change applies**, and **out-of-band access verified working right now** — connected to and tested this session, not assumed from the last time someone used it. An out-of-band path nobody has logged into in a year is not a rollback plan.

3. **Rollback you can still reach.** Name the mechanism, not the intention. The platform-specific forms are in the reference — a confirmed commit with a timer, a scheduled reload, a config replace from a saved checkpoint, a candidate rolled back before commit. Then ask the question that kills most plans: *after this change is wrong, can I still get to the thing that performs the rollback?* If the answer depends on the change having worked, the rollback is imaginary. Automatic timers do not have this problem, which is why they rank above manual reverts.

   Check what the rollback does **not** undo: sessions torn down, neighbor relationships reset, caches flushed, leases handed out, a route already withdrawn from a peer you do not control. Reverting the config restores the config, not the state.

4. **Blast radius by topology, not by line count.** State concretely what breaks if this is wrong: how many devices, where they sit, and what is behind them. Position dominates — the same three lines on an access switch and on the transit edge are not the same change. Check redundancy honestly: is the peer healthy *right now*, is it running the version you think, and is this change going to both halves of a pair. Both halves of a pair in one window is a fail, every time, regardless of how safe the change looks.

5. **Order of operations.** Sequence the change so that no intermediate state is broken. The general rule is additive-before-subtractive: the new permit exists before the old one is removed, the new session is established before the old one is torn down, the new next-hop is reachable before the old is withdrawn. Symmetric settings that must match on both ends (MTU, encapsulation, authentication, timers) have a window where they disagree — name it and say how long it lasts. For a fleet, give the device order explicitly, least critical first, with a soak between the first device and the rest.

6. **Stateful teardown.** List what is stateful on the path and will not survive: firewall sessions and NAT translations, established TCP through a load balancer, routing adjacencies and their convergence time, ARP and neighbor caches, DHCP leases, tunnel security associations, session affinity. Users experience the teardown, not the config. A change that is correct and drops every established session at 14:00 is still an outage.

7. **How you will know.** State what you will watch during and after, and — the part usually missed — confirm that the monitoring does not traverse the thing being changed. If your only reachability check comes back through the interface you are reconfiguring, you have no monitoring during the window. Name a check that survives, and name the specific signal that means back it out.

8. **Give the verdict.** One of:
   - **Push** — gates pass, severance risk understood, rollback is real and reachable.
   - **Push with conditions** — go, but arm X first, order it this way, or watch Y before continuing past the first device.
   - **Do not push** — a gate failed. Say which, and what would clear it.

   Do not hedge. The user came for a decision, usually with a window open and a clock running. Give it, with reasons, and let them override a clear no.

## Output format

```markdown
## Change check: [what] → [targets]

| Gate | Result | Detail |
| Intent vs running | pass/warn/fail | [unexplained drift on the target?] |
| Self-severance | pass/warn/fail | [what the change touches that it travels over] |
| Rollback | pass/warn/fail | [mechanism, timer, and whether it is reachable after failure] |
| Blast radius | — | [device count, position, what is behind them, pair status] |
| Ordering | pass/warn/fail | [sequence, and any window where both ends disagree] |
| Stateful teardown | — | [what drops: sessions, adjacencies, leases] |
| Observability | pass/warn/fail | [what you watch, and whether it survives the change] |

## Verdict: **[PUSH / PUSH WITH CONDITIONS / DO NOT PUSH]**
[the reasoning, conditions or blockers, specifically]

## Sequence
1. [step, with the rollback timer armed at the point it is armed]
2. [...]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Change check on <what> across <n> devices: <verdict> — <one clause>."`
- `kind`: `deploy`
- `skills_used`: `["dragon-dev-buddy:change-window"]`

Report the check whether it cleared or blocked; a caught no-push is the work. Relay the reaction verbatim.

## File output

Usually none — the verdict goes in chat, where the decision is being made against a clock. Where a change record is required, write the check to `output.reports_dir` as `YYYY-MM-DD-change-check-<what>.md`, including the sequence actually followed and anything that deviated from it. This skill does not push config. Applying the change is a separate, explicit action the user takes.

## Reference library

Load these for depth when the task calls for it:
- `references/change-gates.md`: the pass/warn/fail thresholds, the self-severance catalogue by device class, rollback mechanisms per platform with their timer semantics, the stateful-teardown catalogue, ordering rules for symmetric and additive changes, and the change shapes that warrant extra caution.

## Worked example

See `examples/change-window-run.md` for a firewall policy push that would have cut the management path, and the cleared re-check. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- The candidate was diffed against the *running* config, not against the repo, and unexplained drift was surfaced rather than overwritten.
- Every path the change travels over was checked for severance, and a positive hit forced both an armed timer and a tested out-of-band path.
- The rollback names a mechanism and survives the question "can I reach it after this goes wrong."
- What the rollback does not undo — sessions, adjacencies, caches, leases — is stated.
- Blast radius is expressed in devices and topology position, and the redundant peer's current health was checked, not assumed.
- The sequence has no broken intermediate state, and any window where two ends disagree is named with its duration.
- The monitoring named for the window does not depend on the thing being changed.
- The verdict is unambiguous, and the gate is willing to say no with a window already open.
