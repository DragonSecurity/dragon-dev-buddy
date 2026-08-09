# Worked example: a fabric plus edge, where the template was the wrong one

Shows the separation that makes this skill worth running: 2,900 diff lines down to three findings, and one of them resolved by fixing the reference rather than the fleet.

---

**User:** we've got 14 devices in the fabric and 4 edge routers. someone changed something on a leaf last month and nobody knows what. can you audit the whole lot?

**Assistant** *(reads `.dragon-buddy/config.json` — `fleet.config_backup: "Oxidized"`, `fleet.golden_config: "net-templates/ repo"`, `fleet.vendors: ["Arista EOS 4.31", "Juniper Junos 22.4"]`, `security.exposure: "public"`, `security.compliance: ["PCI DSS"]`; pulls from Oxidized, normalizes, diffs within classes)*

Before the report — one thing from the collection itself. Oxidized's last successful pull for `leaf-09` is **2026-04-30, 97 days ago**. Every diff below treats `leaf-09` as matching its class, and that is only true of a config from April. I have excluded it from the comparison and listed it as a finding instead.

# Fleet drift audit: fabric + edge
2026-08-06 · 18 devices across 3 classes · reference: rendered intended-config from `net-templates/`

## Summary

The fabric is in good shape: 12 leaves and 2 spines are near-identical to the rendered template, with the differences confined to per-device values. The edge is not — the four routers have diverged from each other and from the template in ways that predate anyone's memory. Raw diff was ~2,900 lines; after normalization, 61 real differences, of which 3 are security-relevant.

## Security-relevant drift (ranked)

| # | Finding | Devices | Class | Position | Exposes | Direction |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Management ACL absent; SSH reachable from any source | `edge-03` | edge | internet-facing | Management plane exposed to every network the router routes for, including transit | reconcile |
| 2 | Stale TACACS+ server, falls back to local auth | 8 leaves | leaf | access | Shared local credential is the live auth path on a third of the fabric | reconcile |
| 3 | Inbound prefix filter missing on one peering session | `edge-01` | edge | internet-facing | No protection against a peer leaking or hijacking prefixes on that session | reconcile |

### 1. `edge-03` has no management-plane ACL

The template renders `ip access-class MGMT-IN in` on the SSH VTY lines. `edge-01`, `edge-02` and `edge-04` all have it. `edge-03` does not, and its `MGMT-IN` list does not exist at all — this is not a modified ACL, it is an absent one.

Oxidized's history dates the removal to **2025-09-12**, in a commit that also added a temporary BGP session for a partner turn-up. The read is that someone stripped the ACL to get access from an unexpected source during the turn-up and never put it back. Eleven months.

`edge-03` is internet-facing and this is a `public` exposure environment with PCI DSS in scope. This is the finding.

**Direction:** reconcile the device — the template is right and every peer agrees with it.

### 2. Eight leaves point at a decommissioned TACACS+ server

`leaf-01` through `leaf-08` list `tacacs-server host 10.4.1.20`. That host was replaced by `10.4.1.25` in the template in January; `leaf-10` through `leaf-14` and both spines carry the new one. The old address does not answer.

The consequence is not "those switches cannot authenticate." It is worse and quieter: their AAA config falls through to `local`, so eight switches are authenticating against a local account whose password is the same on all eight and rotates on nobody's schedule. Central logging shows no TACACS+ auth events from these devices, which is consistent.

These eight are the eight that were being re-cabled during the January change window and were unreachable when the template went out. Nothing re-ran the play afterwards.

**Direction:** reconcile the devices — and then treat the gap in the process as the real fix, since the reconcile only holds until the next partial rollout. See "Making it stick."

### 3. `edge-01` peering session has no inbound prefix filter

Of four eBGP sessions on `edge-01`, three apply `route-map PEER-IN in`. The session to AS 64512 does not. The template renders the filter for every session in the peer group; this session was configured outside the group, individually, which is why the group's policy never reached it.

**Direction:** reconcile the device, and move the session into the peer group so it cannot fall out again.

## Decay (not security-relevant, still unmanaged)

