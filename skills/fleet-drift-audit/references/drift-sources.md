# Drift sources, normalization and what actually matters

## Where to get running config

| Source | What it gives you | Caveats |
| --- | --- | --- |
| Oxidized / RANCID | A git history of every device's running config, pulled on a schedule | The history is the real prize — it answers *when* the drift appeared and often *who*. Check the last successful pull per device; a device that has failed collection for months looks perfectly compliant because its stored config is frozen. |
| NAPALM | Normalized config and facts across vendors | `get_config()` returns running, startup and candidate separately. Compare the right one; a device whose running and startup differ is its own finding — the drift disappears on next reboot, or the fix does. |
| Ansible network facts / `*_config` in check mode | Live config plus a diff against the intended play | Check mode is the cheapest recurring drift detector you already own. Only as good as the play's coverage: config the play never touches shows as compliant. |
| NetBox / Nautobot | The *intended* state, and often a rendered config | This is the strongest reference when it is maintained. Verify it is maintained — an inventory nobody updates makes drift look like fleet failure when the source of truth is the stale party. |
| Vendor managers (Panorama, FortiManager, DNA Center, CVP) | Managed config plus out-of-band change detection | Most of these already flag local changes made outside the manager. Read that report first; it is drift detection you are paying for. |
| Direct SSH / API pull | Ground truth | Slowest, rate-limited, and on a large fleet a source of load. Fine for a class, painful for a fleet. Prefer a backup system and fall back to this for spot checks. |
| AWS Config / Azure Resource Graph / gcloud asset inventory | Cloud resource state plus change history | The cloud analogue of a config backup, and the change history is queryable. Pair with `terraform plan -refresh-only` to see drift against declared intent. |
| `kubectl diff` / Argo CD or Flux drift status | Live cluster versus declared manifests | GitOps tools report drift natively; read their status before writing your own diff. Watch for resources excluded from sync — they are unmanaged by definition and never appear as drift. |

## Normalization: what to mask before diffing

Diff noise is the reason drift audits get abandoned. Mask these, then diff.

**Universally per-device — always mask**
- Hostname, and the hostname embedded in prompts, banners, certificate subjects and syslog source strings.
- Management and interface addresses, router IDs, loopbacks.
- Serial numbers, chassis IDs, MAC-derived identifiers, license keys.
- Timestamps, uptime, "last changed" comments, config save markers and version counters.
- Encrypted or hashed secrets that re-encrypt differently on every write — mask the value, but **compare presence and type**. That a device has a local user is drift; that its hash string differs is not.
- Neighbor-specific state: BGP peer addresses and ASNs where they are legitimately per-device, LLDP tables, learned entries.

**Platform-specific**
- **Cisco IOS/NX-OS** — `ntp clock-period`, crypto RSA key blobs, `boot-start-marker` blocks, `!Time:` headers, certificate chains. `show running-config` ordering is stable enough to diff directly once these are gone.
- **Junos** — prefer `show configuration | display set` for line-wise diffing; hierarchical output makes diffs report a whole block for a one-line change. Mask `## Last commit` headers and `apply-flags omit` markers.
- **Arista EOS** — mask the `! device:` header and `!! ` comment lines. `show running-config sanitized` handles most secret material for you.
- **PAN-OS / FortiOS** — export as XML/CLI rather than the GUI dump; mask UUIDs on rules (regenerated per device even for identical rules), session counters, and `set date`/`set time` lines. Rule UUIDs differing while rule *content* matches is the classic false positive here.
- **Terraform state** — compare plan output, never state files. State carries per-resource IDs and timestamps that guarantee a diff.

**Order sensitivity — get this right or the audit lies in both directions**
- **Order matters, compare in order:** ACL and firewall rule entries, route-map and prefix-list sequences, policy chains, NAT rules, class-map matching. Two devices with the same rules in a different order are genuinely different devices.
- **Order does not matter, compare as a set:** SNMP hosts, NTP servers, syslog targets, AAA server lists (mostly — check whether the platform treats list order as preference), VLAN definitions, user accounts, static route sets with distinct prefixes.

