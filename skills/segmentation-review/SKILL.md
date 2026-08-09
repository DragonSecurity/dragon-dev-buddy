---
name: segmentation-review
description: Establishes whether a network actually enforces the boundaries you believe it does. Enumerates every path between two zones, analyses each ruleset for the four ways rulesets decay, and where authorized, proves reachability instead of reading it. Use when someone says "review our firewall rules", "segmentation review", "are these ACLs right", "check our security groups", "is prod actually isolated from dev", "who can reach the database", "our network is flat", "audit the ruleset", "PCI segmentation", "east-west traffic", or "we think this VLAN is isolated". A boundary is only as strong as the path nobody remembered to check.
---

# Segmentation Review

Most segmentation reviews examine the firewall between two zones, find it well-configured, and conclude the zones are separated. They are wrong roughly as often as they are right, because the zones are also connected by the backup network, the hypervisor management interface, a monitoring agent that dials out, and a CI runner with credentials in both. The ruleset analysis matters. The path enumeration matters more, and almost nobody does it.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `security.trust_boundaries`, `security.exposure`, `security.data_sensitivity`, `security.compliance`, `fleet.device_classes`, `fleet.vendors`, `engagement.authorized_scope`, `engagement.out_of_scope`, `output.reports_dir`.

If `security.trust_boundaries` is empty, this review has no intended policy to test against. Derive a candidate in step 1 and get it confirmed, or run `threat-model` first if the system is unfamiliar enough that the boundaries are genuinely unknown.

## Inputs

Ask only for what is missing:
- **Which boundary.** Two zones, named. "Review our segmentation" with no boundary named becomes an unbounded audit; make the user pick the boundary that would hurt most if it were not real.
- **The intended policy**, in reachability terms: who should be able to reach what, from where, on what. If nobody can state it, that is the first finding, and you propose one.
- **The enforcement points**, as far as they are known: firewalls, ACLs, security groups, NACLs, network policies, VRFs, VLANs, service mesh policy.
- **Whether active reachability testing is permitted**, and against which environment. Reading config is always fine. Sending packets across a boundary in production is a change in behaviour, not just an observation.

## Workflow

1. **State the intended policy before looking at any rule.** Write down what should be reachable from where, on which ports, for what reason. This is the yardstick; without it you are describing a ruleset rather than assessing it. If nobody in the room can state the intent, record that as a finding in its own right — a boundary nobody can describe is a boundary nobody is maintaining — then propose one from the topology and get it confirmed.

2. **Enumerate every path between the zones. All of them.** This is the step that finds what the others miss. Go looking specifically for the paths that are not the obvious routed one: the management VLAN, the out-of-band network, the backup and replication network, the hypervisor or storage fabric, monitoring and agent traffic, a CI runner or jump host with reach into both, VPN and split-tunnel clients, cloud peering and transit gateways, a shared identity or secrets service, and any host with an interface in each zone. `references/ruleset-analysis.md` has the full checklist. Every path found gets analysed; a path found and skipped should be named as unassessed rather than quietly dropped.

3. **Analyse each ruleset for the four decay patterns.** Rulesets do not fail randomly; they fail in four recognisable ways:
   - **Overly broad** — `any`/`any`, a `0.0.0.0/0` where a host was meant, a whole protocol where one port was meant, a `/16` inherited from a range that was subnetted years ago.
   - **Orphaned** — a rule whose source, destination or purpose no longer exists. The most dangerous of the four, because an address freed by a decommission gets reissued, and the rule now permits whatever landed on it.
   - **Shadowed** — a rule that can never match because an earlier one covers it. Rarely a hole by itself; reliable evidence that nobody understands the ruleset, which predicts the other three.
   - **Implicit reliance** — what happens to traffic no rule matches, and whether the boundary depends on that default. A default-permit at the end of an isolation boundary is not a subtle finding, and it is common in rulesets that grew from a permissive starting point.

   Ruleset semantics differ enough between platforms that the same rule text means different things — stateful versus stateless, first-match versus priority-ordered versus additive-allow, per-interface versus per-zone. The reference has the per-platform semantics; get them right or the analysis is confidently wrong.

