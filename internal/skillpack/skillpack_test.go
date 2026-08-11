package skillpack_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

func loadPlugin(t *testing.T) skillpack.Plugin {
	t.Helper()
	p, err := skillpack.LoadPlugin(root)
	if err != nil {
		t.Fatalf("loading manifest: %v", err)
	}
	return p
}

func loadReleases(t *testing.T) []skillpack.Release {
	t.Helper()
	releases, err := skillpack.ChangelogVersions(root)
	if err != nil {
		t.Fatalf("reading changelog: %v", err)
	}
	if len(releases) == 0 {
		t.Fatal("CHANGELOG.md names no released version; every shipped version is written down there")
	}
	return releases
}

// TestVersionIsSemver parses the manifest version instead of eyeballing its
// shape, because every check below compares versions numerically. A version that
// does not parse would not fail those checks — it would skip them.
func TestVersionIsSemver(t *testing.T) {
	v, err := skillpack.ParseVersion(loadPlugin(t).Version)
	if err != nil {
		t.Fatalf("%s: %v", ".claude-plugin/plugin.json", err)
	}
	if (v == skillpack.Version{}) {
		t.Error("manifest version is 0.0.0, which is the placeholder nobody replaced rather than a release")
	}
}

// TestVersionCompareOrdersNumerically exists because the obvious implementation
// of every ordering check in this file is a string comparison, and a string
// comparison is right for eleven versions and wrong for the twelfth: "1.10.0"
// sorts below "1.9.0". The file-based tests below cannot catch that — they would
// pass today and fail on the release after 1.9.0, months later, reading as a
// changelog mistake rather than a comparison bug.
func TestVersionCompareOrdersNumerically(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"1.9.0", "1.10.0", -1},
		{"1.10.0", "1.9.0", 1},
		{"1.1.0", "1.1.0", 0},
		{"2.0.0", "1.99.99", 1},
		{"1.1.2", "1.1.10", -1},
		{"0.9.0", "1.0.0", -1},
	} {
		a, err := skillpack.ParseVersion(tc.a)
		if err != nil {
			t.Fatalf("parsing %q: %v", tc.a, err)
		}
		b, err := skillpack.ParseVersion(tc.b)
		if err != nil {
			t.Fatalf("parsing %q: %v", tc.b, err)
		}
		if got := a.Compare(b); got != tc.want {
			t.Errorf("%s compared to %s = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}

	for _, bad := range []string{"1.1", "v1.1.0", "1.1.0-rc.1", "01.1.0", "", "latest"} {
		if v, err := skillpack.ParseVersion(bad); err == nil {
			t.Errorf("ParseVersion(%q) accepted the version as %s; the manifests only ever carry plain releases", bad, v)
		}
	}
}

// TestParseGitHubSpecRejectsEveryOtherForm guards the .mcp.json check below by
// pinning what the parser refuses. Each entry here is a specifier npx would
// happily accept and install *something* from, so a lenient parser would not
// make the check fail — it would make the check pass while installing the wrong
// thing, which is the failure mode that costs a day to find.
//
// The registry form leads the list because it is the one this repository
// actually shipped: `.mcp.json` declared `buddy-mcp@^2` for a package that is
// distributed as GitHub releases and was never published to npm, so the pack
// advertised a companion server that could not start on any machine but the
// author's, where it was already installed by path.
func TestParseGitHubSpecRejectsEveryOtherForm(t *testing.T) {
	for _, tc := range []struct {
		spec             string
		owner, repo, rng string
	}{
		{"github:DragonSecurity/buddy-mcp#semver:^2", "DragonSecurity", "buddy-mcp", "^2"},
		{"github:DragonSecurity/buddy-mcp#semver:^2.1.0", "DragonSecurity", "buddy-mcp", "^2.1.0"},
		{"github:someone-else/buddy-mcp#semver:^2", "someone-else", "buddy-mcp", "^2"},
	} {
		got, err := skillpack.ParseGitHubSpec(tc.spec)
		if err != nil {
			t.Errorf("ParseGitHubSpec(%q): %v", tc.spec, err)
			continue
		}
		if got.Owner != tc.owner || got.Repo != tc.repo || got.Range != tc.rng {
			t.Errorf("ParseGitHubSpec(%q) = %s/%s at %q, want %s/%s at %q", tc.spec, got.Owner, got.Repo, got.Range, tc.owner, tc.repo, tc.rng)
		}
	}

	for _, tc := range []struct {
		spec string
		why  string
	}{
		{"buddy-mcp@^2", "a registry range for a package that is not on the registry"},
		{"buddy-mcp", "a bare registry name"},
		{"@dragonsecurity/buddy-mcp@^2", "a scoped registry name"},
		{"github:DragonSecurity/buddy-mcp", "no ref at all, so npm takes the default branch"},
		{"github:DragonSecurity/buddy-mcp#main", "a branch ref, which is not a range"},
		{"github:DragonSecurity/buddy-mcp#v2.1.0", "a tag ref, which freezes on one release"},
		{"github:DragonSecurity/buddy-mcp#semver^2", "a `#semver:` prefix with the colon dropped"},
		{"github:buddy-mcp#semver:^2", "a repository with no owner"},
		{"", "an empty specifier"},
	} {
		if got, err := skillpack.ParseGitHubSpec(tc.spec); err == nil {
			t.Errorf("ParseGitHubSpec(%q) accepted %s as %s", tc.spec, tc.why, got)
		}
	}
}

// TestCaretMajorRejectsFloatingRanges guards the check below it. If CaretMajor
// were lenient — reporting a major for "latest" or "*" — the .mcp.json test
// would still pass while asserting nothing, which is the worse of the two
// failures because it looks like coverage.
func TestCaretMajorRejectsFloatingRanges(t *testing.T) {
	if major, ok := skillpack.CaretMajor("^2.1"); !ok || major != 2 {
		t.Errorf("CaretMajor(\"^2.1\") = %d, %v; want 2, true", major, ok)
	}
	for _, floating := range []string{"latest", "*", "", "2", ">=2", "~2.1.0", "^"} {
		if major, ok := skillpack.CaretMajor(floating); ok {
			t.Errorf("CaretMajor(%q) reported major %d; a range that is not a caret range pins nothing", floating, major)
		}
	}
}

// TestManifestVersionIsNewestRelease is the drift the whole exercise exists to
// stop. The pack shipped four substantive changes — plugin hooks, PR and batch
// review, the can-this-land gate, the inbound dependency mode — while
// plugin.json sat at 1.0.0, and nothing anywhere noticed. Nobody is offered an
// update they are not told about: Claude Code compares the published version
// against the installed one, and the copy installed on the machine that wrote
// those changes is still recorded as 1.0.0. Cutting a changelog section is the
// step people remember; bumping the manifest is the one they forget.
//
// The newest release is found by comparing versions rather than by taking the
// first heading, so this test says what it means even while the changelog is out
// of order — keeping the ordering failure in the test that is about ordering.
func TestManifestVersionIsNewestRelease(t *testing.T) {
	p := loadPlugin(t)
	manifest, err := skillpack.ParseVersion(p.Version)
	if err != nil {
		t.Fatalf(".claude-plugin/plugin.json: %v", err)
	}

	newest := loadReleases(t)[0]
	for _, r := range loadReleases(t) {
		if r.Version.Compare(newest.Version) > 0 {
			newest = r
		}
	}

	if manifest.Compare(newest.Version) != 0 {
		t.Errorf("plugin.json version is %s but the newest released CHANGELOG.md heading is %s (line %d); bump the manifest in the same commit that cuts the section, or the release is invisible to everyone who already installed the pack",
			manifest, newest.Version, newest.Line)
	}
}

// TestChangelogReleasesDescend keeps the changelog readable in the one way that
// matters: the top entry is the current release. A section inserted in the wrong
// place, or a version reused for a second set of notes, makes "the newest
// heading" a different answer from "the highest version" — and every human who
// reads this file reads only the top of it.
func TestChangelogReleasesDescend(t *testing.T) {
	releases := loadReleases(t)

	seen := map[string]int{}
	for _, r := range releases {
		if first, dup := seen[r.Version.String()]; dup {
			t.Errorf("CHANGELOG.md:%d: %s appears again, first written at line %d; one version, one section", r.Line, r.Version, first)
			continue
		}
		seen[r.Version.String()] = r.Line
	}

	for i := 1; i < len(releases); i++ {
		prev, curr := releases[i-1], releases[i]
		if prev.Version.Compare(curr.Version) <= 0 {
			t.Errorf("CHANGELOG.md:%d: %s is listed below %s (line %d); releases run newest first",
				curr.Line, curr.Version, prev.Version, prev.Line)
		}
	}
}

// TestMarketplaceEntryMatchesManifest checks the second copy of the pack's
// identity instead of trusting it. The marketplace entry is what a user reads
// before installing and what auto-update compares against; plugin.json is what
// they get. Nothing reconciles the two, so a bump applied to one of them
// advertises a version that does not match its contents — and this entry used to
// live outside the repository entirely, where it had already drifted.
func TestMarketplaceEntryMatchesManifest(t *testing.T) {
	p := loadPlugin(t)

	m, err := skillpack.LoadMarketplace(root)
	if err != nil {
		t.Fatalf("loading marketplace: %v", err)
	}

	entry, ok := m.Entry(p.Name)
	if !ok {
		t.Fatalf(".claude-plugin/marketplace.json lists no plugin named %q; the marketplace this repository publishes about itself has to list itself", p.Name)
	}

	for _, f := range []struct {
		field           string
		entry, manifest string
	}{
		{"version", entry.Version, p.Version},
		{"description", entry.Description, p.Description},
		{"author.name", entry.Author.Name, p.Author.Name},
	} {
		if f.entry != f.manifest {
			t.Errorf("marketplace.json %s = %q, plugin.json says %q", f.field, f.entry, f.manifest)
		}
	}
}

// TestMarketplaceSourceResolvesToThisPack checks that the entry points at the
// plugin sitting next to it. `source` is the only field that is not a copy of
// plugin.json, which is exactly why nothing else here would catch it being
// wrong: a self-listing marketplace whose source has drifted installs a
// different pack than the one whose version and description it advertises, and
// the install still succeeds.
func TestMarketplaceSourceResolvesToThisPack(t *testing.T) {
	p := loadPlugin(t)

	m, err := skillpack.LoadMarketplace(root)
	if err != nil {
		t.Fatalf("loading marketplace: %v", err)
	}
	entry, ok := m.Entry(p.Name)
	if !ok {
		t.Fatalf(".claude-plugin/marketplace.json lists no plugin named %q", p.Name)
	}

	if entry.Source == "" {
		t.Fatal("marketplace entry has no source; Claude Code would not know what to install")
	}
	if !strings.HasPrefix(entry.Source, ".") {
		t.Fatalf("marketplace source = %q, want a path relative to the repository root; this marketplace exists to publish the pack it ships with, and a remote source would publish somebody else's copy of it", entry.Source)
	}

	resolved, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(entry.Source)))
	if err != nil {
		t.Fatalf("resolving marketplace source: %v", err)
	}
	packRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolving pack root: %v", err)
	}
	if resolved != packRoot {
		t.Errorf("marketplace source %q resolves to %s, want the repository root %s", entry.Source, resolved, packRoot)
	}

	// And the plugin it names is really there — the check that would still fail
	// if the source pointed at a sibling directory that resolves fine and holds
	// no plugin.
	installed, err := skillpack.LoadPlugin(resolved)
	if err != nil {
		t.Fatalf("marketplace source %q has no readable plugin manifest: %v", entry.Source, err)
	}
	if installed.Name != entry.Name {
		t.Errorf("marketplace source %q holds the plugin %q, but the entry advertises %q", entry.Source, installed.Name, entry.Name)
	}
}

