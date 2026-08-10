# Skill mechanics and this pack's contract

The skill-specific branch of writing for agents: what changes when the document is a skill rather than a `CLAUDE.md` or a reference file. Everything else about writing it is the lever set in the sibling reference.

## Frontmatter

A skill opens with `---` delimited frontmatter holding flat `key: value` pairs on single lines. This pack's parser is deliberately not a YAML parser — a parser that accepted more than the convention allows would stop enforcing it. What that means in practice:

- Exactly two keys: `name` and `description`. No `metadata`, no nested block, no list.
- Every line inside the delimiters is `key: value` or blank. A line with no colon is a parse error, which is what a folded multi-line description produces.
- A duplicate key is an error rather than a last-one-wins.
- `name` must equal the directory name and be lowercase kebab-case, matching `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`. Claude Code addresses the skill by the frontmatter name; humans and every cross-reference in this pack address it by the directory. A mismatch breaks invocation and nothing else notices.
- `description` is one line, under 1024 characters, and contains the literal string `Use when`. Colons inside the value are fine — only the first one splits the line.

The 1024 limit is where the description is truncated, not where it is rejected. This pack puts its trigger phrases at the end of the description, so an over-long description loses exactly the phrases that make the skill fire, and keeps the part that describes what it is. It looks fine and stops working.

## Invocation

Two choices, trading the two loads:

- A **model-invoked** skill keeps a `description`, so the agent can fire it on its own and other skills can reach it by name. Typing the name still works — model invocation *includes* user reach; a description only adds agent discovery, it never removes the human's. The description is the skill's top-level context pointer, forced to stay loaded at all times: permanent context load in exchange for discoverability. A model-invoked skill whose content is all reference is also a home for shared reference, since another skill can invoke it. Mechanics: omit `disable-model-invocation` and write a model-facing description carrying the trigger branches.
- A **user-invoked** skill strips the description from the agent's reach. Only a human typing its name can invoke it, and no other skill can. Zero context load, but it spends cognitive load — you become the index that has to remember it exists. Mechanics: set `disable-model-invocation: true`, and the description becomes human-facing, with the trigger list stripped.

Pick model invocation only when the agent must reach the skill on its own, or another skill must. If it only ever fires by hand, user invocation costs nothing per turn.

**In this pack the choice is already made.** Every skill here is model-invoked, the frontmatter carries exactly two keys, and the tests require a `Use when` clause in every description — so `disable-model-invocation` is not an option a new skill can take. The consequence is worth stating plainly: a skill added here is a description loaded on every turn of every session, forever, for every user of the pack. That is the bar a new skill has to clear.

Shared reference that two user-invoked skills both need can live in neither of them, since with no descriptions neither can fire the other. Push it to a plain file outside the skill system that any document can point at. In this pack the equivalent move is a reference file under one skill that another skill points at by naming its owner — most of the pack routes through `buddy-setup`'s routing table that way.

## Splitting by invocation

The invocation cut, beside the sequence cut in the lever reference: split off a model-invoked skill when it has a distinct leading word that should trigger it on its own — a word you actually type in your prompts — or when another skill must reach it by name. You pay context load for a new always-loaded description, so the independent reach has to be worth it.

Two signals that a split is *not* earned: the proposed skill fires on the same phrases as an existing one, or it is only ever reached by another skill in the middle of a run. The first is a branch of the existing skill. The second is a reference file under the skill that reaches it.

## Router skills

When user-invoked skills multiply past what a human can remember, that piled-up cognitive load is cured by a **router skill**: one user-invoked skill that names the others and says when to reach for each, so there is one thing to remember instead of many. It can only hint, never fire them — user-invoked skills have no description, so nothing but the human reaches them.

This pack solves the same problem from the other end. Every skill is model-invoked, so nothing needs routing by hand; `security-audit-orchestrator` is a chainer rather than a router, invoking skills in dependency order for a full pass, and `buddy-companion` carries the advice ranking that answers "which skill fits this task" at runtime.

## The test contract, as a checklist

Run a draft against this before running `go test ./...`. Every item is a check in `internal/skillpack/skillpack_test.go`.

**The skill directory**

- [ ] `skills/<name>/SKILL.md` exists, opens with `---`, and its frontmatter holds only `name` and `description`.
- [ ] `name` equals the directory name and is lowercase kebab-case.
- [ ] `description` is one line, under 1024 characters, contains `Use when`, and the trigger phrases follow it.
- [ ] At least one file under `references/`, and one worked example under `examples/`.
- [ ] Only `.md` files under `references/` and `examples/`. Every file in those directories is treated as a sidecar, and only a `.md` path can be named in prose — a `.sh` there is unreferenceable by construction and fails the build.
- [ ] A shell sidecar lives in the repo-root `scripts/` directory instead, invoked through `${CLAUDE_PLUGIN_ROOT}`.

**Sidecar references, both directions**

- [ ] Every file created under `references/` or `examples/` is named in SKILL.md prose, spelled exactly as the path sits on disk. A file no SKILL.md ever tells the model to load is the failure that is hard to notice by eye: the file is right there, and never read.
- [ ] Every sidecar path named in prose exists somewhere in the pack.
- [ ] A path owned by another skill is only resolvable when that skill's name appears in backticks in the same document. "Load the routing reference" with no owner named is an instruction the model cannot follow.

**The body**

- [ ] A `## Buddy` heading is present. This pack writes it as `## Buddy (optional, when the MCP server is connected)`, which satisfies the check and tells the reader what happens when no server is connected.
- [ ] The body contains `dragon-dev-buddy:<name>` — the skill's own qualified name, passed to `buddy_observe`.
- [ ] The body contains no *other* skill's qualified name inside a backticked array. That form is read as the skill reporting itself under someone else's name, which is exactly what it would do at runtime, and it poisons the ranking every skill in the pack depends on.
- [ ] A `## Quality bar` heading is present.
- [ ] Every backticked kebab-case token that is within one edit of a real skill name *is* that skill name. Near-misses are reported as typos, so do not invent a name that is one character away from an existing one.
- [ ] No absolute home path anywhere in any markdown file. Use `~` or a placeholder.
- [ ] If the document names a `buddy-mcp` major version, it is the same major every other skill names and the same one `.mcp.json` installs. Two majors in the pack's prose is a hard failure — the docs have to agree before the declaration can be checked against them.

**Outside the directory**

- [ ] The README lists the skill as a table row of the form `| \`<name>\` | … |`, in the group it belongs to. The row count must equal the skill count, so a renamed or deleted skill leaves a failing row behind.
- [ ] The README's prose count says the right number of skills. A spelled-out count is wrong within two commits of anyone adding anything, which is why it is pinned.
- [ ] `CHANGELOG.md` has an entry under `## Unreleased`.

**The version, in three places**

`.claude-plugin/plugin.json` carries `version`. `.claude-plugin/marketplace.json` carries the same version on this pack's entry, along with a `description` and `author.name` that must match the manifest field for field, and a `source` that resolves to this repository's root. `CHANGELOG.md` carries the release headings.

The tests require the manifest version to equal the *highest* released changelog version — found by comparing versions numerically, not by taking the top heading — and require the changelog's sections to descend with no version used twice. `## Unreleased` is skipped, so a skill added while a release is in flight does not force a bump; the three numbers move together in the commit that cuts the release, before the tag exists. Adding a skill is a minor bump when that release comes, because every name that pointed at the pack still points at something.

Nothing reaches an installed copy until the manifest version goes up — auto-update compares that and only that — so a release that forgets the bump is a release nobody receives.