4. **Test the claim where you are allowed to.** A ruleset that reads correct and a packet that arrives are different facts, and the gap between them is where misapplied policies, asymmetric routing, and interfaces in the wrong zone live. Prefer non-intrusive evidence first: existing flow logs, connection logs, and the platform's own policy-lookup tools, which answer "would this be permitted" without sending anything.

   For active testing, check `engagement.authorized_scope`. On the user's own infrastructure, confirm the environment and scope and proceed. On anything belonging to someone else, an authorization reference is required before a single packet — no scope, no testing, and say so plainly rather than working around it.

5. **Rank findings by what they expose.** Score on what becomes reachable, how reachable it is (anonymous versus needing a foothold), what it crosses (a compliance boundary is not just a technical one), and how long it has been that way. A permissive rule between two zones of equal sensitivity is noise next to one narrow path from an untrusted zone into cardholder or credential scope.

6. **Write fixes as ruleset changes, ordered so applying them cannot lock anyone out.** Additive before subtractive: the specific permit exists before the broad one is removed. For anything with a live user population behind it, say what you expect to break and how you would find out before it does. Push through `change-window` — a ruleset tightening is precisely the change shape that gate exists for, and the boundary you are fixing may be the one your session traverses.

## Output format

```markdown
# Segmentation review: [zone A] ⇄ [zone B]
[date] · exposure [level] · data [sensitivity] · compliance [list or none]

## Intended policy
[what should be reachable, from where, on what — as confirmed, or as proposed and by whom confirmed]

## Paths found
| Path | Enforcement point | Assessed | Verdict |
| [routed via fw-01] | [PAN-OS policy] | yes | enforces intent |
| [backup VLAN] | [none] | yes | **bypasses the boundary entirely** |
| [...] | | no | [why not, so it is visible] |

## Findings (ranked)
| # | Finding | Path | Pattern | Exposes | Evidence |
|   | [what] | [which path] | broad/orphaned/shadowed/implicit | [what becomes reachable] | [config line, log, or test] |

### [#] [finding]
[the rule as written, what it permits beyond intent, and what an attacker in zone A reaches because of it]
**Evidence:** [file/device and line, flow log, or reachability test result]
**Fix:** [the ruleset change]

## Not assessed
[paths and enforcement points deliberately left out of scope, so the report does not read as complete when it is not]

## Remediation order
[ordered so no intermediate state locks anyone out; hand off to change-window]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Segmentation review of <boundary>: <n> paths found, <n> findings, worst was <one clause>."`
- `kind`: `config`
- `skills_used`: `["dragon-dev-buddy:segmentation-review"]`

Relay the reaction verbatim.

## File output

The report to `output.reports_dir` as `YYYY-MM-DD-segmentation-<boundary>.md`. Where the boundary is in compliance scope, keep the path enumeration in the report even for paths that turned out clean — the list of paths considered is what an assessor is actually asking for. This skill reads config and, where authorized, tests reachability; it does not change rulesets. Remediation goes through `change-window`.

## Reference library

Load these for depth when the task calls for it:
- `references/ruleset-analysis.md`: per-platform ruleset semantics and the mistakes each invites, the path enumeration checklist, detection recipes for the four decay patterns, non-intrusive evidence sources and policy-lookup tools per platform, and the authorization gate for active reachability testing.

## Worked example

See `examples/segmentation-review-run.md` for a prod/dev boundary where the firewall was correct and two other paths were not. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- The intended policy is written down before any rule is read, and an absent one is recorded as a finding rather than improvised silently.
- Paths between the zones were enumerated deliberately, including the non-routed ones, and the report lists them — not just the one with a firewall on it.
- Any path left unassessed is named as unassessed. The report never reads as complete when it is not.
- Ruleset semantics are correct for the platform: stateful versus stateless, match order, and what happens to unmatched traffic.
- Every finding names the pattern it is an instance of and cites evidence at a specific rule, log line or test result.
- Orphaned rules are checked for whether the address has since been reissued, not just noted as stale.
- Active testing happened only within a confirmed scope, and its absence is stated rather than papered over with a confident reading of config.
- Findings are ranked by what becomes reachable and to whom, not by rule count.
- Fixes are ordered so that applying them cannot cut off legitimate traffic or the reviewer's own access.
