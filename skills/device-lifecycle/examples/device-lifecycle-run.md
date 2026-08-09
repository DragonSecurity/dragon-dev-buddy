# Worked example: 200 criticals down to three, and one that cannot be fixed

Advisory identifiers below are illustrative placeholders, not real published advisories. The triage reasoning is the part to copy.

---

**User:** the vuln scanner is reporting 214 criticals across our network gear and management is asking why we haven't patched anything. help.

**Assistant** *(reads `.dragon-buddy/config.json` — `fleet.inventory_source: "NetBox"`, `fleet.vendors: ["FortiOS", "Arista EOS", "Cisco IOS-XE"]`, `fleet.change_window: "Tue/Thu 22:00-02:00"`, `security.exposure: "public"`, `compliance: ["SOC 2"]`; pulls versions live via NAPALM and reconciles against NetBox)*

The scanner is matching version strings. That is not the same as 214 exploitable problems, and the reason nothing has been patched is that a list of 214 criticals is not actionable — nobody can start it. Let me get it to a list you can start.

One thing from the inventory reconciliation first: NetBox lists 61 devices, live collection reached 58. The three it could not reach are in the report as their own finding, because an unknown version is not a passing grade.

# Device lifecycle audit: network fleet
2026-08-06 · 61 devices · 3 platforms · inventory confidence: live via NAPALM for 58, **NetBox record only (stale) for 3**

## Lifecycle exposure

| Platform | Devices | Version | Stage | Software maintenance ends | Position |
| --- | --- | --- | --- | --- | --- |
| FortiOS | 4 | 7.4.3 | supported | 2028-Q1 | edge, internet-facing |
| Arista EOS | 38 | 4.31.2F | supported | 2027-Q3 | fabric leaf/spine |
| Arista EOS | 8 | 4.28.4M | supported | 2026-Q4 | fabric leaf |
| Cisco IOS-XE | 9 | 17.6.4 | supported | 2027-Q1 | distribution |
| Cisco IOS 15.x | 2 | 15.2(7)E | **past software maintenance since 2024** | passed | access, one in the DMZ rack |
| unknown | 3 | unknown | unknown | unknown | unknown |

## Findings (ranked)

| # | Advisory | Devices | Applicable because | Reachable from | Known exploited | Fixed in | Priority |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | FG-IR-EX-001 — pre-auth RCE in SSL VPN | 2 edge | SSL VPN is configured and published on the internet-facing interface | internet, unauthenticated | **yes, on the KEV list** | 7.4.7 | **critical, now** |
| 2 | EOS-SA-EX-014 — control-plane DoS via crafted BGP UPDATE | 2 spine | eBGP configured to two transit providers; no GTSM on either session | transit peers | no | 4.31.5M | high |
| 3 | (no advisory) — 2 devices past software maintenance | 2 access | No fixes will ever ship for IOS 15.2(7)E; one sits in the DMZ rack | see finding 3 | n/a | none exists | high, replacement |

### 1. FG-IR-EX-001 — pre-auth RCE in SSL VPN, on two internet-facing firewalls

This is the one. SSL VPN is configured on `fw-edge-01` and `fw-edge-02`, published on the internet-facing interface, and the flaw is reachable pre-authentication. The advisory is on the known-exploited list, which means this is being used right now against fleets that look like yours.

`fw-edge-03` and `fw-edge-04` run the same affected version and **do not** have SSL VPN configured — checked in the running config, not assumed. Not applicable to them.

Everything else in this report can wait for a scheduled window. This one should not.

**Not applicable to:** `fw-edge-03`, `fw-edge-04` — affected version, feature not configured.

### 2. EOS-SA-EX-014 — BGP UPDATE parsing DoS on both spines

Applies: eBGP is configured on `spine-01` and `spine-02` to two transit providers. The trigger is a crafted UPDATE from a peer, so the attacker position needed is "the transit provider, or someone who can inject into a session with one".

What raises it: neither session has GTSM/TTL security or session authentication, so the position needed is lower than it should be. What limits it: it is a denial of service, not code execution, and the two spines are redundant — though both are affected, so a simultaneous attack is an outage rather than a failover.

The 36 leaves match on version too. They run iBGP inside the fabric only, no external peers, so the reachable-from answer is "someone already inside your fabric", and at that point this advisory is not your problem. Dismissed for the leaves.

**Not applicable to:** 36 leaves — same version, no external BGP sessions.

### 3. Two devices past software maintenance, one in the DMZ rack

`sw-access-14` and `sw-access-21` run IOS 15.2(7)E. That train passed end of software maintenance in 2024. Cisco will still take a TAC call; Cisco will not issue a fix.

The scanner reported 31 criticals against these two. Individually most are not reachable — they are access switches with a restricted management plane. The finding is not any one of those 31. It is that the number only goes up, and none of them will ever be patchable. Every advisory published against this platform from now on lands here permanently.

`sw-access-21` is in the DMZ rack, which changes its position materially: its management plane is one misconfiguration away from a zone you do not trust. `sw-access-14` is in an internal wiring closet and is a lower priority for the same platform state.

This is a replacement decision with a procurement lead time, not a patch. Raise it now so the budget conversation starts while the compensating controls still hold.

### The 3 devices nobody could reach

`sw-access-07`, `sw-access-19`, `rtr-branch-03` are in NetBox and did not answer live collection. They are recorded as their last-known version, which was last confirmed in 2025-11. They are the only devices in this report whose risk is genuinely unknown, and one of them is a branch router.

