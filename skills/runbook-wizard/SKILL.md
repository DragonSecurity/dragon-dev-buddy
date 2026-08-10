---
name: runbook-wizard
description: Generates an interactive bash script that walks a human through the steps only a human can take — provisioning infrastructure, creating credentials and CI secrets, clicking through an unfamiliar third-party dashboard, a one-off migration or cutover. Use when someone says "walk me through setting this up", "turn this manual procedure into a script", "we do this by hand every time", "generate a setup wizard", "write the runbook for this", "I have to click through the console for this", "onboarding script for new devs", "script the credential rotation", or when the same manual procedure has now been re-explained to an agent twice. If the agent can do the step itself, it should — this is only for where a human is genuinely in the loop.
---

# Runbook Wizard

Some steps cannot be delegated. A console with no API. A credential the agent must never hold. An approval that has to come from a named human. A cutover where the judgement call in the middle is the whole point. Those steps get done by hand, badly, from a stale wiki page, or by explaining the same procedure to an agent for the fourth time.

This skill turns one of those procedures into an interactive bash script. It opens each URL, says exactly what to click, prompts for each value, validates it against the shape the provider actually issues, and writes it where it belongs. It records what it has completed so a run that stops half-way resumes rather than restarts. The procedure stops being tribal knowledge and becomes a file.

**The rule that keeps it honest: if the agent could do the step itself, it should.** A wizard that opens a browser so a human can click a button with a documented API is a worse version of a shell command, and it teaches people to hand-drive things that should have been automated. Split the procedure in two, do the agent half now, and generate a wizard only for what genuinely needs the human.

This pack ships a wizard skill because wizards handle credentials. A procedure that walks a human to a dashboard is a procedure whose whole point is usually a token, and every generated line decides whether that token ends up in a process table, a shell history, a screen recording, a world-readable file, or a commit. The template this skill generates from settles all of those before a stage is written.

Skills in this pack that hand off here: `change-window` for the console-side half of a device cutover that cannot be automated — the out-of-band login, the physical confirmation, the vendor GUI step with no API. `incident-response` for rotating a leaked credential through a provider's dashboard under time pressure, where the order is the safety property and a half-done rotation is an outage. `buddy-setup` for a repository whose onboarding needs accounts and keys a human has to create. `ship-it` for a release whose last mile is manual — a store submission, a DNS change made in a registrar's UI, an approval gate.

## First-run check

Read `.dragon-buddy/config.json`. This skill needs little of it: `project.name` for the wizard's banner, `practice.ci` to know whether `gh secret set` is the right destination for a value or whether CI lives somewhere else entirely, and `security.exposure` plus `security.data_sensitivity` to set how hard the confirmations bite. Missing config is not fatal — say so, offer `buddy-setup`, and carry on from what the repository tells you.

What this skill does need is the git state, and it needs it before writing anything. Confirm the working tree is a git repository, and find out which paths are ignored. A wizard that captures a credential is only safe writing it where git will not commit it, and the generated script refuses to write anywhere else.

## Inputs

Read the repository before asking anything. `.env.example`, `.env.*`, the README, `docker-compose*`, framework config, `.github/workflows/*` — every `secrets.*` and `vars.*` reference in CI is a value the wizard has to produce. For a migration or cutover, read the current state, the target state, and whatever describes the path between them.

Then ask only for what the repository cannot tell you:
- **Which steps genuinely need the human**, and why the agent cannot do them. This is the first question and it decides whether there is a wizard at all.
- **Where each value comes from** — which page, which panel, which button. If you do not know the current UI, say so rather than inventing it.
- **What is irreversible**, and what order the steps have to happen in for the irreversible one to be safe.
- **Who runs it and where.** A laptop, a bastion, a screen shared with four other people on a bridge call. That answer changes how much the script is willing to print.

## Workflow

1. **Rule out the agent, step by step.** Walk the procedure and mark each step: the agent can do this, or a human must. Do the agent's half now — write the config file, run the CLI, open the PR — and say what you did. What is left is the wizard. If nothing is left, say so and do not generate a script; a wizard with no human step is a shell script with extra ceremony.

2. **Scope the stages and confirm them.** Produce the ordered list of stages, and for each one the values it captures, where the human gets them, and where they land. Show it to the user before writing any bash. They will reorder it, drop a stage that no longer exists, and name a step you could not have known about — all of which is cheaper now than in a generated script.

3. **Map each stage's journey.** Write the precise path: which URL, what to click, where the value is displayed, which variable it fills. "Dashboard → Manage Account → API Tokens → Create Token → Edit zone DNS template → copy". A stage a stranger cannot follow is not done. Where you do not know the current UI or the exact command, check the documentation or ask — a wizard that sends a human to a page that does not exist is worse than no wizard, because they will trust it.

4. **Classify every value.** Secret or public. Destination: a local env file, a CI secret, a CI variable, both, or nowhere — some stages capture nothing and are pure actions. A credential goes to CI as a secret and never as a variable, because a variable is readable in the UI and printable in a log. And a value only gets a local copy if something on that machine actually reads it.

