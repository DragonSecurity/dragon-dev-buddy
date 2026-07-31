package skillpack_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/DragonSecurity/dragon-dev-buddy/internal/skillpack"
)

// root is the pack root, relative to this package's directory.
const root = "../.."

// descriptionLimit is the ceiling Claude Code applies to a skill's frontmatter
// description. Past it the skill still loads but the tail is cut, which silently
// drops the trigger phrases at the end — where this pack puts most of them.
const descriptionLimit = 1024

func loadSkills(t *testing.T) []skillpack.Skill {
	t.Helper()
	skills, err := skillpack.LoadSkills(root)
	if err != nil {
		t.Fatalf("loading skills: %v", err)
	}
	if len(skills) == 0 {
		t.Fatal("no skills found; the pack cannot be empty")
	}
	return skills
}

func readFile(t *testing.T, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(raw)
}

func TestPluginManifest(t *testing.T) {
	p, err := skillpack.LoadPlugin(root)
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}

	if p.Name != "dragon-dev-buddy" {
		t.Errorf("manifest name = %q, want %q; the name is the skill-invocation prefix and cannot drift", p.Name, "dragon-dev-buddy")
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(p.Version) {
		t.Errorf("manifest version = %q, want semver", p.Version)
	}
	if p.Description == "" {
		t.Error("manifest description is empty")
	}
	if p.Author.Name == "" {
		t.Error("manifest author.name is empty")
	}
}

// TestSkillNameMatchesDirectory guards the one mismatch that breaks invocation:
// Claude Code addresses a skill by its frontmatter name, humans and every
// cross-reference in this pack address it by its directory.
func TestSkillNameMatchesDirectory(t *testing.T) {
	for _, s := range loadSkills(t) {
		if s.Name == "" {
			t.Errorf("%s: frontmatter has no name", s.Path())
			continue
		}
		if s.Name != s.DirName {
			t.Errorf("%s: frontmatter name %q != directory %q", s.Path(), s.Name, s.DirName)
		}
		if !regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`).MatchString(s.Name) {
			t.Errorf("%s: name %q is not lowercase kebab-case", s.Path(), s.Name)
		}
	}
}

// TestSkillDescriptions enforces the trigger-phrase convention. A description
// that only says what a skill *is* never fires; the pack's descriptions all say
// what a user would have typed.
func TestSkillDescriptions(t *testing.T) {
	for _, s := range loadSkills(t) {
		switch {
		case s.Description == "":
			t.Errorf("%s: frontmatter has no description", s.Path())
		case len(s.Description) > descriptionLimit:
			t.Errorf("%s: description is %d chars, over the %d limit; the trigger phrases at the end will be truncated", s.Path(), len(s.Description), descriptionLimit)
		case !strings.Contains(s.Description, "Use when"):
			t.Errorf("%s: description has no \"Use when\" trigger clause, so the skill will not reliably fire", s.Path())
		case strings.Contains(s.Description, "\n"):
			t.Errorf("%s: description spans multiple lines; keep it on one", s.Path())
		}
	}
}

// TestSidecarsAreReferenced catches both directions of reference rot: a skill
// pointing at a reference file that does not exist, and a reference file no
// skill ever tells the model to load. The second is the common one and the
// harder to notice — the file is right there, and never read.
//
// A skill may point at another skill's reference file (most of the pack routes
// through buddy-setup's routing table). That resolves only when the owning
// skill is named in the same document, because "load references/foo.md" with no
// owner named is an instruction the model cannot follow.
func TestSidecarsAreReferenced(t *testing.T) {
	skills := loadSkills(t)

	owners := map[string][]string{}
	for _, s := range skills {
		for _, f := range s.Sidecars {
			owners[f] = append(owners[f], s.Name)
		}
	}

	for _, s := range skills {
		own := map[string]bool{}
		for _, f := range s.Sidecars {
			own[f] = true
		}
		mentioned := s.SidecarsMentioned()

		for path := range mentioned {
			if own[path] {
				continue
			}
			var resolved bool
			for _, owner := range owners[path] {
				if strings.Contains(s.Body, "`"+owner+"`") {
					resolved = true
					break
				}
			}
			switch {
			case len(owners[path]) == 0:
				t.Errorf("%s: points at %s, which does not exist anywhere in the pack", s.Path(), path)
			case !resolved:
				t.Errorf("%s: points at %s without naming the skill that owns it (%s)", s.Path(), path, strings.Join(owners[path], ", "))
			}
		}

		for _, path := range s.Sidecars {
			if mentioned[path] {
				continue
			}
			// Another skill may be the only one that routes to this file.
			var referencedElsewhere bool
			for _, other := range skills {
				if other.Name != s.Name && other.SidecarsMentioned()[path] {
					referencedElsewhere = true
					break
				}
			}
			if !referencedElsewhere {
				t.Errorf("%s: %s exists but no SKILL.md ever tells the model to load it", s.Path(), path)
			}
		}
	}
}

// TestEverySkillHasAWorkedExample holds the pack to its own promise: each skill
// ships one example that doubles as its quality target.
func TestEverySkillHasAWorkedExample(t *testing.T) {
	for _, s := range loadSkills(t) {
		var examples int
		for _, f := range s.Sidecars {
			if strings.HasPrefix(f, "examples/") {
				examples++
			}
		}
		if examples == 0 {
			t.Errorf("%s: no examples/ file", s.Path())
		}
	}
}

// TestBuddyReportingContract checks the convention documented in
// buddy-companion: every skill closes by observing under its own qualified name.
// A skill reporting someone else's name poisons the ranking the buddy builds.
func TestBuddyReportingContract(t *testing.T) {
	for _, s := range loadSkills(t) {
		if !strings.Contains(s.Body, "## Buddy") {
			t.Errorf("%s: no \"## Buddy\" section; every skill reports what it did", s.Path())
			continue
		}
		want := fmt.Sprintf("dragon-dev-buddy:%s", s.Name)
		if !strings.Contains(s.Body, want) {
			t.Errorf("%s: never passes %q to buddy_observe(skills_used)", s.Path(), want)
		}
		for _, other := range loadSkills(t) {
			if other.Name == s.Name {
				continue
			}
			qualified := fmt.Sprintf("`[\"dragon-dev-buddy:%s\"]`", other.Name)
			if strings.Contains(s.Body, qualified) {
				t.Errorf("%s: reports itself as %s", s.Path(), other.Name)
			}
		}
	}
}

