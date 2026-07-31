// Package skillpack loads the Dragon Dev Buddy skill pack off disk so its
// structure and conventions can be asserted in tests.
//
// A skill pack is documentation, which means nothing here fails at runtime the
// way a compiler failure does. A skill with a broken reference link, a name that
// disagrees with its directory, or a description that never made it into the
// README does not error — it just quietly does less than it claims. That class
// of defect is what this package exists to make loud.
package skillpack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SkillsDir is the pack directory holding one subdirectory per skill.
const SkillsDir = "skills"

// ManifestPath is the Claude Code plugin manifest, relative to the pack root.
const ManifestPath = ".claude-plugin/plugin.json"

// Skill is one skill directory: the SKILL.md frontmatter, its body, and the
// sidecar files under references/ and examples/.
type Skill struct {
	// Name is the frontmatter `name` field.
	Name string
	// Description is the frontmatter `description` field.
	Description string
	// DirName is the directory the skill lives in, which must equal Name.
	DirName string
	// Dir is the path to the skill directory.
	Dir string
	// Body is everything in SKILL.md below the closing frontmatter delimiter.
	Body string
	// Sidecars are paths of files under references/ and examples/, relative to
	// Dir and slash-separated, e.g. "references/review-patterns.md".
	Sidecars []string
}

// Path returns the path to the skill's SKILL.md.
func (s Skill) Path() string { return filepath.Join(s.Dir, "SKILL.md") }

// Plugin is the .claude-plugin/plugin.json manifest.
type Plugin struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      struct {
		Name string `json:"name"`
	} `json:"author"`
}

// LoadPlugin reads and decodes the plugin manifest from the pack root.
func LoadPlugin(root string) (Plugin, error) {
	var p Plugin
	path := filepath.Join(root, filepath.FromSlash(ManifestPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	// DisallowUnknownFields: a typo'd key in the manifest is silently ignored by
	// the plugin loader, which is exactly how a manifest ends up half-applied.
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return p, fmt.Errorf("%s: %w", ManifestPath, err)
	}
	return p, nil
}

// LoadSkills reads every skill under root/skills, sorted by name.
func LoadSkills(root string) ([]Skill, error) {
	base := filepath.Join(root, SkillsDir)
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}

	var skills []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s, err := loadSkill(filepath.Join(base, e.Name()), e.Name())
		if err != nil {
			return nil, err
		}
		skills = append(skills, s)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].DirName < skills[j].DirName })
	return skills, nil
}

func loadSkill(dir, dirName string) (Skill, error) {
	s := Skill{Dir: dir, DirName: dirName}

	raw, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return s, err
	}

	fields, body, err := parseFrontmatter(string(raw))
	if err != nil {
		return s, fmt.Errorf("%s/SKILL.md: %w", dirName, err)
	}
	s.Name = fields["name"]
	s.Description = fields["description"]
	s.Body = body

	for _, sub := range []string{"references", "examples"} {
		files, err := os.ReadDir(filepath.Join(dir, sub))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return s, err
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			s.Sidecars = append(s.Sidecars, sub+"/"+f.Name())
		}
	}
	sort.Strings(s.Sidecars)
	return s, nil
}

// parseFrontmatter splits a `---` delimited YAML header off the front of a
// document. The header in this pack is only ever flat `key: value` pairs on a
// single line, so this deliberately does not pull in a YAML dependency — a
// parser that accepts more than the convention allows would stop enforcing it.
func parseFrontmatter(src string) (map[string]string, string, error) {
	const delim = "---"

	lines := strings.Split(src, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != delim {
		return nil, "", fmt.Errorf("does not open with a %q frontmatter delimiter", delim)
	}

	fields := map[string]string{}
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == delim {
			return fields, strings.Join(lines[i+1:], "\n"), nil
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, "", fmt.Errorf("frontmatter line %d is not `key: value`: %q", i+1, line)
		}
		key = strings.TrimSpace(key)
		if _, dup := fields[key]; dup {
			return nil, "", fmt.Errorf("frontmatter key %q appears twice", key)
		}
		fields[key] = strings.TrimSpace(value)
	}
	return nil, "", fmt.Errorf("frontmatter is never closed by %q", delim)
}

// sidecarRef matches a references/ or examples/ path as written in prose,
// whether or not it is wrapped in backticks or a markdown link.
var sidecarRef = regexp.MustCompile(`(?:references|examples)/[A-Za-z0-9._-]+\.md`)

// SidecarsMentioned returns the set of sidecar paths the skill body points at.
func (s Skill) SidecarsMentioned() map[string]bool {
	found := map[string]bool{}
	for _, m := range sidecarRef.FindAllString(s.Body, -1) {
		found[m] = true
	}
	return found
}

// kebabToken matches a backticked lowercase kebab-case word — the shape this
// pack writes skill names in when one skill hands off to another.
var kebabToken = regexp.MustCompile("`([a-z][a-z0-9]*(?:-[a-z0-9]+)+)`")

// KebabTokens returns the distinct backticked kebab-case tokens in the body.
// Tokens containing a dot or slash are excluded: those are filenames, not skill
// references.
func (s Skill) KebabTokens() []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range kebabToken.FindAllStringSubmatch(s.Body, -1) {
		tok := m[1]
		if strings.ContainsAny(tok, "./") || seen[tok] {
			continue
		}
		seen[tok] = true
		out = append(out, tok)
	}
	sort.Strings(out)
	return out
}

// EditDistance is the Levenshtein distance between a and b. It is used to catch
// a mistyped skill handoff (`security-test-writter`) without flagging every
// unrelated hyphenated word in the prose.
func EditDistance(a, b string) int {
	if a == b {
		return 0
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, min(curr[j-1]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}
