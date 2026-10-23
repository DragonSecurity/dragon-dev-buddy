# Routing evals

Whether a skill's description gets it loaded for the requests it exists for.
`go test ./...` proves the files are well-formed; only the model can say whether
the description fires, so this suite asks it, with
[`claude plugin eval`](https://code.claude.com/docs/en/plugin-evals)
(Claude Code 2.1.269 or later).

```sh
# Every case, three runs each, against a no-plugin baseline.
claude plugin eval . --no-publish

# One case while iterating on a description: cheaper, no baseline arm.
claude plugin eval . --case 'routes-release-*' --runs 3 --ablation none --no-publish
```

Every run is a real model call on your account, and a full suite is roughly
cases × runs × 2 sessions. That is why it is run by hand, not in CI; pass
`--max-cost-usd` when you want a ceiling.

## What is here

- `routes-*` — a request phrased the way a user would type it, never naming the
  skill, and a `tool_used: Skill` grader for the skill it should load. The
  prompts are chosen where the trigger is easy to confuse with a neighbour's:
  one CVE is `vuln-triage`, not `dependency-audit`; reviewing one handler is
  `secure-code-review`, not `ship-it`.
- `no-skill-for-general-knowledge` — the negative case. A pack whose
  descriptions fire on everything is as broken as one that fires on nothing.
- `mocks/buddy/` — fixed answers for the buddy tools, so a skill that calls
  `buddy_advise` or `buddy_observe` gets a reply without the real server, which
  eval runs do not start. `_tools.json` is the server's own `tools/list`, so the
  model sees the real schemas.

## Reading a result

In a two-arm run a `tool_used: Skill` grader is reported but not scored, since it
can never pass without the plugin. Read it as the routing verdict: a case whose
`skill-fired` indicator fails means the description does not trigger on that
phrasing, and the fix is in the skill's frontmatter, not here. Rerun the same
case after changing it.

`buddy_skills` reports the other half from real use: a skill the buddy's
stocktake lists as "fit your work, never loaded" is the one to write a case for.

## What this cannot test

The skill tracker hook (`hooks/skill-tracker.mjs`) rewrites `buddy_observe`'s
input through `updatedInput`. Under `claude plugin eval` the hook runs, but a
mocked MCP tool records the model's original input rather than the rewritten
one, so a case asserting on `mock_calls` fails however the hook behaves. A real
MCP server does receive the rewrite. The tracker is covered by
`scripts/check-skill-tracker.sh` instead.
