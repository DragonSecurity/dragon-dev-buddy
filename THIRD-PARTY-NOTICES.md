# Third-party notices

This pack is licensed Apache-2.0 (see [LICENSE](LICENSE)). Part of it is derived
from third-party work under a different licence, and this file says which part
and under what terms.

## Matt Pocock — `mattpocock/skills`

- **Upstream:** https://github.com/mattpocock/skills
- **Licence:** MIT
- **Copyright:** © 2026 Matt Pocock

The material below was adopted, rewritten and extended for this pack. Wording,
structure and scope have changed — every adopted skill was rewritten against
this pack's conventions (a `.dragon-buddy/config.json` first-run check, a
`Use when` trigger clause, the buddy advise/observe contract, a worked example
and a quality bar) and given the security argument the pack exists to carry.
The MIT licence still governs the parts that came from upstream, and the notice
below travels with them.

### Skills derived from upstream skills

| This pack | Upstream skill |
| --- | --- |
| `skills/codebase-design/` | `skills/engineering/codebase-design` |
| `skills/design-interview/` | `skills/productivity/grilling` |
| `skills/git-guardrails/` | `skills/misc/git-guardrails-claude-code` |
| `skills/runbook-wizard/` | `skills/engineering/wizard` |
| `skills/session-handoff/` | `skills/productivity/handoff` |
| `skills/skill-authoring/` | `skills/productivity/writing-for-agents`, including its `SKILL-MECHANICS.md` |

Two shell scripts at the repository root ship with those skills and carry the
same provenance:

| This pack | Upstream file |
| --- | --- |
| `scripts/wizard-template.sh` | `skills/engineering/wizard/template.sh` |
| `scripts/block-dangerous-git.sh` | the hook script in `skills/misc/git-guardrails-claude-code` |

### Reference material harvested into existing skills

These skills are this pack's own; the sections named below are derived from
upstream and were merged into them.

| This pack | What was adopted | Upstream skill |
| --- | --- | --- |
| `skills/debug-and-fix/references/debugging-method.md` and its SKILL.md step 1 | building and tightening a feedback loop, the red-capable gate, instrumentation hygiene, the measure-first branch for performance regressions, and what to ask for when no loop can be built | `skills/engineering/diagnosing-bugs` |
| `skills/security-test-writer/references/security-test-patterns.md` | the structural test anti-patterns — tautological, implementation-coupled, horizontal slicing — and agreeing seams before writing | `skills/engineering/tdd` |
| `skills/secure-code-review/references/pr-review.md` and its SKILL.md step 7 | the spec-conformance axis: reviewing a change against its originating spec as a second, separately reported axis | `skills/engineering/code-review` |
| `skills/secure-feature-build/SKILL.md` | the **Out of scope** section of the spec template | `skills/engineering/to-spec` |

### MIT licence

```
MIT License

Copyright (c) 2026 Matt Pocock

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
