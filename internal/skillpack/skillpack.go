// Package skillpack loads the Dragon Dev Buddy skill pack off disk so its
// structure and conventions can be asserted in tests.
//
// A skill pack is documentation, which means nothing here fails at runtime the
// way a compiler failure does. A skill with a broken reference link, a name that
// disagrees with its directory, or a description that never made it into the
// README does not error — it just quietly does less than it claims. That class
// of defect is what this package exists to make loud.
//
// The release metadata is the same problem wearing a different hat. The pack's
// version is written down three times — in the plugin manifest, in the
// marketplace entry that advertises it, and at the top of the changelog — and
// nothing at runtime reads two of them together. When they disagree, installs
// keep succeeding: the marketplace hands out a version number that no longer
// describes the contents, and Claude Code's auto-update compares the installed
// version against a published one that never moved, so nobody is offered the
// change. The loaders below exist so the copies can be compared to each other in
// a test instead of trusted.
package skillpack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// SkillsDir is the pack directory holding one subdirectory per skill.
const SkillsDir = "skills"

// ManifestPath is the Claude Code plugin manifest, relative to the pack root.
const ManifestPath = ".claude-plugin/plugin.json"

// MarketplacePath is the marketplace manifest this repository publishes about
// itself, relative to the pack root. It repeats the plugin's name, version,
// description and author, and it is the copy a user installing the pack
// actually reads.
const MarketplacePath = ".claude-plugin/marketplace.json"

// ChangelogPath is the changelog, relative to the pack root. Its topmost
// released heading is the third copy of the version.
const ChangelogPath = "CHANGELOG.md"

// MCPConfigPath is the project-scoped MCP server declaration, relative to the
// pack root. Every skill in this pack calls the server it declares.
const MCPConfigPath = ".mcp.json"

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

// Author is the `author` object, shared by the plugin manifest and the
// marketplace entry that repeats it.
type Author struct {
	Name string `json:"name"`
}

// Plugin is the .claude-plugin/plugin.json manifest.
type Plugin struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      Author `json:"author"`
}

// LoadPlugin reads and decodes the plugin manifest from the pack root.
func LoadPlugin(root string) (Plugin, error) {
	var p Plugin
	err := decodeManifest(root, ManifestPath, &p)
	return p, err
}

// decodeManifest reads the JSON file at rel, relative to root, into v.
//
// DisallowUnknownFields: a typo'd key in any of these files is silently ignored
// by the loader that consumes it, which is exactly how a manifest ends up
// half-applied — `descripton` costs a description, `arg` costs the arguments the
// buddy server is launched with, and nothing complains at the point of use. The
// same discipline applies to every file here, so it lives in one place.
func decodeManifest(root, rel string, v any) error {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	return nil
}

// Version is a plain `major.minor.patch` release version.
type Version struct {
	Major, Minor, Patch int
}

// versionPattern is semver's numeric core and nothing else. The pack has only
// ever shipped plain releases, and — as with the frontmatter parser above —
// accepting more than the convention allows would stop enforcing it.
var versionPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

// ParseVersion parses a `major.minor.patch` version string.
func ParseVersion(s string) (Version, error) {
	m := versionPattern.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("%q is not a `major.minor.patch` version", s)
	}
	// Every group is digits with no leading zero, so none of these can fail.
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// String renders the version the way the manifests write it.
func (v Version) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch) }

// Compare returns -1 when v precedes o, 0 when they are the same release, and 1
// when v follows o. It compares each field as a number on purpose: as strings
// "1.10.0" sorts *before* "1.9.0", so a changelog ordering check built on string
// comparison passes on exactly the release that breaks it.
func (v Version) Compare(o Version) int {
	for _, pair := range [][2]int{{v.Major, o.Major}, {v.Minor, o.Minor}, {v.Patch, o.Patch}} {
		switch {
		case pair[0] < pair[1]:
			return -1
		case pair[0] > pair[1]:
			return 1
		}
	}
	return 0
}

// MarketplacePlugin is one entry in a marketplace's plugin list. Every field
// except Source is a second copy of something plugin.json already says.
type MarketplacePlugin struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Author      Author `json:"author"`
}

// Marketplace is the .claude-plugin/marketplace.json manifest.
type Marketplace struct {
	Name     string `json:"name"`
	Owner    Author `json:"owner"`
	Metadata struct {
		Description string `json:"description"`
	} `json:"metadata"`
	Plugins []MarketplacePlugin `json:"plugins"`
}

// LoadMarketplace reads and decodes the marketplace manifest from the pack root.
func LoadMarketplace(root string) (Marketplace, error) {
	var m Marketplace
	err := decodeManifest(root, MarketplacePath, &m)
	return m, err
}

// Entry returns the marketplace's listing for the named plugin.
func (m Marketplace) Entry(name string) (MarketplacePlugin, bool) {
	for _, p := range m.Plugins {
		if p.Name == name {
			return p, true
		}
	}
	return MarketplacePlugin{}, false
}

// Release is one released version heading in the changelog.
type Release struct {
	// Version is the version the heading names.
	Version Version
	// Date is whatever followed the version on the heading line, empty when the
	// heading carried nothing but a version.
	Date string
	// Line is the 1-indexed line the heading is on, so a failure can point at it.
	Line int
}

// releaseHeading matches the pack's release heading: a bare version, optionally
// followed by a dash and a date.
var releaseHeading = regexp.MustCompile(`^(\d+\.\d+\.\d+)(?:\s*[—–-]\s*(\S.*))?$`)

