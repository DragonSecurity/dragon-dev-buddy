# Incident response runbook

## Containment by compromise type

### Leaked credential (API key, token, password)

1. Capture the credential's usage log first — provider dashboard, audit log, access log — over the whole possible exposure window. This is evidence and it is gone once you rotate.
2. Rotate: issue new, deploy, revoke old. For high-privilege credentials, revoke immediately even at the cost of an outage.
3. Assume the leaked credential was used to obtain others. Rotate the blast radius, not just the one key.
4. Determine how it leaked — repo, log, phishing, third party — because that vector is the actual incident.

### Compromised account

1. Disable the account (do not delete — you need its activity trail). Revoke all active sessions.
2. Capture its recent activity: logins, source IPs, actions taken, data accessed, changes made.
3. Reset credentials and require re-enrollment of MFA.
4. Check what the account could reach and whether it created persistence (new tokens, added collaborators, forwarding rules, OAuth grants).
5. Assume lateral movement. What trusts this account?

### Host / server intrusion

1. Isolate from the network — security group to deny-all, or unplug — but **do not power off**. Powering off loses volatile memory (running processes, injected code, encryption keys) and can trigger destructive logic.
2. If you have the capability, capture volatile state first: memory image, process list, network connections, logged-in users.
3. Do not clean the host and return it to service. Rebuild from known-good; you cannot prove a cleaned host is clean.
4. Hunt persistence: new accounts, cron/systemd/scheduled tasks, modified startup and rc files, SSH authorized_keys, new services, altered binaries.
5. Determine entry vector before recovery, or you rebuild into the same hole.

### Supply-chain compromise (malicious dependency, poisoned build)

1. Identify the affected package and versions. Pin away from them across every project.
2. Assume anything the build touched is suspect: build-time secrets, signing keys, published artifacts.
3. If you published a compromised artifact, this is now your users' incident too — notify, and pull the artifact.
4. Rotate every secret the compromised build had access to.

### Ransomware / destructive attack

1. Isolate to stop spread — network segmentation, disable file shares — immediately.
2. Do not power off encrypting machines if avoidable; some encryption keys live only in memory and are recoverable.
3. Identify patient zero and the spread path.
4. Recover from offline, known-good backups — which is why backups must be offline and tested. Assume online backups are also encrypted.
5. Involve legal and, where relevant, law enforcement before any ransom consideration.

## Evidence preservation

- **Capture before you change.** Every containment action potentially destroys evidence. Log usage, image memory, export the audit trail *first*.
- **Isolate, do not destroy.** Disable not delete, quarantine not wipe, network-off not power-off. You can always destroy later; you cannot un-destroy.
- **Preserve integrity.** Copy evidence to write-once or hashed storage. Record a hash of each artifact when you capture it, so its integrity is demonstrable later.
- **Chain of custody.** For anything that might see legal use: who collected it, when, from where, and every hand it passed through. If law enforcement or litigation is even possible, treat evidence formally from the first minute — you cannot retroactively establish custody.
- **Timestamps in a fixed zone.** Pick one timezone (UTC is safest) and stamp everything in it. Mixed local times across responders make a timeline unreconstructable.

## Notification obligations (verify current specifics; these are orientation, not legal advice)

| Regime | Rough trigger | Rough clock |
| --- | --- | --- |
| GDPR | Personal data breach with risk to individuals | 72 hours to the supervisory authority |
| HIPAA | Breach of protected health information | Notification duties; 60 days is the outer bound for individuals |
| PCI DSS | Cardholder data compromise | Notify the card brands/acquirer promptly; contractually fast |
| US state laws | Personal information of residents | Varies by state; several are "without unreasonable delay" |
| SOC 2 / contracts | Per your customer commitments | Whatever your DPAs and contracts say — often 24-72h |

Start the clock at detection, in the live log, the moment the data class is confirmed. The investigation does not pause the clock. When in doubt, involve legal early — notification decisions are legal decisions, not only technical ones.

## Blameless post-incident review

Blameless means the analysis targets the system and the conditions, not the person who clicked or committed. People act reasonably given what they knew and the pressures they were under; if a single mistake could cause this, the system that allowed one mistake to cause it is the finding.

Structure:

1. **Summary** — what happened, impact, duration, in three sentences.
2. **Timeline** — detection to resolution, the consolidated live log.
3. **Root cause** — how they got in. Go past the proximate cause: not "a key leaked" but "keys were committable because `.env.*` was not gitignored and there was no pre-commit scan."
4. **Detection gap** — why it was not caught sooner, and what would have caught it.
5. **Response assessment** — what went well, what slowed containment, honestly.
6. **Preventive actions** — specific, owned, dated. Each routed to the skill that implements it: `hardening-playbook` for the systemic gap, `secure-code-review` for the code path, `debug-and-fix` for the specific defect. "Be more careful" is not a preventive action; a pre-commit hook is.

The output of a good review is a short list of changes that make this class of incident harder next time — and each one is something a skill in this pack can actually go and do.
