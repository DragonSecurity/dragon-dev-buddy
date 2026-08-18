---
name: buddy-setup
description: Onboarding skill for the Dragon Dev Buddy pack. Interviews you about the codebase, its exposure and your tooling, writes .dragon-buddy/config.json, and hatches your buddy companion. Use when someone says "set up dragon dev buddy", "run setup", "onboard this repo", "start using the security pack", "configure the buddy", "I'm new here", or when any other skill in this pack reports that config is missing. Run this first, once per repository.
---

# Dragon Dev Buddy Setup

This is the first skill you run in a repo. It works out what the project is, how exposed it is, what data it touches and what tooling already exists, then saves all of that to `.dragon-buddy/config.json`. Every other skill in this pack reads that file, so nothing else has to ask you what your stack is again.

It also introduces you to your buddy: a persistent companion served over MCP that gains XP as you work, keeps a streak, and learns which skills you reach for.

Setup is deliberately short. Most of what the other skills need can be read out of the repo, so this asks about the things that are not in the code: who can reach this system, what happens if it leaks, and what you are authorized to attack.

## First-run check

Look for `.dragon-buddy/config.json` in the repo root.

If it exists and `project.name` is filled in, summarize the current profile in five lines and ask: "Want to change anything, or are we good?" Stop there unless changes are requested. Do not re-run the interview over a working config.

If it is missing or `project.name` is empty, run the full workflow below.

## Inputs

Read before you ask. Ask only for what you genuinely cannot determine:

- Detectable from the repo: language, framework, package manager, test/lint/build commands, CI provider, deploy target, whether a lockfile exists, whether a secret scanner or SAST config is present.
- Must be asked: exposure level, data sensitivity, compliance obligations, authorized engagement scope, who to call on a critical finding.

If the user already described the project in this conversation, use their words rather than asking again.

## Workflow

1. **Survey the repo first.** Before asking anything, look at: `package.json` / `pyproject.toml` / `go.mod` / `Cargo.toml` / `pom.xml`, the CI config under `.github/workflows` or equivalent, `Dockerfile` and any IaC directories, and the README. Build a draft profile from what you find. This is the difference between a two-minute setup and a twenty-question interrogation.

   Watch for a **spec-driven codegen stack** while you survey — a Huma/OpenAPI backend with generated Go/TS SDKs or a generated Terraform provider (`docs/openapi.json`, `.openapi-generator/`, `openapi-ts.config.ts`, `sdk/`, `terraform-provider-*`). `references/repo-survey.md` has the full signature. If present, record it in `project.stack`; it changes which references the audit skills load.

2. **Confirm the draft, batch one.** Show what you inferred and ask the user to correct it, all in one message:
   - "Here's what I read off the repo: [name], [language/stack], tests via `[command]`, deploys to [target]. Right?"
   - "One sentence: what does this system actually do, and for whom?"
   - "Any other repos this one is entangled with? The one that consumes it, the one it ships or deploys alongside, the upstream it gets confused for." Record what the user names into `related_repos`, with a relative `path` when it is checked out beside this repo. Directory adjacency is not evidence — a projects folder holds dozens of siblings and proximity says nothing about entanglement.

3. **Ask batch two, the exposure questions.** These are not in the code and they drive every severity rating this pack will ever produce. Ask all three together:
   - Who can reach it? Internal only, authenticated users, or open to the public internet?
   - What is the worst data it holds or touches? Nothing sensitive, PII, financial, health, or credentials and keys for other systems?
   - Any compliance regime that applies? SOC 2, GDPR, HIPAA, PCI DSS, or none.

4. **Ask batch three, boundaries and tooling.** Together:
   - Where are the trust boundaries? Name the two or three places where data crosses from something you control to something you do not, or vice versa. If they are unsure, offer to derive these later in `threat-model` and move on.
   - Anything you already know is weak and have not gotten to yet?
   - Which security tooling is in play, if any: SCA, SAST, secret scanning.

5. **Ask the fleet questions, only if relevant.** If the survey found network or device configuration — Ansible network roles, an Oxidized or RANCID backup tree, a NetBox or Nautobot export, vendor config files, IaC for firewalls or routing — or the user mentions switches, routers, firewalls or a device fleet, ask these together and fill the `fleet` block:
   - Which vendors and platforms, and roughly how many devices in which classes?
   - Where does device config come from and get backed up to, and is there a golden config or template?
   - What out-of-band access exists if a change cuts the network path? This is the one that matters most — `change-window` cannot gate a self-severing change without it.
   - Is there a change window, and who approves a production network change?

   If none of that applies, leave `fleet.managed` false and the block empty. No other skill in the pack reads it.