## Security-relevant drift catalogue

Drift belongs in the security bucket when it changes what the device permits, who can administer it, or whether you would find out. Look specifically for:

**Who can get in**
- A local account present on some devices and not others — especially one not in the reference at all. Old break-glass accounts outlive the person who made them.
- AAA server list differences: a device pointing at a decommissioned TACACS+/RADIUS server, or one with a fallback to local auth that its peers do not have.
- Missing AAA entirely on a device, so it is authenticating locally while the fleet authenticates centrally.
- SSH key differences in `authorized_keys` equivalents, and keys with no owner.
- Differences in the management-plane ACL — the subnet allowed to reach SSH/API/SNMP.

**What is listening**
- Management services enabled on some devices and not others: telnet, HTTP (not HTTPS), SNMP v1/v2c, legacy discovery and provisioning protocols, debug or diagnostic services left on after a support case.
- Service versions and cipher/protocol configuration on the management plane.
- An interface in the wrong VRF or zone, exposing the management plane to a network the reference does not.

**What it forwards**
- Extra ACL or firewall entries present on a subset — the single most common security-relevant drift, and usually the residue of a temporary permit added during an incident.
- A missing deny that the rest of the class has.
- Differences in uRPF, storm control, port security, DHCP snooping, dynamic ARP inspection, BPDU guard — the anti-spoofing and anti-disruption layer decays quietly because nothing breaks when it is absent.
- Routing policy differences: a missing inbound prefix filter on one peering session is invisible until the day it is not.

**Whether you would find out**
- Syslog or SNMP trap targets missing or pointing somewhere dead. A device that logs nowhere is where an attacker would want to be.
- NTP differences — a device with the wrong time produces logs that cannot be correlated, which quietly degrades every investigation you will ever run.
- Differences in logging level or in what is logged for the management plane specifically (login success/failure, config change).
- Flow export (NetFlow/sFlow/IPFIX) configured on peers but not here.

**Fleet-wide, therefore invisible to a diff**
Run these as explicit checks, because a setting that is uniformly bad produces no drift at all: SNMP v2c everywhere, a shared local password across the fleet, no management-plane ACL anywhere, telnet enabled fleet-wide. Report these separately as a posture finding and hand to `hardening-playbook`.

## Sampling when you cannot reach everything

Full coverage is best. When it is not available:

- **Stratify by class, then by position.** Sample every class; within a class weight toward the devices closest to untrusted networks.
- **Always include the outliers you already know about**: the oldest hardware, the most recently touched, the ones a recent incident involved, anything commissioned outside the normal process.
- **Include every device the backup system has failed to collect from.** These are not gaps in your data; they are the finding. A device that has not been backed up in six months is unmanaged.
- **State the coverage in the report.** "42 of 380 devices, stratified by class" is a usable claim. Presenting a sample as a fleet audit is not.

## Making it stick, ranked by effort

1. **Alert on unexpected change in the config backup.** If you run Oxidized or RANCID, the diff already exists and a webhook on unexpected commits is an afternoon. Cheapest real detection there is.
2. **Scheduled orchestrator run in check mode.** Nightly Ansible in `--check --diff` against the intended play, output to a report. Catches drift on everything the play covers.
3. **Render-and-compare in CI.** Render intended config from the source of truth on every template change and diff against the fleet's last known state. Catches template regressions before they are pushed.
4. **Make the source of truth authoritative.** Push-only from the source of truth, with local changes reverted rather than merged. The strongest option and the largest cultural change — it fails if there is no fast path for emergency changes, because people will keep making them at 03:00 and now the automation fights them.

Whichever you recommend, say what it would have caught in *this* audit. A durable mechanism justified by a concrete finding gets built; one justified by principle does not.
