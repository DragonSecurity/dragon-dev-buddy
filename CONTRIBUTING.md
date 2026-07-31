# Contributing

## Adding a skill

A skill is a directory under `skills/` containing `SKILL.md`, a `references/`
directory for depth, and an `examples/` directory holding one worked run.

```
skills/<name>/
  SKILL.md
  references/<topic>.md
  examples/<name>-run.md
```

`SKILL.md` opens with frontmatter and nothing else:

```markdown
---
name: <name>
description: <what it does>. Use when someone says "...", "...", or "...". <the opinion it holds>.
---
```

The description is the whole of the routing logic — it is the only part of the
skill Claude sees before deciding whether to load it. Write the phrases a user
would actually type, not a summary of the skill's contents. A description that
only says what a skill *is* never fires.

Then the body, in this order:

| Section | Purpose |
| --- | --- |
| Opening paragraphs | What the skill is for and the failure mode it exists to prevent. |
| `## First-run check` | Read `.dragon-buddy/config.json`, name the fields used, route to `buddy-setup` if absent. |
| `## Inputs` | What to ask for, and only what cannot be read from the repo. |
| `## Workflow` | Numbered steps. The opinionated part. |
| `## Output format` | A fenced skeleton of the deliverable. |
| `## Buddy` | The advise-before / observe-after contract. |
| `## File output` | Where the artifact goes, and what the skill must not modify. |
| `## Reference library` | One line per `references/` file saying when to load it. |
| `## Worked example` | Points at the `examples/` file as the quality target. |
| `## Quality bar` | The checklist a good run satisfies. |

## What the tests enforce

```sh
go test ./...
```

The pack is documentation, which means it has no runtime and therefore no
runtime failures. A broken reference link, a skill missing from the README, or a
name that disagrees with its directory does not error — it quietly does less than
it claims. `internal/skillpack` exists to turn that class of defect into a
failing build.

- **Name matches directory.** Claude addresses a skill by its frontmatter name;
  every cross-reference in this pack addresses it by its directory. They cannot
  drift.
- **Descriptions carry a `Use when` clause** and fit inside the 1024-character
  limit. Past the limit the tail is truncated — which is exactly where the
  trigger phrases live.
- **Every sidecar is reachable and every reference resolves**, in both
  directions. A `references/` file no `SKILL.md` tells the model to load is dead
  weight; a pointer to a file that does not exist is an instruction the model
  cannot follow. Pointing at *another* skill's reference file is allowed — most
  of the pack routes through `buddy-setup`'s routing table — but only if the
  owning skill is named in the same document.
- **Every skill ships a worked example and a quality bar.**
- **Every skill observes under its own qualified name.** `buddy_observe` with
  someone else's `skills_used` poisons the ranking `buddy_advise` returns, which
  degrades every skill in the pack, not just the one that lied.
- **No local absolute paths in the docs.** `/Users/...` and `/home/...` fail the
  build. This repo's own `secrets-and-config-audit` would flag it.
- **The README lists exactly the skills that exist**, count included.

Adding a skill therefore means touching three things: the directory, the README
table, and nothing else — the tests will tell you if you missed one.

## Style

Match the pack. It is written in a specific register and a skill that breaks it
reads as bolted on:

- **State the failure mode.** Every skill exists because something predictable
  goes wrong. Name it in the opening paragraph.
- **Prefer an opinion to a checklist.** "Ranked by risk reduction per unit of
  effort" beats "here are 40 things to check."
- **No hedging in the output format.** If a skill produces a verdict it produces
  a verdict; "possibly exploitable, recommend further investigation" is what
  these skills exist to replace.
- **Second person, present tense, no filler.** Do not open a section by
  restating its heading.

## Releasing

Bump `version` in `.claude-plugin/plugin.json`, add a `CHANGELOG.md` entry, then:

```sh
./scripts/build-plugin.sh
```

CI builds the same bundle on every push and attaches it as an artifact.
