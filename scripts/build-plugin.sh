#!/usr/bin/env bash
# Bundle the pack into dist/dragon-dev-buddy.plugin, the zip Claude accepts as an
# uploaded plugin.
#
# Only what a plugin loader needs goes in: the manifest, the skills, the hooks,
# the config template and the docs. The Go validator, CI config and git metadata
# stay out — they are how the pack is maintained, not part of what it does.
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

out="dist/dragon-dev-buddy.plugin"
mkdir -p dist
rm -f "$out"

zip -r -q "$out" \
	.claude-plugin \
	skills \
	hooks \
	config.example.json \
	README.md \
	LICENSE \
	-x '*.DS_Store'

echo "built $out (v$version, $(unzip -l "$out" | tail -1 | awk '{print $2}') files)"