6. **Ask about engagement scope, only if relevant.** If this is the user's own code and they are not doing offensive work, skip this entirely and leave the `engagement` block empty. Ask only if they mention pentesting, a bug bounty, a client, or a red team exercise:
   - What is in scope, precisely? Hosts, domains, repos, accounts.
   - What is explicitly out of scope?
   - What authorizes this work? A ticket, an SOW, a written approval. Record the reference, not the document.

   Explain plainly why you are asking: `vuln-triage` will not write proof-of-concept exploit code without a recorded scope, and `pentest-report` will not produce a client deliverable without one.

7. **Summarize and confirm.** Write back the profile in plain prose, not JSON. Six lines maximum. Ask: "Saving this. Anything wrong?"

8. **Write the config.** Create `.dragon-buddy/config.json` using the structure in `config.example.json` at the pack root. Fill every key you have a real answer for. Leave `[BRACKET]` placeholders only where the user genuinely did not know. Add `.dragon-buddy/` to `.gitignore` if the `engagement` block names a client or contains anything the repo should not carry.

9. **Hatch the buddy.** Call `buddy_status`. On first ever use this hatches a companion with a name and personality rolled once and kept for life. Show the returned card to the user exactly as it comes back. Offer `buddy_rename` if they want to name it themselves, and mention that personality and progress survive a rename.

   If the `buddy` MCP server is not connected, say so once, point at the install line in the pack README, and carry on. The pack is fully functional without it.

10. **Point at what is next.** Recommend exactly three skills for this specific repo, one line each, using the routing table in `references/setup-routing.md`. A public service with PII gets different advice than an internal CLI tool.

## Output format

```
Read off the repo: [stack summary]
Related repos: [name — relation, one per line — or omit the line entirely if this repo stands alone]
Exposure: [level] · Data: [sensitivity] · Compliance: [list or none]
Trust boundaries: [list or "to be derived in threat-model"]
Tooling: tests `[cmd]` · SCA [tool] · SAST [tool] · secrets [tool]
Fleet: [vendors, device count, backup system, out-of-band — or omit the line entirely if not managed]
Engagement: [scope reference, or "internal work, no engagement recorded"]

Saved to .dragon-buddy/config.json

[buddy status card, verbatim]

Start with:
1. `[skill]` — [why, for this repo specifically]
2. `[skill]` — [why]
3. `[skill]` — [why]
```

## Buddy (optional, when the MCP server is connected)

**Advise first.** Before you begin — and at any handoff to another skill — call `buddy_advise` with a one-line description of the *task* (the work, not a skill name). Load whatever it ranks highest if you have not already committed to a skill. It is advisory, never blocking: if the registered server has no `buddy_advise` (an older buddy, or the Fable build), route by hand via `buddy-setup`'s `references/setup-routing.md` and carry on. The `skills_used` you pass to `buddy_observe` below is exactly what trains that ranking, so always pass it.

Close the run by calling `buddy_observe`:
- `summary`: `"Onboarded <project> to dragon-dev-buddy: <exposure> exposure, <sensitivity> data."`
- `kind`: `config`
- `skills_used`: `["dragon-dev-buddy:buddy-setup"]`

Relay the reaction verbatim. If the server is not connected, skip silently.

## File output

Writes `.dragon-buddy/config.json`. No report. Nothing else in the repo is modified except a one-line `.gitignore` addition, and only with permission.

## Reference library

Load these for depth when the task calls for it:
- `references/setup-routing.md`: which three skills to recommend for each project shape, plus the full config key reference and which skills consume each key.
- `references/repo-survey.md`: what to look for in a first pass over an unfamiliar repo, by ecosystem, and how to infer exposure when the user is unsure.

## Worked example

See `examples/buddy-setup-run.md` for a complete onboarding run. Treat it as the quality target, not a script to copy verbatim.

## Quality bar

- The repo was surveyed before the first question was asked. Nothing was asked that the code already answers.
- No more than three messages of questions, batched.
- `exposure` and `data_sensitivity` are set to real values, never left as placeholders. Every severity rating downstream depends on them.
- The `engagement` block is empty for ordinary internal work rather than filled with invented values.
- Every entry in `related_repos` is one the user named out loud. A repo was never added because it sat in the same parent directory.
- `.dragon-buddy/config.json` is valid JSON: no trailing commas, no smart quotes, no comments outside the `_comment` keys.
- The buddy card is shown exactly as returned, not paraphrased or summarized.
- The three next-skill recommendations are specific to this repo. A public payments API and an internal CLI do not get the same list.
