---
name: vuln-triage
description: Turns a vulnerability report, CVE, scanner alert or bug bounty submission into a verdict: is it real, is it reachable here, how bad is it, and what is the fix. Use when someone says "is this CVE relevant to us", "someone reported a vulnerability", "triage this finding", "we got a bug bounty report", "how bad is this", "is this exploitable", or "the scanner found X, do we care". Produces a decision with evidence, not a maybe.
---

# Vulnerability Triage

Most reported vulnerabilities are not exploitable in the system they were reported against. Some are worse than reported. Triage is the work of finding out which, quickly enough that the answer is still useful, and with enough evidence that the answer holds up when someone disagrees.

The output is a verdict and a fix, not a hedge. "Possibly exploitable, recommend further investigation" is what triage exists to replace.

## First-run check

Read `.dragon-buddy/config.json`. If missing, run `buddy-setup` first. Pull: `project.stack`, `project.runtime`, `security.exposure`, `security.data_sensitivity`, `security.compliance`, `engagement.authorized_scope`, `engagement.contact`, `practice.test_command`, `output.reports_dir`.

## Authorization gate

This skill can produce proof-of-concept exploit code. Before it does, one of these must hold:

- The target is code in this repository, which the user controls, and the PoC runs locally against their own build; **or**
- `engagement.authorized_scope` is populated and the target falls inside it; **or**
- The user states this is a CTF, a lab, or a deliberately vulnerable training application.

If none holds, still do the full triage — the verdict, the reachability analysis, the severity and the fix. Just describe the exploitation path in prose rather than writing runnable code against a system nobody has confirmed you may attack. Say which you are doing and why, in one line, and move on. Do not turn it into a negotiation.

Regardless of authorization: no code that targets systems in bulk, no worm or self-propagating logic, and nothing whose purpose is to evade detection rather than to demonstrate the flaw.

## Inputs

Ask only for what is missing:
- **The report.** A CVE id, a scanner finding, a bug bounty writeup, or a description. Take it verbatim; reporters' own words carry detail that summaries lose.
- **Where the reporter says it is.** An endpoint, a file, a package, a parameter.
- **Any PoC they supplied.** Read it; do not run it against anything live.
- **Whether a deadline applies.** A bounty SLA or a disclosure date changes the depth you can afford.

## Workflow

1. **Restate the claim precisely.** In one sentence: who can do what, to get what. Reports are frequently vaguer than the underlying bug, and occasionally describe a different bug from the one that exists. This sentence is what you are going to verify or refute.

2. **Establish whether the code exists here.** Find the actual code path. For a CVE, find the dependency and version and confirm the affected range genuinely covers it. For a reported endpoint, find the handler. If the code does not exist or the version is outside the range, you are done: **not applicable**, with the evidence.

3. **Establish reachability.** Trace from an entry point an attacker can reach to the vulnerable code. Name every precondition on the way: authentication, a role, a feature flag, a specific configuration, a race window. A vulnerability behind a flag that is off in production is a different finding from one on the login path.

4. **Attempt to refute it.** Actively look for the reason it does not work: a validation layer upstream, a WAF rule, a type coercion that neutralizes the payload, a framework default that already mitigates it. Report what you tried. A finding that survives a genuine refutation attempt is worth far more than one that was never challenged, and this step is where most false positives die.

5. **Verify.** Within the authorization gate: reproduce it against a local build, and write the smallest input that demonstrates the flaw. Record exactly what you ran and what came back. If you could not verify, say **why** — no local environment, needs production data, needs a second account — and state the confidence level honestly.

6. **Rate it for this system.** Use the rubric in `references/triage-rubric.md`. The reporter's severity and the advisory's CVSS are inputs, not conclusions. Say plainly when you are rating it above or below the source, and why.

7. **Decide.** One of:
   - **Confirmed** — real, reachable, fix it. Give the timeline.
   - **Confirmed, lower severity** — real, but the outcome is weaker than claimed. Say why.
   - **Not exploitable** — the code exists but something blocks it. Name the blocker, and name what would remove it.
   - **Not applicable** — the code, version or configuration is not present.
   - **Duplicate** — of a known finding. Link it.
   - **Cannot determine** — with the specific missing information and how to get it.

8. **Write the fix.** Minimal correct change first, then the structural fix if the minimal one leaves the class open. Name both and say which to ship now. Hand it to `security-test-writer` so the regression is locked down.

9. **If this came from an external reporter,** draft the reply: the verdict, the reasoning, and the timeline. Reporters who receive a real technical answer come back with more findings; reporters who receive a template do not.

## Output format

```markdown
# Triage: [identifier or short title]
[date] · verdict **[CONFIRMED | LOWER | NOT EXPLOITABLE | N/A | DUPLICATE | UNDETERMINED]** · severity **[level]**

**Claim:** [who can do what, to get what]
**Code path:** `file:line` — [present? affected version?]
**Reachability:** [entry point → sink, with every precondition named]
**Refutation attempted:** [what you tried to make it not work, and the result]
**Verification:** [what you ran, what came back — or why not, and confidence]
**Severity here:** [level] vs [reported/CVSS level] — [why they differ]

## Fix
**Now:** [minimal change, with file]
**Then:** [structural change, if the minimal one leaves the class open]

## Reply to reporter
[drafted response, if external]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Triaged <identifier>: <verdict>, <severity> — <one clause of reasoning>."`
- `kind`: `bugfix`
- `skills_used`: `["dragon-dev-buddy:vuln-triage"]`

Relay the reaction verbatim.

## File output

One markdown file in `output.reports_dir` as `YYYY-MM-DD-triage-<identifier>.md`. PoC code, where authorized, goes in a clearly marked directory outside the build — never in `src/`, never wired into the test suite by default. Hand the regression test to `security-test-writer` instead.

## Reference library

Load these for depth when the task calls for it:
- `references/triage-rubric.md`: the severity rubric with reporter-versus-actual calibration, the reachability precondition checklist, common false-positive patterns by class, and reporter reply templates for each verdict.

## Worked example

See `examples/vuln-triage-run.md` for a triage that downgrades a reported Critical. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- The claim is restated in one precise sentence before any analysis.
- Reachability names every precondition. "It's reachable" without the path is not analysis.
- A genuine refutation was attempted and reported, including when it failed to refute.
- The verdict is one of the seven. No "possibly exploitable, needs investigation."
- Severity disagreements with the reporter or the CVSS score are stated explicitly with reasoning.
- Unverified findings say why, and state confidence honestly rather than implying certainty.
- PoC code exists only within the authorization gate, and the gate decision is stated in one line.
- The fix separates what ships now from what closes the class.
