# Lifecycle and advisory sources

## Where advisories and lifecycle dates come from

Prefer the machine-readable feed over the web page for anything you will repeat. The point of wiring these up is that the next audit starts at triage instead of at data collection.

| Vendor | Advisories | Lifecycle dates | Machine-readable |
| --- | --- | --- | --- |
| Cisco | PSIRT security advisories | EoX bulletins per product | openVuln API (advisories by product/version) and the EoX API. Both need a developer account; both are worth the setup. Cisco also publishes a twice-yearly bundled IOS/IOS-XE advisory set — audit around those dates. |
| Juniper | JSA advisories | EoL/EoE tables per platform and per Junos train | Advisory feed, plus the Junos software EoL table. Note that Junos support runs per *train* — the same box on a different train has a different end date. |
| Arista | Security advisories, numbered | Software lifecycle page, per EOS train | Advisory list and per-release lifecycle. Arista publishes a recommended release per platform — use it as the default target. |
| Palo Alto Networks | PAN-SA / CVE advisories | PAN-OS release lifecycle with per-version end dates | Security advisories feed and the release-lifecycle table. Preferred/recommended releases are published and are usually the right target. |
| Fortinet | FortiGuard PSIRT advisories | Product lifecycle by model and FortiOS branch | PSIRT feed. Fortinet advisories frequently include workarounds — read them; they are often the compensating control you need. |
| HPE / Aruba | Security bulletins | Product end-of-support matrix | Bulletin feed per product family. |
| MikroTik, Ubiquiti, smaller vendors | Release notes and forum posts, frequently not formal advisories | Often none published | Assume no advisory feed exists and diff release notes. The absence of a PSIRT process is itself a risk input for the platform choice. |
| Anything with an embedded Linux | Vendor advisory plus the underlying component's own CVEs | Vendor's, not the component's | The vendor decides when a component fix ships. A patched OpenSSL upstream means nothing until the vendor rebuilds. |

**Cross-vendor signals**
- **CISA KEV (Known Exploited Vulnerabilities).** The strongest single reorder signal for network gear. Edge devices — VPN concentrators, firewalls, remote-access gateways — are among the most consistently exploited categories, and a KEV listing means the attack is real and running now, not theoretical. A KEV-listed advisory outranks an unlisted one with a higher score, every time.
- **NVD / CVE records.** Useful for cross-referencing, weak for applicability — the CVE record rarely says whether the affected feature is on by default.
- **Vendor mailing lists and RSS.** Low effort, high value; the alternative is finding out from a scanner two months later.
- **The vendor's own manager** (Panorama, FortiManager, DNA Center, CVP) frequently reports version compliance and advisory exposure across the fleet already. Check before building it.

## What each lifecycle stage means for risk

| Stage | What it means | Effect on the audit |
| --- | --- | --- |
| Fully supported | Fixes issued, TAC available | Normal patching. Advisories are queue items. |
| End of sale announced | Cannot buy more; support continues to a published date | No immediate risk change. Start the replacement conversation while it is cheap — the lead time is the point. |
| **End of software maintenance / end of vulnerability support** | **No further fixes, including security fixes** | **The stage that matters.** Every advisory published against this platform from now on is permanently unpatchable. Reframes the device from "patch it" to "replace or contain it". Frequently arrives years before end of support, and is the date people miss because support technically continues. |
| End of support | No TAC, no RMA, no fixes | Hardware failure and vulnerability both become unrecoverable. Usually also a compliance finding on its own. |

The distinction that gets missed: a device can be *in support* and past *software maintenance*. TAC will take your call and there will be no patch. Check the software maintenance date specifically, per train, not the headline support date.

## Applicability triage by advisory class

Version match is the beginning of the analysis. These questions are what turn 200 matches into the handful that matter.

**Management-plane advisories** (the web UI, SSH, SNMP, API, the CLI parser)
- Is the service enabled at all? Many are on by default and unused.
- What is the management ACL? An advisory in the web UI on a device whose management plane is restricted to one jump-host subnet is a very different finding from the same advisory on a device with the UI on an internet-facing interface.
- Is authentication required to reach the flaw? Pre-auth on an exposed management plane is the top of the list, always.
- Are there management interfaces you forgot — a secondary address, an in-band management path, a redirect from an old migration?

