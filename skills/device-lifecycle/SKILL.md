---
name: device-lifecycle
description: Audits firmware and OS versions across a fleet against vendor advisories and lifecycle dates, triages which advisories are actually applicable, and produces an upgrade sequence that will not take the network down. Use when someone says "are our switches patched", "firmware audit", "EOL devices", "end of support", "PSIRT advisory", "is this device still supported", "what version are we running", "upgrade plan for the fleet", "vendor advisory came out", "do we need to patch this router", or when a scanner reports every device as critical. An advisory in a feature you do not run is noise; a device past end of software maintenance is permanently unpatched.
---

# Device Lifecycle

This is the dependency audit for things with a serial number. The shape is familiar — inventory, advisories, triage, upgrade plan — and three things make it harder than its software equivalent. There is no lockfile, so the inventory is usually wrong. The upgrade is itself the leading cause of outages, so the fix carries more risk than the finding. And a device past end of software maintenance has no patch to apply, which turns every future advisory against it into permanent exposure rather than a queue item.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `fleet.inventory_source`, `fleet.device_classes`, `fleet.vendors`, `fleet.change_window`, `security.exposure`, `security.compliance`, `output.reports_dir`.

## Inputs

Ask only for what is missing:
- **The fleet, and where version data comes from.** An inventory system, an orchestrator's facts, a vendor manager, or SSH. State which — a spreadsheet last updated in January is a different confidence level from live facts, and every finding inherits it.
- **The trigger.** A specific advisory just published, a periodic audit, a compliance requirement, or an upgrade being planned. A single advisory means start at step 3 and work outward; the rest start at step 1.
- **What upgrades are actually possible.** Maintenance contract status, change windows, whether anyone has done an upgrade on this platform recently. A plan that assumes support entitlement the user does not have is not a plan.

## Workflow

1. **Build the version inventory, and rate its confidence.** Platform, model, current version and train, per device. Pull it live where you can; where you cannot, say so per device rather than fleet-wide. Devices missing from the inventory are the highest-risk population in the audit — an unknown version is not a passing grade — so list them explicitly rather than dropping them.

2. **Map every device to a lifecycle stage, not just a version.** The stages that matter are: fully supported, end-of-sale announced, **end of software maintenance**, and end of support. The one that changes the analysis is end of software maintenance: after that date the vendor issues no more fixes, so every advisory that lands against that platform from then on is unpatchable by definition. That converts a device from "needs patching" to "needs replacing or containing", which is a budget conversation with a lead time — flag it early, not in the same list as this quarter's patches.

   End-of-life is a risk multiplier rather than a finding on its own. An EoL switch in a locked closet with no management-plane exposure is a lower priority than a supported edge firewall with an actively exploited advisory. Say the multiplier; do not lead with it.

3. **Match advisories to the inventory, then triage each one for applicability.** The match produces noise at a ratio that makes fleet audits useless if you stop there — a scanner reporting 400 criticals is reporting version strings, not risk. For every advisory that matches a version, ask:
   - **Is the affected feature enabled on this device?** Most advisories are conditional on a feature, protocol or service being configured. Check the running config, do not assume.
   - **Is the affected surface reachable, and from where?** Management-plane, control-plane and data-plane advisories have completely different exposure depending on where the device sits and what the management ACL permits. This is where `segmentation-review`'s findings feed in directly.
   - **Does exploitation require authentication or adjacency?** Pre-auth and internet-reachable is a different class from post-auth on a management interface restricted to one subnet.
   - **Is it known exploited?** For network devices this signal is strong enough to reorder the list on its own — edge devices are among the most actively exploited categories, and an advisory on the known-exploited list outranks a higher-scored one that is not.

   `references/lifecycle-sources.md` has the per-vendor advisory sources, the applicability questions by advisory class, and the machine-readable feeds worth wiring up. For a single advisory the user is asking about in depth, hand to `vuln-triage` — this skill decides which of many matter, that one decides whether a specific claim is real.

4. **Rank by exposure and position, not by score.** Combine what triage established with where the device sits and what is behind it. The same advisory on an internet-facing firewall and on an access switch in a locked room are not the same finding, and a report that lists them together at the same severity is the reason nobody acts on these.