// attemptedRelease matches a heading that was *trying* to be a release heading —
// it leads with a digit, or with a `v` and a digit — so a misspelled one can be
// reported rather than skipped. A skipped heading is the failure mode this whole
// file is about: the version quietly stops being the newest release and every
// check downstream compares against the wrong number.
var attemptedRelease = regexp.MustCompile(`^v?\d`)

// ChangelogVersions returns the released versions in the order the changelog
// lists them, newest first by convention — which is a convention a test has to
// enforce, since nothing else reads the file in order.
//
// "Unreleased" is skipped rather than reported: it is a standing heading that
// names no version, and its whole job is to hold notes for a release that has
// not been cut yet.
func ChangelogVersions(root string) ([]Release, error) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ChangelogPath)))
	if err != nil {
		return nil, err
	}

	var releases []Release
	for i, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		if strings.EqualFold(heading, "unreleased") {
			continue
		}
		m := releaseHeading.FindStringSubmatch(heading)
		if m == nil {
			if attemptedRelease.MatchString(heading) {
				return nil, fmt.Errorf("%s:%d: %q is not a release heading; write `## <major>.<minor>.<patch>` optionally followed by a date", ChangelogPath, i+1, heading)
			}
			continue
		}
		v, err := ParseVersion(m[1])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", ChangelogPath, i+1, err)
		}
		releases = append(releases, Release{Version: v, Date: m[2], Line: i + 1})
	}
	return releases, nil
}

// MCPServer is one entry in .mcp.json's `mcpServers` map.
type MCPServer struct {
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// MCPConfig is the .mcp.json project MCP server declaration.
type MCPConfig struct {
	Servers map[string]MCPServer `json:"mcpServers"`
}

// LoadMCPConfig reads and decodes .mcp.json from the pack root.
func LoadMCPConfig(root string) (MCPConfig, error) {
	var c MCPConfig
	err := decodeManifest(root, MCPConfigPath, &c)
	return c, err
}

// GitHubSpec is an npm git specifier as written on an `npx` command line:
// `github:<owner>/<repo>#semver:<range>`. npm resolves the range against the
// repository's git tags rather than against the registry, clones the tag it
// picks and runs that tag's `prepare` script to build it — which is how a
// package that is never published to npm is still installable by range.
type GitHubSpec struct {
	// Owner and Repo are the GitHub coordinates the specifier clones from.
	Owner, Repo string
	// Range is whatever followed `#semver:`, unparsed.
	Range string
}

// String renders the specifier the way .mcp.json writes it.
func (g GitHubSpec) String() string {
	return fmt.Sprintf("github:%s/%s#semver:%s", g.Owner, g.Repo, g.Range)
}

// ParseGitHubSpec splits `github:DragonSecurity/buddy-mcp#semver:^2` into its
// owner, its repository and its range.
//
// Every rejection here is a specifier npx would accept and install something
// wrong from, so each one names what it would have installed rather than
// reporting a generic parse failure. The registry form is called out first
// because it is the shape this file used to carry and the shape anyone editing
// it from memory will write again: buddy-mcp is distributed as GitHub releases
// and has never been published to npm, so `buddy-mcp@^2` resolves against a
// registry entry that does not exist and the buddy server simply never starts.
func ParseGitHubSpec(s string) (GitHubSpec, error) {
	body, ok := strings.CutPrefix(s, "github:")
	if !ok {
		return GitHubSpec{}, fmt.Errorf("%q is a registry specifier, want `github:<owner>/<repo>#semver:<range>`; buddy-mcp ships as GitHub releases and is never published to npm, so a registry name resolves to nothing and the server never starts", s)
	}

	path, ref, ok := strings.Cut(body, "#")
	if !ok {
		return GitHubSpec{}, fmt.Errorf("%q carries no `#` ref, want `github:<owner>/<repo>#semver:<range>`; a bare repository specifier installs whatever the default branch holds at the moment npx runs, which is unreleased work at an unpredictable version", s)
	}

	owner, repo, ok := strings.Cut(path, "/")
	if !ok || owner == "" || repo == "" {
		return GitHubSpec{}, fmt.Errorf("%q names %q where an `<owner>/<repo>` pair belongs; npm reads a single segment as a repository under whatever owner it defaults to, which is not this one", s, path)
	}

	rng, ok := strings.CutPrefix(ref, "semver:")
	if !ok {
		return GitHubSpec{}, fmt.Errorf("%q resolves the ref %q, want `semver:<range>`; a branch or commit ref is not a range at all — it either floats with whatever lands on that branch or freezes on one commit, and neither takes the fixes released inside the major this pack is written against", s, ref)
	}

	return GitHubSpec{Owner: owner, Repo: repo, Range: rng}, nil
}

// CaretMajor returns the major version a caret range is pinned to, and whether
// the range is a caret range at all. `^2`, `^2.1` and `^2.1.0` all report 2.
func CaretMajor(rng string) (int, bool) {
	if !strings.HasPrefix(rng, "^") {
		return 0, false
	}
	major, _, _ := strings.Cut(strings.TrimPrefix(rng, "^"), ".")
	n, err := strconv.Atoi(major)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// NPXSpec returns the specifier this server's command asks npx to install,
// unparsed. Flags are skipped, so `npx -y github:DragonSecurity/buddy-mcp#semver:^2`
// resolves to the specifier and not to `-y`. It reports false for a server
// launched any other way, because then there is no range to check at all.
//
// The specifier comes back as written rather than parsed, so that a caller
// holding a specifier this pack does not accept can still name the exact string
// somebody put in the file.
func (s MCPServer) NPXSpec() (string, bool) {
	if s.Command != "npx" {
		return "", false
	}
	for _, arg := range s.Args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg, true
	}
	return "", false
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