5. **Author from the template.** Copy `${CLAUDE_PLUGIN_ROOT}/scripts/wizard-template.sh` to the target path and write one stage per step below the `STAGES` marker. Never hand-edit the library above it — it is identical in every wizard so that a human who has run one knows how the next one behaves, including how to stop it. Use the helpers: they are what enforce hidden entry, stdin-not-argv for secrets, mode 600 on every file written, the gitignore refusal, and plan mode. `references/wizard-template.md` has the full library with the reasoning for each rule.

6. **Sequence for the resume, not for the happy path.** Give every stage a stable id and record it the moment it completes. Order the stages so the irreversible step is as late as it can be and everything it depends on is already recorded as done — in a credential rotation that means issue, install, verify, and only then revoke. Mark any stage that is not idempotent with `irreversible` so the human is told before they act, not after. A cutover that cannot be resumed leaves the system in a state neither the old runbook nor the new one describes.

7. **Verify without running it.** `bash -n`, then `shellcheck` if it is available, then run the wizard in plan mode — safe by construction, and the only end-to-end exercise available. Never run `--apply` yourself: it blocks on human input and performs real outward-facing actions against a live account. Then check the two halves of the file separately — `diff` the library above the `STAGES` marker against the shipped template to prove nothing was hand-edited, and grep only the stages you wrote for what must not be in them: `--body` carrying a secret, `set -x`, any git write, any echo of a captured credential. Finally trace statically that every value from step 2 is captured once, lands where step 2 said, and that every CI secret name matches a `secrets.*` reference in a workflow.

8. **Hand it over.** `chmod +x`, then tell them the two commands: run it with no flags to see the plan, re-run with `--apply` to do it. Say which stages are irreversible and where the state file lives. If the procedure repeats, commit the script and link it from the README so the next person runs it instead of asking an agent; if it was for one cutover, say that it is disposable and where it is.

## Output format

```markdown
## Wizard: [procedure]

**Done by the agent instead:** [the steps that did not need a human, and what was done about each]
**Stages:** [ordered — name, what the human does, values captured, where each lands]
**Secrets handled:** [each — hidden entry, destination, and how it gets there]
**Irreversible:** [which stages, and why they are sequenced where they are]
**Resume:** [state file path, stage ids, how to redo one]
**Verified:** [bash -n, shellcheck, plan run, anti-pattern grep — stated as run]
**Run it:** [the two commands, and what each does]
**Lifetime:** [committed and linked, or disposable and where]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Generated a <n>-stage wizard for <procedure>, handling <m> credentials; <k> steps were automated instead."`
- `kind`: `tooling`
- `skills_used`: `["dragon-dev-buddy:runbook-wizard"]`

Relay the reaction verbatim.

## File output

The generated wizard goes to `scripts/<slug>-wizard.sh` in the target repository when the procedure repeats, or to a scratch path when it is for one cutover. It is a script, not a document, and it holds no values — only the instructions for obtaining them.

This skill never runs the wizard's `--apply` mode and never captures a credential itself. The generated script never commits: no `git add`, no `git commit`, no `git push`, and it refuses to write a credential to any path that is tracked or not gitignored. If a destination file is missing an ignore rule, add the rule and say so — that is the same guarantee `secrets-and-config-audit` checks for after the fact, enforced before the value exists.

## Reference library

- `references/wizard-template.md` — the annotated wizard library, and the patterns it encodes: prompt-and-validate loops, opening a URL portably including under WSL, choosing between a local env file and a CI secret, resumability via a state file, plan mode, bash 3.2 portability, and the secret-handling rules with the leak each one prevents. Load it before authoring stages. The copy to actually use is `${CLAUDE_PLUGIN_ROOT}/scripts/wizard-template.sh`.

## Worked example

`examples/runbook-wizard-run.md` is the quality target: a leaked deploy token rotated through a provider dashboard, including the steps the agent refused to put in the wizard because it could do them itself, the plan run, and a real run that stops half-way and resumes.

## Quality bar

A good run satisfies all of these:

- Every step an agent could take was taken by the agent, named, and left out of the wizard. Only genuinely human steps survived into stages.
- The stage list was confirmed with the user before any bash was written.
- Every stage traces to instructions a stranger could follow, and nothing about a third-party UI was invented — unknowns were checked or asked about.
- Every captured value is validated at the prompt against the shape the provider issues, with a hint that says what was expected.
- No secret is echoed, and none is written to a file that is not mode 600 and gitignored. The gitignore check ran before the human was asked for anything. Secrets reached commands over stdin and not argv — and where one genuinely could not, that single line was named in the handover with its reason, not left to be found.
- The generated script contains no git write of any kind.
- Every stage is idempotent or says on screen that it is not, and the irreversible ones are sequenced last.
- The wizard is resumable: stable stage ids, state recorded as each stage lands, and a documented way to redo one.
- Plan mode is the default and it was run — `bash -n` clean, plan walked end to end, the library diffed clean against the shipped template, the anti-pattern grep over the stages clean, `--apply` never run by the agent.
- The user was told the two commands, which stages cannot be undone, and whether the script is committed or disposable.