// buddyMajorRef matches the buddy-mcp major the documentation commits to,
// written as `buddy-mcp v2` or `buddy-mcp v2+`.
var buddyMajorRef = regexp.MustCompile(`buddy-mcp v(\d+)`)

// buddyMajorTheSkillsExpect reads the major out of the pack's own prose. The
// skills describe a specific tool surface — `buddy_advise` exists from v2, the
// figures in buddy-operations.md are v2 figures — so the range .mcp.json
// installs has to agree with the docs, and the docs are the side a reader
// trusts.
func buddyMajorTheSkillsExpect(t *testing.T) int {
	t.Helper()

	found := map[int]string{}
	err := filepath.WalkDir(filepath.Join(root, "skills"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(raw), "\n") {
			for _, m := range buddyMajorRef.FindAllStringSubmatch(line, -1) {
				major, convErr := strconv.Atoi(m[1])
				if convErr != nil {
					return convErr
				}
				if _, seen := found[major]; !seen {
					found[major] = fmt.Sprintf("%s:%d", path, i+1)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scanning skills for the expected buddy-mcp major: %v", err)
	}

	switch len(found) {
	case 0:
		t.Fatal("no skill document names a buddy-mcp major version, so there is nothing to check .mcp.json against; say which major the tool surface described here belongs to")
	case 1:
		for major := range found {
			return major
		}
	}

	var where []string
	for major, at := range found {
		where = append(where, fmt.Sprintf("v%d (%s)", major, at))
	}
	sort.Strings(where)
	t.Fatalf("the skills name more than one buddy-mcp major: %s; the docs have to agree before .mcp.json can be checked against them", strings.Join(where, ", "))
	return 0
}

// TestMCPConfigDeclaresBuddyServer pins the dependency the whole pack rests on.
// Every skill ends by calling `buddy_observe`, and before this file existed the
// server behind that call was wired by absolute path in one person's global
// config — so the pack looked self-contained and was not installable by anyone
// else. The declaration only helps if it stays true.
func TestMCPConfigDeclaresBuddyServer(t *testing.T) {
	cfg, err := skillpack.LoadMCPConfig(root)
	if err != nil {
		t.Fatalf("loading .mcp.json: %v", err)
	}

	// The map key is the server's name, and it is what the tool names in every
	// skill are namespaced by. Rename it and `buddy_observe` addresses nothing.
	server, ok := cfg.Servers["buddy"]
	if !ok {
		var names []string
		for name := range cfg.Servers {
			names = append(names, name)
		}
		sort.Strings(names)
		t.Fatalf(".mcp.json declares no server named \"buddy\" (found: %s); the skills call `buddy_*` tools under that name", strings.Join(names, ", "))
	}

	raw, ok := server.NPXSpec()
	if !ok {
		t.Fatalf(".mcp.json runs the buddy server as %q, not through npx; the pack ships a resolvable specifier so that installing the pack installs the server", server.Command)
	}

	// The specifier has to be the git form. buddy-mcp is distributed as GitHub
	// releases and is never published to the npm registry, so `buddy-mcp@^2` —
	// the form this file used to carry — asks npm for a package that is not
	// there. npx reports a 404 and the buddy server never starts, which surfaces
	// to a user as every `buddy_*` tool being absent rather than as an install
	// failure they can read.
	spec, err := skillpack.ParseGitHubSpec(raw)
	if err != nil {
		t.Fatalf(".mcp.json: %v", err)
	}
	if spec.Owner != buddyOwner || spec.Repo != buddyRepo {
		t.Errorf(".mcp.json installs %s/%s; the pack targets %s/%s, and the other companion projects answer to a different tool surface entirely", spec.Owner, spec.Repo, buddyOwner, buddyRepo)
	}

	// A caret range and not a floating one. A branch ref, `*` or no ref at all
	// resolves to whatever that repository holds most recently, which means the
	// next major — the release that is allowed to change the shape of the buddy
	// tools — arrives on its own, into a pack whose skills still document the old
	// surface, and it arrives at a different time for every user because npx
	// resolves at launch. Every skill here calls `buddy_advise` and
	// `buddy_observe` with a specific argument shape; a major that changes that
	// shape turns each of those calls into an error the model then has to work
	// around mid-task. A caret takes the fixes inside the major and refuses the
	// break; the break becomes a deliberate edit to this file, made alongside the
	// docs that describe the new surface.
	major, ok := skillpack.CaretMajor(spec.Range)
	if !ok {
		t.Fatalf(".mcp.json pins %s to %q, want a caret range like \"^2\"; a floating range upgrades across a major on its own, at a different moment for every user", spec.Repo, spec.Range)
	}
	if want := buddyMajorTheSkillsExpect(t); major != want {
		t.Errorf(".mcp.json installs %s ^%d but the skills document v%d; one of the two is describing a tool surface that will not be there", spec.Repo, major, want)
	}

	// What this range resolves against lives outside the repository, and no test
	// here can reach it: npm matches `#semver:^2` against the *git tags* of
	// DragonSecurity/buddy-mcp, clones the highest tag inside the range, and runs
	// that tag's `prepare` script to build it. So the assertions above are only
	// half the contract. The other half is that buddy-mcp keeps pushing `vN.N.N`
	// tags for every release and keeps its build wired to `prepare` rather than
	// `prepublishOnly`, which never runs for a consumer installing from git. If a
	// release ships with no tag, this file still reads as correct and npx resolves
	// to the previous one; if the build moves off `prepare`, npx resolves the
	// right tag and installs a package with no compiled output. Neither shows up
	// here, and both are why this comment is here instead of a network call.
}

// buddyOwner and buddyRepo are the GitHub coordinates .mcp.json installs the
// companion server from. They are the distribution channel: the pack does not
// resolve buddy-mcp by name against any registry, it clones a tag out of this
// repository.
const (
	buddyOwner = "DragonSecurity"
	buddyRepo  = "buddy-mcp"
)

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

// TestObserveGateMatcherCoversTheRealToolName pins the PostToolUse matcher
// against the name the buddy's observe tool actually has at runtime.
//
// The gate marks a session dirty on an edit and clears it when buddy_observe
// runs. The clear never fired: the matcher read "buddy_observe", but an MCP tool
// is addressed as "mcp__<server>__<tool>", so nothing ever matched and every
// turn that touched a file blocked on Stop whether or not it had recorded. The
// hook itself was correct throughout — it was never invoked — which is exactly
// the failure a matcher typo produces, and it is silent by construction.
func TestObserveGateMatcherCoversTheRealToolName(t *testing.T) {
	raw := readFile(t, "hooks/hooks.json")

	var cfg struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("hooks/hooks.json does not parse: %v", err)
	}

	// The names Claude Code passes as tool_name. The qualified form is the one
	// that regressed; the bare edit tools are here so a fix for it cannot break
	// the marking half.
	toolNames := []string{
		// Both installs. The first is a hand-registered server, the second is what
		// this pack's own .mcp.json produces — which is the name in use the moment
		// anyone follows the documented install, so a matcher that covers only the
		// first is broken for exactly the users who did as they were told.
		"mcp__buddy__buddy_observe",
		"mcp__plugin_dragon-dev-buddy_buddy__buddy_observe",
		"Edit",
		"Write",
		"MultiEdit",
		"NotebookEdit",
	}

	entries := cfg.Hooks["PostToolUse"]
	if len(entries) == 0 {
		t.Fatal("hooks/hooks.json declares no PostToolUse hook, so the gate can never clear")
	}

	for _, name := range toolNames {
		matched := false
		for _, e := range entries {
			// Anchored, because the client matches the whole tool name rather
			// than a substring of it. That is the entire bug: "buddy_observe"
			// occurs inside "mcp__buddy__buddy_observe", so it looks correct to
			// any substring check — including Go's own MatchString — and a test
			// written the obvious way passes against the matcher that shipped
			// broken. The anchors are what make this test able to fail.
			re, err := regexp.Compile(`^(?:` + e.Matcher + `)$`)
			if err != nil {
				t.Fatalf("matcher %q does not compile: %v", e.Matcher, err)
			}
			if re.MatchString(name) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("no PostToolUse matcher matches tool_name %q; the gate cannot see it", name)
		}
	}
}

// TestMemoriesAreIgnored pins the one guarantee project-memory makes to the
// user: what it writes stays on their machine.
//
// The pack's own .gitignore already excludes all of .dragon-buddy/, but that is
// not the rule being asserted — buddy-mcp commits its .dragon-buddy config and
// reports, and any project may reasonably do the same. The memories directory
// has to be excluded on its own account, or it rides along with them into a
// public repository the first time someone decides their config is worth
// committing.
func TestMemoriesAreIgnored(t *testing.T) {
	ignore := readFile(t, ".gitignore")
	if !strings.Contains(ignore, ".dragon-buddy/memories/") {
		t.Error(".gitignore has no rule naming .dragon-buddy/memories/ specifically; " +
			"a project that commits the rest of .dragon-buddy would publish its memories")
	}

	// The skill promises this script exists and tells the user to symlink it as
	// a pre-commit hook. A promise to a path is a promise the tests can keep.
	guard := filepath.Join(root, "scripts", "pre-commit-memory-guard.sh")
	info, err := os.Stat(guard)
	if err != nil {
		t.Fatalf("scripts/pre-commit-memory-guard.sh is missing, but project-memory tells the user to install it: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("pre-commit-memory-guard.sh is not executable, so symlinking it as a git hook does nothing")
	}
}

// TestSessionStartHooksAreRegistered pins every hook script the pack ships to an
// entry in hooks.json.
//
// A hook that exists and is not registered is the most expensive shape of dead
// code here: it is fully written, it looks installed, and it silently never
// runs. That is exactly how the observe gate's clear half went missing for a
// whole release cycle.
func TestSessionStartHooksAreRegistered(t *testing.T) {
	raw := readFile(t, "hooks/hooks.json")

	var cfg struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("hooks/hooks.json does not parse: %v", err)
	}

	registered := map[string]bool{}
	for _, entries := range cfg.Hooks {
		for _, e := range entries {
			for _, h := range e.Hooks {
				for _, script := range scriptNames(h.Command) {
					registered[script] = true
				}
			}
		}
	}

	files, err := os.ReadDir(filepath.Join(root, "hooks"))
	if err != nil {
		t.Fatalf("reading hooks/: %v", err)
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".mjs") {
			continue
		}
		if !registered[f.Name()] {
			t.Errorf("hooks/%s is never referenced by hooks.json, so it can never run", f.Name())
		}
	}
}

// scriptNames pulls the *.mjs basenames out of a hook command line.
func scriptNames(command string) []string {
	var out []string
	for _, field := range strings.FieldsFunc(command, func(r rune) bool {
		return r == '/' || r == '"' || r == ' '
	}) {
		if strings.HasSuffix(field, ".mjs") {
			out = append(out, field)
		}
	}
	return out
}

// pluginRootRef matches a path a skill tells the user to reach through
// ${CLAUDE_PLUGIN_ROOT} — the installed pack's directory. It is the only way a
// skill can name a file that is not one of its own sidecars.
var pluginRootRef = regexp.MustCompile(`\$\{CLAUDE_PLUGIN_ROOT\}/([A-Za-z0-9._/-]+)`)

// TestBundledScriptsExist is the check that was missing when git-guardrails and
// runbook-wizard shipped in 1.4.0.
//
// Both told the user to copy a script out of ${CLAUDE_PLUGIN_ROOT}/scripts/, and
// neither script was in the list build-plugin.sh zips. In this checkout the file
// is right there and every install step appears to work; from a marketplace
// install the copy fails, the hook is never written, and the only symptom is a
// shell error the user has to be reading for. A guard that was never installed
// is the one failure mode the guard skill exists to prevent, so it is worth a
// test that reads the bundle list rather than the working tree.
//
// Both halves are checked: the referenced file exists at all, and the bundle
// carries it. The first alone passes in this repository forever.
func TestBundledScriptsExist(t *testing.T) {
	build := readFile(t, "scripts/build-plugin.sh")

	// The zip invocation is the authority on what an install receives. Reading
	// it as text keeps one list rather than a second copy that drifts from it.
	zipArgs, _, found := strings.Cut(build, "-x '*.DS_Store'")
	if !found {
		t.Fatal("scripts/build-plugin.sh: cannot find the zip argument list; this test reads it to learn what ships")
	}
	if _, after, ok := strings.Cut(zipArgs, "zip -r -q \"$out\""); ok {
		zipArgs = after
	}

	bundled := func(path string) bool {
		for _, line := range strings.Split(zipArgs, "\n") {
			entry := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "\\"))
			if entry == "" {
				continue
			}
			// A bundled directory carries everything under it.
			if entry == path || strings.HasPrefix(path, entry+"/") {
				return true
			}
		}
		return false
	}

	for _, s := range loadSkills(t) {
		for _, m := range pluginRootRef.FindAllStringSubmatch(s.Body, -1) {
			ref := m[1]
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(ref))); err != nil {
				t.Errorf("%s: points at ${CLAUDE_PLUGIN_ROOT}/%s, which does not exist in the pack", s.Path(), ref)
				continue
			}
			if !bundled(ref) {
				t.Errorf("%s: points at ${CLAUDE_PLUGIN_ROOT}/%s, which scripts/build-plugin.sh does not zip; the file is present here but absent from every install, so the step fails for everyone but you", s.Path(), ref)
			}
		}
	}
}

// TestNoticesShipWithTheBundle keeps the pack's MIT obligation attached to the
// material it covers. Six skills are derived from MIT-licensed work, and the
// licence requires its notice to travel with substantial portions — the copy in
// this repository does nothing for someone holding only the installed plugin.
func TestNoticesShipWithTheBundle(t *testing.T) {
	build := readFile(t, "scripts/build-plugin.sh")
	if !strings.Contains(build, "THIRD-PARTY-NOTICES.md") {
		t.Error("scripts/build-plugin.sh does not zip THIRD-PARTY-NOTICES.md; the bundle ships MIT-derived skills without the notice that licence requires to travel with them")
	}
	notices := readFile(t, "THIRD-PARTY-NOTICES.md")
	if !strings.Contains(notices, "MIT License") {
		t.Error("THIRD-PARTY-NOTICES.md no longer reproduces the MIT licence text")
	}
}
