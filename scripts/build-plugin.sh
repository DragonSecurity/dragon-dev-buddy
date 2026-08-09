#!/usr/bin/env bash
# Bundle the pack into dist/dragon-dev-buddy-<version>.plugin, the zip Claude
# accepts as an uploaded plugin.
#
# Only what a plugin loader needs goes in: the manifest, the skills, the hooks,
# the MCP server declaration, the config template and the docs. The Go
# validator, CI config and git metadata stay out — they are how the pack is
# maintained, not part of what it does.
#
# .mcp.json is on the inside of that line even though it is not a skill: every
# skill in the pack talks to the buddy server, and a bundle that declares no MCP
# server installs cleanly and then reports to nothing.
# .claude-plugin/marketplace.json is on the outside of it. A marketplace says
# where this plugin is published and which version is current; it is how the
# plugin is distributed, not part of what it does, and a copy of it riding along
# inside the artifact is a second version string that nothing keeps in step with
# the manifest beside it.
#
# The version is in the filename because the bundle is otherwise
# indistinguishable between releases. Two downloads both called
# dragon-dev-buddy.plugin cannot be told apart once they are sitting in the same
# directory, and the one that gets installed is whichever overwrote the other.
#
# stdout is the artifact path and nothing else, so a caller — the release
# workflow, mainly — can do `artifact="$(./scripts/build-plugin.sh)"` without
# reconstructing the version-stamped name for itself. The human-readable summary
# goes to stderr, where CI still prints it but no script has to parse past it.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

version="$(
	sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
		.claude-plugin/plugin.json | head -1
)"
if [ -z "$version" ]; then
	echo "error: no version in .claude-plugin/plugin.json" >&2
	exit 1
fi

out="dist/dragon-dev-buddy-$version.plugin"
mkdir -p dist
rm -f "$out"

zip -r -q "$out" \
	.claude-plugin \
	skills \
	hooks \
	.mcp.json \
	scripts/pre-commit-memory-guard.sh \
	config.example.json \
	README.md \
	LICENSE \
	-x '*.DS_Store' '.claude-plugin/marketplace.json'

echo "built $out (v$version, $(unzip -l "$out" | tail -1 | awk '{print $2}') files)" >&2
echo "$out"
