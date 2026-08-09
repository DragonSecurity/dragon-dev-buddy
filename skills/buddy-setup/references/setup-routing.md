# Setup routing and config reference

## Which three skills to recommend

Match on the strongest signal, top to bottom. Stop at the first match.

**Responding to something right now** (user mentions a breach, an alert, an active exploit)
1. `incident-response` — contain first, understand later
2. `vuln-triage` — once contained, work out what was actually reachable
3. `secrets-and-config-audit` — assume credential exposure until proven otherwise

**Public internet + sensitive data** (exposure `public`, sensitivity `pii`/`financial`/`phi`/`credentials`)
1. `threat-model` — you need the map before the audit, or you will audit the wrong things
2. `secrets-and-config-audit` — highest hit rate per minute spent on any public repo
3. `hardening-playbook` — the shipped-vs-defensible gap is widest here

**Public internet, low sensitivity** (marketing site, docs, open-source library)
1. `dependency-audit` — supply chain is the realistic attack path
2. `secure-code-review` — on whatever handles input
3. `ship-it` — a deploy gate is cheap insurance

**Authenticated product, any sensitivity**
1. `threat-model` — authz is where these fail, and only a model finds authz bugs
2. `secure-code-review` — target the authorization layer specifically
3. `security-test-writer` — turn each authz rule into a test that fails when it regresses

**Internal only**
1. `secure-code-review` — on the highest-traffic path
2. `dependency-audit` — internal does not mean safe, it means less watched
3. `debug-and-fix` — most internal work is correctness, and correctness is a security property

**Greenfield, little code yet**
1. `threat-model` — cheapest it will ever be
2. `secure-feature-build` — put abuse cases in the spec before the first line
3. `buddy-companion` — get the reporting habit established early

**Manages a device fleet or network** (`fleet.managed` true)
1. `fleet-drift-audit` — you cannot secure a fleet whose actual state nobody knows
2. `segmentation-review` — on the boundary that would hurt most if it were not real
3. `change-window` — before the first change that comes out of either

Route to `device-lifecycle` ahead of these when the trigger is a published advisory
or a scanner report rather than general unease.

**Offensive engagement recorded** (`engagement.authorized_scope` non-empty)
1. `security-audit-orchestrator` — runs the full chain in dependency order
2. `vuln-triage` — for each finding as it surfaces
3. `pentest-report` — the deliverable, set up early so findings accumulate into it

## Inferring exposure when the user is unsure

Ask "if this went down at 3am, who notices first?" A customer means public or authenticated. A colleague means internal. Nobody means it may not need this pack at all.

For data sensitivity, ask what the worst row in the biggest table is. People describe schemas more honestly than they describe risk.

## Config keys and who reads them

| Key | Consumed by |
| --- | --- |
| `project.name`, `project.what_it_is` | every skill, for report headers and context |
| `project.primary_language`, `project.stack` | `secure-code-review`, `dependency-audit`, `security-test-writer` |
| `project.runtime`, `project.deploy_target` | `hardening-playbook`, `ship-it` |
| `security.exposure` | every skill that assigns severity. The single most important key. |
| `security.data_sensitivity` | severity ratings, `threat-model` impact scoring, `incident-response` notification duty |
| `security.compliance` | `pentest-report`, `incident-response`, `hardening-playbook` |
| `security.auth_model` | `threat-model`, `secure-code-review` |
| `security.trust_boundaries` | `threat-model` starts from these instead of deriving them |
| `security.known_risk_areas` | `security-audit-orchestrator` prioritizes these first |
| `engagement.*` | gates `vuln-triage` PoC output and `pentest-report` entirely |
| `fleet.managed` | whether the fleet skills apply at all; leave false for an ordinary app repo |
| `fleet.inventory_source`, `fleet.device_classes` | `fleet-drift-audit`, `device-lifecycle` |
| `fleet.config_backup`, `fleet.golden_config` | `fleet-drift-audit` — without a backup the audit is a snapshot that goes stale in a week |
| `fleet.vendors` | all four fleet skills, for ruleset and rollback semantics |
| `fleet.out_of_band` | `change-window`. The key that decides whether a self-severing change can be gated at all. |
| `fleet.change_window`, `fleet.approval` | `change-window`, `device-lifecycle` upgrade sequencing |
| `practice.test_command` | `security-test-writer`, `debug-and-fix`, `ship-it` |
| `practice.sca_tool` | `dependency-audit` |
| `practice.sast_tool`, `practice.secret_scanner` | `secrets-and-config-audit`, `secure-code-review` |
| `practice.ci` | `ship-it`, `hardening-playbook` supply chain section |
| `buddy.*` | every skill's reporting step |
| `output.reports_dir` | every skill that writes a report |
| `skill_level` | explanation depth across all skills |

## Exposure values

| Value | Means | Effect on severity |
| --- | --- | --- |
| `internal` | reachable only from a trusted network or VPN | caps most findings one level lower |
| `authenticated` | public endpoint, credentials required | authz bugs rate higher, authn bugs are critical |
| `public` | anonymous internet access to some surface | no severity capping; unauthenticated paths are critical by default |

When exposure is mixed, record the highest and note the split in `security.trust_boundaries`.
