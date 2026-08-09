---
name: fleet-drift-audit
description: Finds where devices that should be identical are not, separates deliberate variation from decay, and says which drift actually changes your security posture. Use when someone says "config drift", "these boxes should be identical", "why is this switch different", "audit the fleet", "compare configs across", "golden config", "did someone change this by hand", "we have hundreds of devices and no idea what's on them", "drift report", or after any outage traced to one device being unlike its peers. Ranks by what the drift exposes, not by how many lines differ.
---

# Fleet Drift Audit

Every fleet drifts. Someone fixed something at 03:00 and it worked, the template was updated but eleven devices were unreachable that day, a vendor default changed between firmware versions. The audit is not hard because finding differences is hard — a diff finds thousands. It is hard because almost all of those differences are legitimate, and the three that matter are buried in them. This skill does the separation: what is meant to differ, what has decayed, and what has quietly changed the security posture of the fleet.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `fleet.inventory_source`, `fleet.config_backup`, `fleet.golden_config`, `fleet.device_classes`, `fleet.vendors`, `security.exposure`, `security.compliance`, `output.reports_dir`.

If `fleet.config_backup` is `none`, say so early and plainly: without a config backup system this audit is a point-in-time snapshot that will be stale within a week, and standing one up is worth more than the audit itself. Do the audit anyway, then recommend it in step 7.

## Inputs

Ask only for what is missing:
- **The population.** Which devices, and how you can reach their configs — a backup tree, an orchestrator, an API, or SSH.
- **The classes.** How devices are supposed to be grouped. Comparing a spine to a leaf produces noise, not drift.
- **The reference**, if one exists: a golden config, a template, an intended-config render. If there is none, say so — the audit still works, but the reference becomes "the majority", which is a weaker claim and gets treated as one.
- **What prompted this.** An outage, an audit, a migration, or general unease. Each front-loads a different part of the fleet.

## Workflow

1. **Establish the population and split it by class.** Drift only means something within a class. Group by role and platform — edge routers, spines, leaves, access switches, firewalls, and each vendor and major version separately. State the group sizes. A class of one is not a class; it is a snowflake, and worth naming as its own finding.

2. **Pick the reference for each class, and be honest about it.** In order of strength: a rendered intended-config from the source of truth, a maintained golden template, then the majority. Say which you used. Drift *from the reference* and drift *from correct* are different claims — a weak setting present on 100% of the fleet is not drift, it is a fleet-wide finding, and `hardening-playbook` is where it goes.

3. **Collect and normalize.** This step decides whether the audit is usable. Raw configs differ on things that are supposed to differ — hostnames, addresses, serials, certificate fingerprints, timestamps, counters, neighbor IDs, encrypted secret blobs that re-encrypt differently on every write. Mask those before diffing or every device shows as 100% drifted and the real findings drown. Ordering matters for some blocks and not others: an ACL is order-dependent and must be compared in order; an SNMP host list or an NTP server set usually is not. `references/drift-sources.md` has the per-platform normalization recipe and where to pull configs from.

4. **Classify every difference.** Four buckets, and every difference lands in exactly one:
   - **By design** — per-device by definition. Falls out of normalization; if it does not, the normalization is wrong.
   - **Sanctioned exception** — deliberate, with an owner and a reason on record. If nobody can produce the reason, it is not sanctioned, it is decay wearing a badge.
   - **Decay** — a manual change that was never fed back to the template. Not necessarily dangerous, always a signal: the config on the device is now authored in a place your automation does not read.
   - **Security-relevant** — drift that changes what the device permits, who can administer it, or whether you would find out. This is the bucket the report is about. `references/drift-sources.md` catalogues what belongs here.

5. **Rank by exposure, not by count.** Score each finding on what it exposes, how many devices carry it, and where those devices sit. One internet-facing router with a management service it should not be running outranks forty access switches with the wrong syslog facility, and the report must show that ordering. Use `security.exposure` and device position; a locked wiring closet and the transit edge are not the same risk.

6. **Split the remediation two ways.** Every finding resolves in one of two directions, and choosing wrong is the most common failure of a drift audit:
   - **Reconcile the device to the reference** — the device is wrong.
   - **Absorb the drift into the reference** — the *template* is wrong. When drift appears on most of a class, the fleet is usually telling you the template never matched reality. Pushing the template over it re-breaks whatever the manual change fixed, and it comes back next quarter.

   Say which for each finding, with the reason. Where remediation means pushing config, hand off to `change-window`; a bulk reconcile across a class is exactly the change shape that gate exists for.

7. **Close the loop, or the audit expires.** Drift returns the moment the audit ends unless something watches for it. Recommend the cheapest durable mechanism available: a scheduled config backup with a diff alert on unexpected change, a CI check that renders the intended config and compares, an orchestrator run in check mode on a schedule. One of these is worth more than repeating this audit twice a year.

## Output format

```markdown
# Fleet drift audit: [scope]
[date] · [n] devices across [n] classes · reference: [intended-config / golden template / majority]

## Summary
[what the fleet actually looks like versus what it is supposed to, in three sentences]

## Security-relevant drift (ranked)
| # | Finding | Devices | Class | Position | Exposes | Direction |
|   | [what differs] | [n of n, named] | [class] | [edge/core/access] | [what it lets happen] | reconcile / absorb |

### [#] [finding]
[which devices, what the reference says, what they say instead, and what that changes]
**Direction:** [reconcile device / absorb into reference] — [why]

## Decay (not security-relevant, still unmanaged)
[grouped, with device counts — these are where the next surprise comes from]

## Sanctioned exceptions
[the ones with an owner and a reason, recorded so the next audit does not re-flag them]

## Snowflakes
[devices in a class of one, or so far from their class they are effectively unmanaged]

## Making it stick
[the watch mechanism recommended, and what it would have caught here]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Drift audit across <n> devices: <n> security-relevant findings, worst was <one clause>."`
- `kind`: `config`
- `skills_used`: `["dragon-dev-buddy:fleet-drift-audit"]`

Relay the reaction verbatim.

## File output

The report to `output.reports_dir` as `YYYY-MM-DD-fleet-drift-<scope>.md`. Keep the sanctioned-exception list in the repo rather than only in the report — it is the thing that stops the next audit re-litigating the same eight devices. This skill reads configs and writes a report; it does not push config. Remediation goes through `change-window`.

## Reference library

Load these for depth when the task calls for it:
- `references/drift-sources.md`: where to pull running and intended config from per platform and per orchestrator, the normalization recipe for each vendor, the security-relevant drift catalogue, sampling strategy when you cannot reach every device, and the durable watch mechanisms ranked by effort.

## Worked example

See `examples/fleet-drift-audit-run.md` for an audit of a small fabric plus edge routers, including a case where the template was wrong rather than the fleet. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- Devices are compared within a class, never across classes, and the class sizes are stated.
- The reference is named and its strength is stated. "The majority" is not presented as if it were an intended config.
- Configs were normalized before diffing, and the report does not contain per-device values that were always going to differ.
- Every difference is classified into exactly one of the four buckets; nothing is left as an unexplained diff hunk.
- Findings are ranked by what they expose and where the device sits, not by how many lines differ or how many devices carry it.
- Each finding says whether to reconcile the device or absorb the drift into the reference, with the reason — and fleet-wide drift is treated as evidence the template is wrong.
- Fleet-wide weak settings are called out as a posture finding rather than silently passing because everything matches.
- A durable watch mechanism is recommended, with what it would have caught in this audit.
