# Worked example: the firewall was fine, the boundary was not

Shows the step that earns this skill its place: the firewall between prod and dev enforced the intended policy correctly, and three other paths crossed the boundary without it.

---

**User:** we're being asked to prove prod is isolated from dev for our PCI assessment. can you review the firewall rules between them?

**Assistant** *(reads `.dragon-buddy/config.json` — `security.exposure: "public"`, `data_sensitivity: "financial"`, `compliance: ["PCI DSS"]`, `fleet.vendors: ["FortiOS 7.4", "AWS"]`, `engagement.authorized_scope: []`)*

Two things before I start.

First, you asked for a firewall rule review and the assessment question is broader: not "are these rules correct" but "is prod reachable from dev by any means". I will do the ruleset review, and I am also going to enumerate every path between the two zones, because a correct ruleset on one path is not isolation. If the other paths turn out clean the report is stronger for having listed them — that list is most of what an assessor actually wants.

Second, `engagement.authorized_scope` is empty. This is your own infrastructure, so I can test reachability if you authorize it — but tell me explicitly, and tell me whether we are testing in prod or somewhere equivalent.

**User:** yep our own kit, go ahead, prod is fine, just don't scan anything. and yeah — enumerate everything, that's the actual question.

---

# Segmentation review: prod ⇄ dev
2026-08-06 · exposure public · data financial · compliance PCI DSS

## Intended policy

Confirmed with the user before analysis:

- **No traffic from dev to prod.** None. Not for deployment, not for debugging, not for monitoring.
- **Prod to dev is also unnecessary**, though nobody had considered the direction before I asked.
- Deployment to prod flows from CI, which is understood to be "outside both" — an assumption examined in finding 2, because it is not correct.

## Paths found

| Path | Enforcement point | Assessed | Verdict |
| --- | --- | --- | --- |
| Routed, dev VLAN 200 → prod VLAN 100 | `fw-core-01` FortiOS policy | yes | **enforces intent correctly** |
| Backup network, VLAN 900 | none | yes | **bypasses the boundary entirely** |
| CI runner (`runner-03`) | AWS security groups | yes | **reaches both zones** |
| Hypervisor management, VLAN 910 | management ACL on `sw-mgmt-01` | yes | reaches both; ACL restricts to the admin subnet, holds |
| Shared AD (`dc-01`, `dc-02`) | n/a | yes | both zones authenticate against it; noted, not a packet path |
| Log collector, agents outbound to `siem-01` | none needed | yes | one-directional, no return path, no finding |
| VPN remote-access pool | FortiOS policy | **no** | not assessed — see "Not assessed" |

## Findings (ranked)

| # | Finding | Path | Pattern | Exposes | Evidence |
| --- | --- | --- | --- | --- | --- |
| 1 | Backup VLAN is trunked to hosts in both zones with no filtering | backup net | implicit reliance | Any dev host reaches prod hosts on the backup interface, unfiltered, including the cardholder database | `sw-core-01` trunk config + confirmed by test |
| 2 | CI runner has security-group reach into prod and dev, and dev developers can queue jobs on it | CI | overly broad | A dev-authored pipeline job executes with network reach into prod | `sg-runner` rules + runner job config |
| 3 | Orphaned FortiOS rule permits a reissued dev address into prod | routed | orphaned | `10.20.4.31` is permitted to prod on 5432; that address is now a dev workstation | policy id 47, DHCP lease table |

### 1. The backup network crosses the boundary with nothing in the path

VLAN 900 is trunked to every host in both zones so that the backup agent can reach `backup-01`. Every prod and dev host has an interface on it. There is no ACL on the VLAN, no filtering on the switch, and `fw-core-01` is not in the path — the firewall sits between VLAN 100 and VLAN 200, and VLAN 900 is neither.

The firewall policy you asked me to review is correct. It is also not carrying this traffic.

**Evidence:** `sw-core-01` shows VLAN 900 in the allowed list on the trunks to both zones' hosts. Authorized test, single connection, from `dev-app-02` (10.20.4.55, backup interface 10.90.0.55):

```
$ nc -vz 10.90.0.12 5432
Connection to 10.90.0.12 5432 port [tcp/postgresql] succeeded!
```

`10.90.0.12` is `prod-db-01`'s backup interface. That is the cardholder database, reached from a dev application server, in one hop, with no filtering. One connection, logged, torn down immediately; nothing scanned.

For the assessment this is dispositive: prod and dev are not segmented, whatever the firewall policy says.

**Fix:** the backup network is a segmentation boundary and needs to be treated as one. Either put an ACL on VLAN 900 permitting only host-to-`backup-01` traffic and denying host-to-host, or split it into per-zone backup VLANs with separate targets. The ACL is smaller and reversible; prefer it.