Find out what they are running before acting on anything below finding 1. An unreachable device is either decommissioned and still in the inventory, or running and unmanaged — and the second is worse than any advisory here.

## Dismissed on triage

211 of 214 scanner criticals do not apply. Grouped:

| Reason | Count |
| --- | --- |
| Affected feature not configured on the device | 94 |
| Affected surface not reachable from any untrusted zone | 61 |
| Requires local/console access the attacker does not have | 28 |
| Advisory applies to a different hardware platform on the same software train | 19 |
| Already fixed in the running version; scanner matched the train, not the build | 9 |

The last row is worth a note: nine were false on their face. Your scanner is matching train strings rather than build numbers, which is a scanner configuration issue and is inflating every report it gives you.

## Upgrade plan

| Wave | Devices | Current → target | Why this target | Compatibility notes | Window | Rollback |
| --- | --- | --- | --- | --- | --- | --- |
| 0 (out of band) | `fw-edge-02` then `fw-edge-01` | 7.4.3 → 7.4.7 | Lowest version fixing FG-IR-EX-001; known-issues list checked, nothing affecting your model or feature set | HA pair supports mixed version for the upgrade only; passive first, fail over, then active. **Not both in one window.** | tonight, emergency | Previous image retained on flash; config compatible both directions |
| 1 | `spine-02` then `spine-01` | 4.31.2F → 4.31.5M | Arista recommended release for this platform; fixes EOS-SA-EX-014 | Same train, EVPN fabric tolerates mixed 4.31.x. Soak spine-02 a full business day before touching spine-01. | Tue window | Previous image on flash |
| 2 | 8 leaves | 4.28.4M → 4.31.5M | Consolidates the fabric onto one version and gets ahead of the 2026-Q4 maintenance end | Train hop, not a point release — read the 4.31 release notes for config syntax changes before the first one. One leaf, soak a week, then the rest. | Thu windows, over 3 weeks | Previous image on flash |
| 3 | 9 IOS-XE | 17.6.4 → 17.9.x | No applicable advisory; this is lifecycle hygiene ahead of the 2027-Q1 date | Not urgent. Do it after waves 0–2 land and not in the same period. | scheduled, Q4 | Archive + `configure replace` |

**In-service upgrade:** available on the spines in principle, and I would not rely on it for the 4.28 → 4.31 hop in wave 2. Plan those windows as reloads; treat a hitless upgrade as a bonus.

Every wave goes through `change-window` when you run it. Wave 0 in particular is an out-of-hours change to the device that terminates your remote access — check the severance path before you start, because the VPN you would use to fix a failed upgrade is the feature being upgraded.

## Cannot upgrade

| Device | Why | Compensating control | Residual risk | Decision owner |
| --- | --- | --- | --- | --- |
| `sw-access-21` (DMZ rack) | Past software maintenance; no fixes exist | Management plane restricted to the jump host subnet; disable the unused HTTP server and legacy discovery protocol; alert on any management-plane auth from outside that subnet | Any future advisory in the remaining management services is permanently unpatchable. Position in the DMZ rack makes this materially worse than its twin. | needs an owner and a replacement date |
| `sw-access-14` (internal closet) | Same platform state | Same controls | Same, lower exposure by position | same |

Both controls are holding actions with no end date attached, which is how they become permanent. Put a replacement date on them in this quarter's planning.

## Making it durable

1. **Fix the scanner's version matching** — build numbers, not train strings. Nine false criticals in one run is a persistent tax on every report it produces.
2. **NAPALM facts into NetBox on a schedule.** Version data was the most expensive part of this audit and should be a query. It would also have surfaced the three unreachable devices in November rather than today.
3. **Subscribe to the three vendors' advisory feeds and match against NetBox automatically.** The goal is that the next run of this skill starts at triage.
4. **Alert on the software maintenance dates already in the table above.** The 8 leaves at 4.28 have a 2026-Q4 date; that should surface as a ticket in Q3, not as a finding in a future audit.

*(calls `buddy_observe`, `summary: "Lifecycle audit across 61 devices: 3 of 214 scanner criticals applicable, worst was a KEV-listed pre-auth SSL VPN RCE on two internet-facing firewalls."`, `kind: "config"`, `skills_used: ["dragon-dev-buddy:device-lifecycle"]`, relays the reaction)*

---

## What this run got right

- **Reconciled the inventory before analysing it**, and treated the three unreachable devices as a finding rather than rounding the fleet down to what answered.
- Triaged every advisory on whether the feature was **configured** and the surface **reachable**, checking running config rather than inferring from a version string — which is what separated two affected firewalls from two unaffected ones on the same build.
- Reported the **211 dismissals with reasons and counts**. That section is what makes a list of three believable to the person who was handed a list of 214.
- Caught a scanner defect (train-string matching) as a durable finding rather than silently correcting nine entries.
- Distinguished **end of software maintenance** from end of support, and reframed those two switches as a procurement decision with a lead time instead of putting 31 unpatchable criticals in the patch queue.
- Used **position** to separate two identical devices — DMZ rack versus internal closet — instead of giving the same platform state the same priority twice.
- Sequenced the upgrades with the passive unit first, an explicit soak, and a warning that the wave 2 train hop is not a point release.
- Flagged that wave 0 upgrades the **VPN you would use to recover a failed wave 0**, and routed it to the gate that handles that.
- Attached an owner and the absence of a replacement date to the compensating controls, so they cannot quietly become the permanent answer.
