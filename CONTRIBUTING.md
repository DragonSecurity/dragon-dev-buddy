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
- **One version, written in three places, agreeing.** The manifest, the
  marketplace entry and the newest released heading in the changelog carry the
  same `X.Y.Z`. Auto-update compares the manifest version and nothing else, so
  the two copies that drift are the two nobody notices: a marketplace entry left
  a release behind advertises something no one can install, and a changelog that
  never got its heading renamed leaves the release with no description at all.
- **The marketplace entry describes this pack**, not a stale ancestor of it —
  same name, same description, pointed at this repository. It is duplicated
  metadata, and duplicated metadata is only safe while something checks it.
- **`.mcp.json` declares the buddy server as a git spec pinned to a major.**
  buddy-mcp is never published to the npm registry, so the specifier is
  `github:DragonSecurity/buddy-mcp#semver:^2` and the range resolves against that
  repository's tags. Every skill here calls `buddy_advise` and `buddy_observe`;
  a spec that tracked a branch, or a range that reached past the major, would let
  a buddy-mcp 3.0 change those tools out from under the whole pack on a machine
  where nothing was changed and nothing was released. The consumer compiles the
  server from the tag, so this is also the line that decides how long a
  contributor's first session waits.

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

A release is a tag on `main`; everything after the tag is automated. The version
is written in three places and the tests refuse to let them disagree, so change
all three in the same commit:

1. In `CHANGELOG.md`, rename `## Unreleased` to `## X.Y.Z — YYYY-MM-DD` and open
   a fresh empty `## Unreleased` above it.
2. Set `version` in `.claude-plugin/plugin.json`, and the `version` of the
   `dragon-dev-buddy` entry in `.claude-plugin/marketplace.json`, to that same
   `X.Y.Z`.
3. Run `go test ./...`. It now enforces manifest, changelog and marketplace
   agreement, which is worth catching here: everything else in this repo can be
   quietly corrected in the next commit, and a tag cannot.
4. Merge to `main`, then tag the merged commit and push the tag:

```sh
git switch main && git pull        # tag the commit CI green-lit, not a branch tip
git tag vX.Y.Z
git push origin vX.Y.Z
```

Tag `main` and only `main`. A tag on a branch cuts a release from a tree that
never landed, and the result is indistinguishable from a real one until someone
installs it and finds work that is not on `main`.

`.github/workflows/release.yml` picks the tag up, builds
`dist/dragon-dev-buddy-X.Y.Z.plugin` the same way `scripts/build-plugin.sh` does
locally, and cuts the GitHub release with the bundle attached. CI also builds
that bundle on every push and attaches it as an artifact, so a broken build
surfaces on the pull request rather than at the tag.

Nothing is published to a package registry — not this pack, and not the buddy
server it depends on. For the pack, the GitHub release and the marketplace entry
in this repository are the whole distribution channel; for buddy-mcp, the tags
are, because `npx -y github:DragonSecurity/buddy-mcp#semver:^2` resolves `^2`
against them and installs by cloning and compiling whichever tag wins. In both
repositories the tag is what someone else's install actually fetches, which is
why step 3 exists before step 4 does.

**The buddy-mcp tag has to exist before this one does.** npm resolves the
`#semver:^2` in `.mcp.json` against the tags in `DragonSecurity/buddy-mcp` at the
moment a session starts, not at the moment this pack was released, so a tag here
that reaches someone before a matching `v2.x` tag is pushed there points at a
range with nothing in it. npm fails the install outright — `npm error code
ENOVERSIONS` — Claude Code reports a server that would not start, no `buddy_*`
tool is in the session at all, and every skill in the pack quietly takes its
no-buddy path while looking installed and healthy. Push the buddy-mcp tag first,
confirm it with `git ls-remote --tags https://github.com/DragonSecurity/buddy-mcp`,
and only then tag here. The other order leaves a window where a fresh install of
this pack is broken, and that window is exactly as long as the gap between the
two pushes.

Which number moves is the README's Versioning section: for a skill pack a
breaking change is a renamed skill or a changed buddy contract, not a rewritten
paragraph. Nothing reaches an installed copy until the manifest version goes up —
auto-update compares that and only that — so a release that forgets the bump is
a release nobody receives.