5. **Sequence the upgrades, and treat the sequence as the deliverable.** For network devices the upgrade plan is harder and more valuable than the finding list, because upgrades cause more outages than the vulnerabilities they fix. Cover:
   - **Target version selection.** Not the newest — the one the vendor recommends for your platform, with the known-issues list read before committing. A release that fixes your advisory and introduces a bug on your hardware is common enough to check every time.
   - **Compatibility between peers.** Control-plane protocols, cluster and HA pairs, fabric members, and management systems each have version-compatibility constraints. A mixed-version state exists during the upgrade whether or not it is supported; find out which it is.
   - **Order.** Standby before active, least critical class first, one device soaked before the class. Never both halves of a pair.
   - **Whether in-service upgrade is genuinely available.** It is frequently advertised, conditional in practice, and silently unavailable for the specific hop you need.
   - **Rollback.** The previous image still on the device, the config compatible with both versions, and a known downgrade path — some upgrades are one-way once a config transform runs.

   Each upgrade goes through `change-window` when it is actually performed.

6. **Give compensating controls for what cannot be upgraded.** A device that cannot be patched is not automatically accepted risk. Restrict the management plane to a jump host, disable the affected feature if it is unused, put the device behind something that filters the affected protocol, or increase monitoring on it. State the residual risk that remains after the control, so the acceptance is informed.

7. **Make the inventory durable.** Most of this audit's cost was assembling version data that should have been a query. Recommend the cheapest mechanism that keeps it current — orchestrator facts on a schedule into the inventory system, a vendor manager's own reporting, or an advisory feed matched against the inventory automatically. The next audit should start at step 3.

## Output format

```markdown
# Device lifecycle audit: [scope]
[date] · [n] devices · [n] platforms · inventory confidence: [live / recent / stale, per source]

## Lifecycle exposure
| Platform | Devices | Version | Stage | Software maintenance ends | Position |
|          | [n]     |         | supported / EoS announced / **past software maintenance** / EoSupport | [date] | [edge/core/access] |

## Findings (ranked)
| # | Advisory | Devices | Applicable because | Reachable from | Known exploited | Fixed in | Priority |

### [#] [advisory] — [what it is]
[what the advisory does, why it applies here specifically — the feature is enabled and the surface is reachable — and what an attacker gets]
**Not applicable to:** [devices that matched on version but failed triage, and why]

## Dismissed on triage
[advisories that matched a version and do not apply, grouped by reason — the count matters, it is what makes the ranked list credible]

## Upgrade plan
| Wave | Devices | Current → target | Why this target | Compatibility notes | Window | Rollback |

## Cannot upgrade
| Device | Why | Compensating control | Residual risk | Decision owner |

## Making it durable
[how version data and advisory matching stay current]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Lifecycle audit across <n> devices: <n> of <n> advisories applicable, worst was <one clause>."`
- `kind`: `config`
- `skills_used`: `["dragon-dev-buddy:device-lifecycle"]`

Relay the reaction verbatim.

## File output

The report to `output.reports_dir` as `YYYY-MM-DD-device-lifecycle-<scope>.md`. Keep the dismissed-on-triage section in the report — it is the evidence that the ranked list is short because the analysis was done, not because the search was shallow, and it is what stops the same advisories being re-triaged next quarter. This skill reads versions and configs and writes a plan; it does not upgrade anything. Execution goes through `change-window`.

## Reference library

Load these for depth when the task calls for it:
- `references/lifecycle-sources.md`: advisory and lifecycle data sources per vendor including the machine-readable APIs, what each lifecycle stage actually means for risk, applicability triage by advisory class, upgrade sequencing rules and in-service upgrade caveats, and compensating controls for devices that cannot be patched.

## Worked example

See `examples/device-lifecycle-run.md` for an audit that cuts a scanner's 200-critical list to three that matter and sequences the upgrades. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- The version inventory states its confidence per source, and devices missing from it are listed as the highest-risk population rather than omitted.
- Every device is mapped to a lifecycle stage, and end of software maintenance is distinguished from end of sale and end of support.
- Devices past software maintenance are framed as permanently unpatchable and escalated as a replacement decision, not queued with this quarter's patches.
- Every advisory was triaged for whether the feature is enabled and the surface reachable, not just matched on a version string.
- Dismissed advisories are reported with their reason and count, so the short list is credibly short.
- Ranking reflects device position and reachability, not the vendor's score alone, and known-exploited advisories are surfaced as such.
- The upgrade plan names a specific target version, says why that one, and cites the known-issues check.
- Sequencing never puts both halves of a redundant pair in one wave, and mixed-version compatibility during the rollout is addressed explicitly.
- Anything that cannot be upgraded has a compensating control and a stated residual risk with an owner.
