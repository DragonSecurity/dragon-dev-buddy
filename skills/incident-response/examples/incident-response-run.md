# Worked example: a leaked credential, from alert to review

---

**User:** GitGuardian just alerted that an AWS key of ours is in a public gist. I don't know how it got there. What do I do

**Assistant:** This is an active incident — a live AWS key, public. Containment first, then we understand it. I'm starting the log now; every action gets a timestamp.

**First: is it live, and what can it do?** Don't test the key. Go to IAM and find it by the `AKIA...` prefix GitGuardian showed you. Tell me: does the key exist, is it enabled, and what policies are attached? That tells us the blast radius before we move.

Meanwhile — `security.data_sensitivity` is `credentials` and you're under SOC 2 and mid-PCI. **I'm starting a notification clock as of now.** If this key could reach cardholder data, PCI notification obligations are fast. We'll confirm reachability in a moment, but the clock starts at detection, not at confirmation.

```
# Incident 2026-03-20-01 — AWS key in public gist · SEVERITY high · STATUS active
Notification clock: SOC2 (contractual, ~24-72h) + PCI (if CHD reachable) — started 14:02 UTC

## Timeline
14:02 UTC  alanis   GitGuardian alert: AKIA... in public gist. Incident opened.
14:03 UTC  alanis   Locating key in IAM before any rotation, to capture scope + usage.
```

---

**User:** it exists, enabled, and it has AmazonS3FullAccess attached. that's way more than it should have.

**Assistant:** S3FullAccess is the whole blast radius — every bucket in the account. Here's the order, and the sequencing matters:

**1. Capture usage BEFORE rotating** (this evidence disappears when you rotate):
- CloudTrail, filtered to that `AccessKeyId`, from as far back as it could have been exposed to now. Export it. We need: what did this key actually do, and from what IPs.

**2. Then contain** — create a replacement key for the legitimate workload, deploy it, and *deactivate* (not delete) the exposed one. Deactivate keeps it in CloudTrail as a known entity; delete muddies later correlation.

**3. Ask the gist host** (GitHub) to remove the gist, but assume it's already scraped. Public means public; the key is compromised regardless of whether the gist comes down.

Do 1 and 2 now. Tell me what CloudTrail shows for that key.

```
14:05 UTC  alanis   Key confirmed: enabled, AmazonS3FullAccess. Blast radius = all S3.
14:06 UTC  alanis   Exporting CloudTrail for AccessKeyId before rotation.
14:11 UTC  alanis   New key deployed to the ETL job; exposed key DEACTIVATED (not deleted).
```

---

**User:** cloudtrail shows the key only ever used from our own NAT IP, ETL job calls only, nothing from outside. gist was posted 40 minutes ago.

**Assistant:** That's the best version of this. Every call from your NAT IP, ETL patterns only, no external source in the window — no evidence of malicious use in the 40 minutes it was exposed. Recording as **contained**, and downgrading the likely-impact, but not the severity: a public S3-full key is high severity regardless of whether we caught it before anyone used it.

```
14:14 UTC  alanis   CloudTrail: only NAT-IP ETL calls in exposure window. No external use observed.
14:15 UTC  system   STATUS active → contained.
```

**Now the actual incident, which is not "a key leaked."** The real questions:

1. **How did an S3-full key end up in a public gist?** A key doesn't post itself. Someone pasted it — debugging, a support thread, an AI prompt, a misdirected snippet. That person and that workflow is the thing to fix.
2. **Why did the ETL job's key have `AmazonS3FullAccess`** when it touches — what, one bucket? That over-permissioning turned a paste into an all-buckets exposure. Least privilege would have made this a shrug.

On notification: CloudTrail shows no access to cardholder data by this key (ETL calls only, to non-CHD buckets — confirm which buckets). If that holds, PCI's breach trigger isn't met; you still owe your SOC 2 customers a factual notice per your DPAs. I'll draft it. Get legal to confirm the PCI read — that's their call, not ours.

---

*(later — the review)*

## Post-incident review (blameless)

**Summary.** An AWS access key with `AmazonS3FullAccess` was posted to a public gist and exposed for ~40 minutes before automated detection. No evidence of malicious use. Contained in 13 minutes from alert. No confirmed data access by an unauthorized party.

**Root cause.** Not "a key leaked." Two systemic conditions:
- The ETL job's key carried `AmazonS3FullAccess` when its actual need is read on two named buckets. Over-permissioning turned a single paste into a whole-account exposure.
- There was no barrier between a developer's hands and a live long-lived key — the key was copyable and pasteable because it was a static credential a human had in a config file.

**Detection gap.** Detection was external (GitGuardian), not ours, and took 40 minutes. We had no secret scanning on our own surfaces and no alert on a new IAM key being used from an unexpected context.

**What went well.** The responder captured CloudTrail before rotating, deactivated rather than deleted, and did not panic-delete the key. The 13-minute containment was good.

**Preventive actions:**
1. Scope the ETL key to read on the two buckets it uses. → `hardening-playbook`, this week. *(Removes the blast radius entirely.)*
2. Replace the static ETL key with an IAM role / instance profile — no long-lived key to paste. → `hardening-playbook`. *(Structural: there is no key to leak.)*
3. gitleaks pre-commit + CI, and enable AWS's own public-access and key-exposure detection. → `hardening-playbook`. *(Closes the 40-minute detection gap.)*
4. Confirm with legal that the no-CHD-access finding closes PCI notification. → owner: alanis, today.

The theme: least privilege (#1) and no-static-credential (#2) would each, independently, have made this a non-event. That's where the effort goes.

---

## What this run got right

- Contained before investigating, and refused to test the key.
- Started the **notification clock at detection**, before confirming reachability, because the clock doesn't wait for the investigation.
- Captured CloudTrail evidence **before** rotation, and chose deactivate-not-delete with a stated reason.
- Kept a live, timestamped, single-timezone log throughout.
- Reframed "a key leaked" into the two systemic causes, and hunted the real vector (who pasted it, and why the key was so over-powered).
- Made the notification call factual and deferred the legal determination to legal.
- Blameless review: targeted the over-permissioning and the static credential, not the person who pasted, and routed every preventive action to a skill that can do it.