### 2. The CI runner is in both zones, and dev can queue jobs onto it

`runner-03` is in `sg-runner`, which permits egress to both the prod and dev application security groups. That by itself is normal for a deployment runner. The problem is the other end: the runner accepts jobs from any branch of any repository in the dev project, and dev developers can push a branch.

The path is: a developer writes a pipeline step, pushes a branch, the job runs on `runner-03`, and that step has network reach into prod. No firewall rule is violated, because the runner is authorized to reach prod — that is its purpose.

Your intended policy said CI is "outside both zones". It is inside both.

**Evidence:** `sg-runner` egress permits `sg-prod-app` on 443 and 5432, and `sg-dev-app` on the same. AWS Network Access Analyzer confirms a path from `runner-03` to `prod-db-01`. Job config permits `branch: *` from the dev project.

Not tested actively — proving it means executing code in your pipeline against prod, which is a bigger action than this review should take. The config path is unambiguous.

**Fix:** separate runners per environment, prod runners restricted to protected branches with required review. Do not fix this by narrowing the security group; the reach is legitimate, the authorization to use it is not.

### 3. Policy id 47 permits an address that now belongs to a dev workstation

FortiOS policy id 47 permits `10.20.4.31` → `10.10.1.20:5432`, named `temp-migration-access`, created 2024-11-19.

`10.20.4.31` was the migration jump host. That host was decommissioned in March. The address returned to the dev DHCP pool and has been leased to a developer workstation since 2026-06-02.

The rule has not been used since March per the FortiOS hit counter, so this is not evidence of misuse. It is a permit into the cardholder database that currently belongs to whoever picks up that lease.

**Evidence:** policy id 47; DHCP lease table shows `10.20.4.31` leased to `LT-4471`, renewed 2026-08-05.

**Fix:** remove policy id 47. Then check the rest of the ruleset for the same pattern — six other rules carry `temp-` names, and four of them predate 2025.

## Not assessed

- **The remote-access VPN pool.** Clients receive addresses from one pool regardless of role, and the FortiOS policy for that pool was out of the time we had. Given findings 1 and 2 I would expect it to matter. It should be the next thing reviewed, and until it is, this report does not establish isolation on that path.
- **Physical and wireless access** to ports in either zone. Out of scope for a network review; flagged because a PCI assessor will ask.
- **Shared AD** is listed as a path but not assessed as one. It is not a packet path, and compromise of `dc-01` reaches both zones. It belongs in `threat-model`, not here.

## Remediation order

Ordered so nothing legitimate breaks, and so the reviewer's own access survives:

1. **Remove policy id 47** — no hits since March, no live dependency, immediate risk reduction. Smallest change first.
2. **ACL on VLAN 900** permitting host→`backup-01` and denying host→host. Add the permit, verify backups complete a full cycle, *then* add the deny. Do not do both in one push; a backup network that stops working is discovered days later, at the worst moment.
3. **Split the CI runners.** Largest change, needs pipeline work, no safe partial state — do it as its own piece with prod runners standing up alongside the existing one before it is retired.
4. **Sweep the remaining `temp-` rules**, one at a time, checking hit counters and address reassignment for each.

All four are changes to enforcement points between production zones. Run them through `change-window` — item 2 in particular changes a VLAN that every host in both zones has an interface on.

*(calls `buddy_observe`, `summary: "Segmentation review of prod/dev boundary: 7 paths found, 3 findings, worst was an unfiltered backup VLAN giving dev hosts direct reach to the cardholder database."`, `kind: "config"`, `skills_used: ["dragon-dev-buddy:segmentation-review"]`, relays the reaction)*

---

## What this run got right

- **Widened the question before answering it**, and said why. The user asked about firewall rules; the assessment question was isolation, and the firewall turned out to be the one path that was fine.
- Got explicit authorization for active testing and pinned the environment, then kept the test to a **single connection** — enough to prove the finding, not enough to page anyone.
- Enumerated paths that have no firewall on them, which is the only reason findings 1 and 2 exist.
- Checked the orphaned rule for **address reassignment** rather than stopping at "stale rule". Stale is housekeeping; reissued to a live workstation is a permit into the cardholder database.
- Declined to actively test finding 2 and said so, rather than either overclaiming from config or executing pipeline code against prod to be thorough.
- Refused the tempting fix on finding 2 — narrowing the security group would look like remediation while leaving the actual problem, which is who can queue a job.
- **Listed what was not assessed**, so the report cannot be read as proof of isolation on the VPN path.
- Ordered remediation smallest-first and split the VLAN ACL into permit-then-verify-then-deny, because a silently broken backup network is discovered late.