- **MTU 9214 on 6 leaves, 9000 on 6 leaves** (template says 9214). No functional impact today because the mismatch is not on any path that carries jumbo traffic. It will have an impact the day someone moves storage traffic. Worth reconciling before it becomes an incident nobody can diagnose.
- **Syslog facility differs on the two spines** (local6 vs local7). Logs arrive; they sort into the wrong bucket.
- **Banner text on the edge routers** — three different versions, one still naming a company acquired in 2023.
- **`edge-02` and `edge-04` have interface descriptions the template does not render.** Better than the template's output, not worse. See below.

## The one to absorb, not reconcile

The template renders no interface descriptions on edge uplinks. Three of four edge routers have detailed, accurate, hand-written descriptions naming circuit IDs and carriers. The literal reading of this audit is "3 of 4 devices drift from the reference." The correct reading is that **the template is missing something the operators needed enough to add by hand three times independently.**

**Direction:** absorb into the reference. Add circuit-ID descriptions to the edge template, sourced from the devices that have them. Pushing the template as-is would erase eleven circuit IDs that exist nowhere else, and they would be re-added by hand within a month — which is how you get here again.

## Sanctioned exceptions

None on record, which is itself worth noting: there is no place to record one, so every deliberate deviation looks like decay to this audit and to the next one. Recommend a file in `net-templates/` listing exception, device, owner, reason, review date.

## Snowflakes

- **`leaf-09`** — no successful config collection in 97 days. Not drifted; *unknown*. It could match perfectly or be the most divergent device in the fabric. Find out why collection fails before anything else here, because a device your backup system cannot read is also a device your automation cannot fix.

## Fleet-wide checks (no drift, still findings)

Uniform settings produce no diff, so these were checked explicitly:
- **SNMP v2c is configured fleet-wide**, all 18 devices, one community string. No drift, and it is the weakest thing in this report after finding 1. Goes to `hardening-playbook`, not here.
- Management-plane ACLs exist on 17 of 18 (see finding 1) — the fleet default is correct, which is what makes `edge-03` an outlier rather than a norm.

## Making it stick

Oxidized is already storing every config and every change, which is how findings 1 and 2 got dated to the day. What is missing is anyone being told.

1. **Webhook on unexpected Oxidized commits** — a commit outside a change window, to a channel someone reads. Would have caught finding 1 on 2025-09-12 instead of today, and finding 3 at turn-up.
2. **Alert on collection failure, not just on change** — `leaf-09` has been silently invisible for three months. This is the smaller change and it closes the worse gap.
3. **Re-run the play against the full inventory after any partial rollout** — findings 2 and the January window are the same problem. A nightly check-mode run over the whole fabric would have surfaced the eight leaves the next morning.

*(calls `buddy_observe`, `summary: "Drift audit across 18 devices: 3 security-relevant findings, worst was an internet-facing edge router with no management-plane ACL for 11 months."`, `kind: "config"`, `skills_used: ["dragon-dev-buddy:fleet-drift-audit"]`, relays the reaction)*

---

Remediation is four config pushes across two classes, including a bulk change to eight leaves. Run that through `change-window` before it goes anywhere — the management-ACL fix on `edge-03` in particular is a change to the management plane of an internet-facing router, which is exactly the shape that gate exists for.

---

## What this run got right

- **Reported the collection gap before the findings.** `leaf-09` looked compliant because its data was frozen; presenting it inside the fleet would have been a false clean bill.
- Normalized first — 2,900 raw diff lines to 61 real differences to 3 that matter. Without that step the three findings are unfindable.
- Used the config backup's **history**, not just its latest snapshot, and dated two findings to the day and to the change that caused them.
- Read finding 2 through to the actual consequence — not "wrong TACACS+ address" but "eight switches silently fell back to a shared local password" — and confirmed it against central logging.
- **Called the template wrong when the template was wrong.** Three devices independently adding circuit IDs is evidence, and reconciling would have destroyed data that exists nowhere else.
- Ran fleet-wide checks that a diff structurally cannot find, and routed the SNMP v2c finding to the skill that owns it rather than padding this report.
- Made the durable recommendations specific to what they would have caught here, and ranked the cheapest one that closes the worst gap first.