**Control-plane advisories** (BGP, OSPF, IS-IS, LLDP, STP, LACP, DHCP, IPv6 ND)
- Is the protocol configured on this device, and on which interfaces?
- Who can send it packets? A BGP parsing flaw is critical on an eBGP session to a transit provider and mostly theoretical on an iBGP session inside a controlled fabric.
- Is there protection in the path — a control-plane policer, GTSM/TTL security, session authentication, an infrastructure ACL? These frequently reduce a critical to a low, and the advisory will not tell you that.
- Adjacency-only protocols (LLDP, STP, LACP) require a foothold on the same segment. That raises the bar substantially and does not eliminate it — an access switch is exactly where an untrusted device plugs in.

**Data-plane advisories** (packet processing, tunnelling, NAT, inspection engines, VPN termination)
- Does traffic that reaches this feature originate anywhere untrusted?
- Is the affected feature configured? VPN termination flaws on a device not terminating VPNs do not apply.
- Can the trigger packet be constructed from outside your perimeter, or does it need to already be inside?

**Advisories in features you do not run**
This is the largest category by count and the reason fleet audits drown. Dismiss them explicitly, with the reason and the count, in the report. A ranked list of three is credible only when it is next to "197 dismissed, here is why".

## Upgrade sequencing

**Choosing the target**
- Take the vendor's recommended or preferred release for your specific platform, not the newest. Newest carries the bugs nobody has hit yet.
- **Read the known-issues list of the target before committing.** A release that fixes your advisory and introduces a forwarding bug on your line cards is a normal occurrence, not a rare one.
- Confirm the target actually fixes the advisory on your hardware — fix availability is per-platform and per-train, and the advisory's "fixed in" column often lists several trains, only one of which is yours.
- Minimise the number of distinct target versions across the fleet. Every extra version is another combination to test and another set of release notes to track.

**Compatibility during the rollout**
- **HA pairs and clusters** usually support a mixed-version state only for the duration of an upgrade, and sometimes only in one direction. Check whether your specific version hop is supported mixed, and how long it may remain so.
- **Fabric members** (VXLAN/EVPN, stacked switches, virtual chassis) have their own compatibility matrix. A stack that will not form with mixed members turns a rolling upgrade into an outage.
- **Management systems** have a supported device-version range. Upgrading a device past what the manager supports means losing management of it, which is a self-inflicted version of the severance problem.
- **Config transforms** applied on upgrade may not reverse. Where the new version rewrites config syntax, the downgrade path is a restore, not a reboot — establish this before, not during.

**Order**
- Least critical class first; within a class, one device soaked before the rest.
- Standby before active. Fail over deliberately, verify, then upgrade the former active. Never both halves in one window.
- Lab or a non-production device of the same model first, where one exists. Where one does not, the first production device is the lab, and it should be the least important one you have.
- Give the soak enough time for slow failure modes — memory leaks, table exhaustion, a daily process — to appear. An hour proves it booted; it does not prove the release is good.

**In-service upgrade caveats**
ISSU, NSF/GR, hitless upgrade and their equivalents are conditional far more often than the marketing implies. Before relying on one: confirm it is supported for your *specific* version hop (not just the platform), that your feature set is compatible (many features disqualify it), that both supervisors or engines are healthy, and what happens if it fails partway. A failed in-service upgrade is worse than a planned reload, because it happens unattended and in a state nobody prepared for. Plan the window as though you will reload, and treat a successful in-service upgrade as an unexpected bonus.

## Compensating controls when upgrade is not possible

Ranked roughly by effectiveness. State the residual risk after applying any of them — the point is informed acceptance, not the appearance of remediation.

1. **Remove the exposure.** If the affected feature is unused, disable it. This eliminates the finding rather than reducing it, and it is available more often than people check.
2. **Restrict the reachable surface.** Management-plane ACL to a single jump host, infrastructure ACL for control-plane protocols, an upstream filter for the affected protocol. Turns a pre-auth remote flaw into one requiring a foothold you can detect.
3. **Put something in front of it.** Terminate the affected protocol elsewhere, or filter it on an upstream device that is patched.
4. **Enable the vendor's published workaround.** Frequently exists, frequently has a functional cost — record the cost so it is not silently reverted later by someone who does not know why it is there.
5. **Increase detection.** Specific logging and alerting on the affected surface. The weakest control and the one to be honest about: it does not prevent anything. Only reasonable alongside a replacement date.
6. **Isolate and schedule replacement.** For anything past software maintenance in an exposed position, this is the actual answer. Everything above is what holds until the replacement lands, and each one needs a date attached or it becomes permanent.
