---
name: secrets-and-config-audit
description: Hunts leaked credentials in code and git history, then audits configuration and infrastructure-as-code for insecure defaults. Use when someone says "check for secrets", "did I commit a key", "scan for credentials", "audit my config", "is my terraform secure", "check my Dockerfile", "secret scanning", "we rotated but I want to be sure", or before making a repository public. Rotation comes before reporting: a live credential is an incident, not a finding.
---

# Secrets and Config Audit

Two jobs that belong together because they fail together. Credentials leak into repositories, and the configuration around them is usually what decides whether a leak becomes a breach.

The order matters and is not negotiable: **find, assess liveness, rotate, then report.** A credential that is live in a real system is not finding number seven in a document. It is an incident, and every hour it stays valid is an hour of exposure that this skill caused by knowing and continuing.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `project.runtime`, `project.deploy_target`, `security.exposure`, `security.data_sensitivity`, `security.compliance`, `practice.secret_scanner`, `practice.ci`, `engagement.contact`, `output.reports_dir`.

## Inputs

Ask only for what is missing:
- **Scope.** Working tree only, or full git history? History is where the real findings are, and it is slower.
- **Is the repository public, or has it ever been?** Changes everything about urgency.
- **Who can rotate credentials**, and is that person reachable right now?

## Workflow

1. **Scan the working tree.** Use `practice.secret_scanner` if set, otherwise `gitleaks detect --no-git` or `trufflehog filesystem`. Also grep manually for the patterns in `references/secret-patterns.md` — scanners miss custom formats, and every company has one.

2. **Scan git history.** `gitleaks detect` over the full history, or `trufflehog git file://.`. A secret removed in a later commit is still in the history and still valid. Also check: branches that were never merged, tags, and stashes if you have local access.

   For every hit, record the commit, the date, the author, and whether the file is still present.

3. **Assess liveness without using the credential.** This is the line. You may:
   - Read the prefix and format to identify the provider and scope (`rk_live_` vs `rk_test_`, `AKIA` vs `ASIA`, `ghp_` vs `github_pat_`)
   - Check whether the file is still in the working tree
   - Check the commit date to bound the exposure window
   - Ask the user to check the provider's own dashboard or audit log

   You may **not** authenticate with a credential you found to see whether it works. That is using someone's key, and "I was only testing" is not a distinction that survives contact with an audit log. Ask the user to test it, or to rotate on the assumption that it is live.

4. **Stop and escalate if anything is live.** Do not continue the audit. Report immediately with:
   - What it is, where it is, and how long it has been there
   - The rotation steps, in order, for that specific provider
   - What to check in the provider's audit log, and what a compromise would look like
   - Whether `engagement.contact` needs to be told now

   If the repository is or ever was public, assume the credential is compromised regardless of what the logs show. Automated scrapers index public pushes within minutes.

5. **Audit configuration.** Once secrets are handled, work `references/config-checklist.md` across whatever applies: Dockerfile and compose, Kubernetes manifests, Terraform and other IaC, CI workflow files, web server and framework config, cloud storage policies, and database settings. Report insecure defaults with the specific line and the specific consequence.

6. **Audit how secrets are supposed to be handled.** The absence of a leak today is not a control. Check: are secrets injected at runtime or baked into images, is there a secret manager, are they in CI as encrypted secrets or plain variables, is `.env` in `.gitignore`, do secrets appear in build logs, and is there a rotation policy anyone follows.

7. **Report.** Findings in two sections: credentials (with rotation status for each) and configuration (with severity). Then the prevention section: pre-commit hook, CI scanning, and the one change most likely to stop the next one.

## Output format

```markdown
# Secrets and config audit: [project]
[date] · [n] credentials found ([n] live) · [n] config findings

## ⚠ Immediate action
[only if live credentials were found — rotation steps, in order]

## Credentials
### S1 — [provider] [type]   **[LIVE | ROTATED | TEST | UNKNOWN]**
Found: `path:line` · commit `sha` · [date] · [author] · [still in tree? y/n]
Exposure window: [dates] · Repository visibility during window: [public/private]
Rotation: [steps]   Audit log check: [what to look for]

## Configuration
### C1 — [title]   **[SEVERITY]**
`path:line` — [what is set] → [what it should be]
**Consequence:** [what an attacker gets from this specific setting]

## Secret handling
[how secrets flow today, and where that breaks down]

## Prevention
[pre-commit, CI, and the single highest-value change]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Audited secrets and config: <n> credentials (<n> live), <n> config findings, worst is <one clause>."`
- `kind`: `config`
- `skills_used`: `["dragon-dev-buddy:secrets-and-config-audit"]`

Report **after** rotation is underway, not while a live key is unrotated. Relay the reaction verbatim.

## File output

One report in `output.reports_dir` as `YYYY-MM-DD-secrets-config-audit.md`. **Never write the credential value into the report.** Record the first four characters, the provider, and the location. A report full of live keys is a second leak, and reports get shared more widely than repositories.

## Reference library

Load these for depth when the task calls for it:
- `references/secret-patterns.md`: provider prefixes and formats, custom-secret grep patterns, liveness assessment by provider, and per-provider rotation steps.
- `references/config-checklist.md`: insecure defaults by platform — Docker, Kubernetes, Terraform/AWS, CI, web frameworks, databases — each with the consequence.

## Worked example

See `examples/secrets-and-config-audit-run.md` for a run that finds a live key. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- History was scanned, not just the working tree. A working-tree-only scan is stated as such.
- No credential was ever used to test liveness. The assessment method is stated explicitly.
- A live credential stopped the audit and produced rotation steps before anything else was reported.
- No credential value appears in the report. Prefix and location only.
- Every config finding names the file, the line, the current value, the target value, and the consequence.
- The prevention section names one highest-value change, not a list of nine.
- Public repository history is treated as compromised regardless of audit log evidence.
