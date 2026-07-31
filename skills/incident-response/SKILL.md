---
name: incident-response
description: Drives a live security incident through triage, containment, eradication, recovery and a written timeline. Use when someone says "we've been breached", "there's an active attack", "I think we're compromised", "a key leaked", "someone's in our system", "unusual activity", "we're under attack", or is mid-incident and needs to move fast without making it worse. Containment before investigation; the timeline gets written as it happens, not reconstructed after.
---

# Incident Response

An incident is not a bug. The clock is running, the adversary may be active, and the wrong move — tipping them off, destroying the evidence, taking the wrong thing down — makes it worse. This skill imposes the order that panic erodes: contain the damage, then understand it, then remove the adversary, then recover, and write down what happened while it is still true rather than reconstructing it from memory a week later.

The order is deliberate and it is not the order instinct suggests. Instinct says "find out what happened." Containment comes first, because every minute spent investigating an uncontained incident is a minute the damage grows.

## First-run check

Read `.dragon-buddy/config.json`. If missing, work from what the user tells you — an incident does not wait for setup. Pull if present: `security.exposure`, `security.data_sensitivity`, `security.compliance`, `engagement.contact`, `project.runtime`, `output.reports_dir`. `compliance` matters early: it may impose a notification clock measured in hours.

## The stance

- **Contain before you investigate.** A live attacker with active access does more damage while you read logs. Stop the bleeding first.
- **Preserve evidence while containing.** Do not `rm` the webshell, wipe the box, or rotate away the compromised key before you have captured what it tells you. Isolate rather than destroy: pull the host from the network, do not terminate it; disable the account, do not delete it. `references/ir-runbook.md` covers evidence handling.
- **Write it down as you go.** Every action, with a timestamp and who did it. The timeline is a live document during the incident, not a report written afterward. Memory is not admissible and is not accurate.
- **Assume you know less than you think.** The first-observed symptom is rarely the origin. One compromised key usually means "how did they get the key," not "one key leaked."
- **Do not tip off an active adversary** until you can contain them fully. Alerting them to your presence before you can lock them out invites them to burn what they have — destroy data, deploy ransomware, dig in deeper.

## Workflow

1. **Triage: is this real, and how bad.** Confirm it is an actual incident, not a misread log or a failed job. Classify: what is affected, what data class is at risk, is the attacker active now or is this a post-hoc discovery, is it spreading. Set a severity, and if `security.compliance` implies a notification deadline, start that clock explicitly — it may be shorter than the investigation.

2. **Contain.** Stop the damage from growing, without destroying what you will need to understand it:
   - Isolate affected systems from the network (do not power off — you lose memory state and may trip a dead-man's switch).
   - Disable compromised accounts and revoke sessions (do not delete the account).
   - Rotate exposed credentials — but capture their usage logs first.
   - Block the attacker's access path if you can identify it without alerting them prematurely.
   State what you are containing and what you are deliberately leaving running to preserve evidence or service.

3. **Assess scope under the assumption it is wider than it looks.** What did the attacker reach, and what could they have reached from there? Follow the access: a compromised key's permissions, an account's lateral reach, a service's blast radius. Look for persistence — added accounts, scheduled tasks, modified startup, new keys, backdoors — because a partial eradication that misses persistence lets them walk back in. Distinguish confirmed from suspected; label everything.

4. **Eradicate.** Remove the attacker's access and everything they left: close the entry vector, remove persistence, rotate every credential in the blast radius (not just the one that leaked), rebuild rather than clean where you cannot be sure. Verify eradication before recovery — bringing a still-compromised system back is starting the incident over.

5. **Recover.** Restore service deliberately: from known-good state, with the vulnerability closed, watching for the attacker's return. Monitor the recovered systems more closely than usual; re-compromise attempts are common and tell you eradication was incomplete. Restore in order of business criticality, verifying integrity as you go.

6. **Notify, per obligation and honestly.** If `security.compliance` or the data class requires it, meet the notification duties — regulators, affected users, partners — within the clock started in step 1. Tell `engagement.contact`. Draft the notifications factually: what happened, what data, what you have done, what the recipient should do. Do not minimize; understated breach notifications age badly.

7. **Write the timeline and the post-incident review.** Consolidate the live notes into a timeline: detection, each action with its timestamp, decision points and who made them. Then, separately and blamelessly, the review: how they got in, why it was not caught sooner, what would have contained it faster, and the specific changes that prevent recurrence. Route the preventive changes to `hardening-playbook` and the root-cause fix to `debug-and-fix` or `secure-code-review`.

## Output format

During the incident, a live running log:

```markdown
# Incident [id] — [short description]  ·  SEVERITY [level]  ·  STATUS [active/contained/eradicated/recovered]
Notification clock: [regime — deadline, or none]

## Timeline (append-only, timestamped)
[HH:MM TZ] [actor] [action] [result]

## Confirmed / Suspected
Confirmed: [...]   Suspected: [...]   Ruled out: [...]

## Blast radius
Reached: [...]   Reachable-from-there: [...]
```

Then the post-incident review as a separate document.

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Call `buddy_observe` **after** the incident is contained, never during — you do not stop mid-containment to update a companion:
- `summary`: `"Handled incident <id>: <what>, contained in <time>, root cause <one clause>."`
- `kind`: `bugfix`
- `skills_used`: `["dragon-dev-buddy:incident-response"]`

Relay the reaction verbatim.

## File output

The live log and the post-incident review to `output.reports_dir`, named `YYYY-MM-DD-incident-<id>.md` and `-postmortem.md`. Handle evidence per `references/ir-runbook.md` — captured separately, integrity-preserved, not pasted into a shared report. **Keep credential values and personal data out of the shared report**, exactly as in the other audit skills.

## Reference library

Load these for depth when the task calls for it:
- `references/ir-runbook.md`: containment actions by compromise type (leaked key, compromised account, host intrusion, supply chain, ransomware), evidence preservation and chain-of-custody basics, notification obligations by regime, and the blameless post-incident review template.

## Worked example

See `examples/incident-response-run.md` for a leaked-credential incident from alert to post-incident review. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- Containment came before investigation, and the log shows it.
- Evidence was preserved while containing — nothing was destroyed that would be needed to understand the incident. Isolation over deletion.
- The timeline was kept live with timestamps, not reconstructed at the end.
- Scope was assessed as wider than the initial symptom; persistence was actively hunted.
- Eradication rotated the whole blast radius, not just the credential that surfaced, and was verified before recovery.
- Notification obligations were identified early with the clock started, and any notification drafted factually without minimizing.
- The post-incident review is blameless and produces specific preventive changes, routed to the skills that implement them.
- No credential values or personal data in the shared report.