// TestEverySkillHasAQualityBar keeps the pack honest about what "done" means.
func TestEverySkillHasAQualityBar(t *testing.T) {
	for _, s := range loadSkills(t) {
		if !strings.Contains(s.Body, "## Quality bar") {
			t.Errorf("%s: no \"## Quality bar\" section", s.Path())
		}
	}
}

// TestHandoffsResolve catches a mistyped skill name in a handoff. It only flags
// tokens within one edit of a real skill, so ordinary hyphenated prose
// ("rate-limit", "merge-base") does not trip it.
func TestHandoffsResolve(t *testing.T) {
	skills := loadSkills(t)
	known := map[string]bool{}
	for _, s := range skills {
		known[s.Name] = true
	}

	for _, s := range skills {
		for _, tok := range s.KebabTokens() {
			if known[tok] {
				continue
			}
			for name := range known {
				if skillpack.EditDistance(tok, name) == 1 {
					t.Errorf("%s: `%s` looks like a typo for `%s`", s.Path(), tok, name)
				}
			}
		}
	}
}

// homePath matches a real absolute home directory — a named user segment
// followed by more path. The elided form (`/Users/...`) is how the docs discuss
// this rule, so it must not trip it.
var homePath = regexp.MustCompile(`/(?:Users|home)/([A-Za-z0-9._-]+)/`)

// TestNoPersonalPaths stops a local absolute path reaching a published repo.
// The pack's own secrets-and-config-audit would flag it, so it should not have
// to.
func TestNoPersonalPaths(t *testing.T) {
	var offenders []string
	err := filepath.WalkDir(filepath.Join(root), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(raw), "\n") {
			for _, m := range homePath.FindAllStringSubmatch(line, -1) {
				if strings.Trim(m[1], ".") == "" {
					continue // an elided placeholder, not a real path
				}
				offenders = append(offenders, fmt.Sprintf("%s:%d: %s", path, i+1, m[0]))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking pack: %v", err)
	}
	for _, o := range offenders {
		t.Errorf("%s: absolute home path in documentation; use ~ or a placeholder", o)
	}
}

// TestReadmeListsEverySkill is the drift check that matters most in practice:
// a skill added to the pack and never added to the table nobody can find it in.
func TestReadmeListsEverySkill(t *testing.T) {
	readme := readFile(t, "README.md")
	skills := loadSkills(t)

	for _, s := range skills {
		if !strings.Contains(readme, "`"+s.Name+"`") {
			t.Errorf("README.md does not list `%s`", s.Name)
		}
	}

	// And the reverse: a row left behind by a renamed or deleted skill.
	known := map[string]bool{}
	for _, s := range skills {
		known[s.Name] = true
	}
	rows := regexp.MustCompile("(?m)^\\| `([a-z][a-z0-9-]+)` \\|").FindAllStringSubmatch(readme, -1)
	for _, m := range rows {
		if !known[m[1]] {
			t.Errorf("README.md lists `%s`, which is not a skill in the pack", m[1])
		}
	}
	if len(rows) != len(skills) {
		t.Errorf("README.md has %d skill rows, the pack has %d skills", len(rows), len(skills))
	}
}

// TestReadmeSkillCountIsCurrent pins the prose count to reality. Spelled-out
// counts in a README are wrong within two commits of anyone adding anything.
func TestReadmeSkillCountIsCurrent(t *testing.T) {
	readme := readFile(t, "README.md")
	want := fmt.Sprintf("%d skills", len(loadSkills(t)))
	if !strings.Contains(readme, want) {
		t.Errorf("README.md does not say %q; update the count", want)
	}
}
