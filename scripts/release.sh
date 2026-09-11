#!/usr/bin/env bash
#
# release.sh — Bump la version, génère le changelog et crée le tag localement.
#
# Usage:
#   ./scripts/release.sh [patch|minor|major] [--dry-run]
#
# Exemples:
#   ./scripts/release.sh              # bump patch
#   ./scripts/release.sh minor        # bump minor
#   ./scripts/release.sh major --dry-run   # préview sans rien modifier
#
# Ensuite:
#   git push origin HEAD --tags       # déclenche le workflow release

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

CHANGELOG_FILE="CHANGELOG.md"
CONFIG_FILE="app.config.json"

BUMP_TYPE="patch"
DRY_RUN=false
for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=true ;;
    patch|minor|major) BUMP_TYPE="$arg" ;;
    *) echo "Usage: $0 [patch|minor|major] [--dry-run]" >&2; exit 1 ;;
  esac
done

generate_entry() {
  local new_tag="$1" prev_tag="$2"

  {
    echo "## [$new_tag] - $(date +%Y-%m-%d)"
    echo ""
    echo "### Features"
    git log --pretty=format:"- %s (%h)" --grep="^feat" "${prev_tag}..HEAD" 2>/dev/null || echo "- No features"
    echo ""
    echo ""
    echo "### Bug Fixes"
    git log --pretty=format:"- %s (%h)" --grep="^fix" "${prev_tag}..HEAD" 2>/dev/null || echo "- No bug fixes"
    echo ""
    echo ""
    echo "### Other Changes"
    git log --pretty=format:"- %s (%h)" --grep="^chore\|^refactor\|^perf\|^style\|^ci\|^build" "${prev_tag}..HEAD" 2>/dev/null || echo "- Maintenance updates"
    echo ""
  }
}

if ! git diff-index --quiet HEAD --; then
  echo "Error: working tree is not clean. Commit or stash your changes first." >&2
  exit 1
fi

prev_tag="$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")"
current="${prev_tag#v}"

IFS='.' read -r major minor patch <<< "$current"
case "$BUMP_TYPE" in
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  patch)
    patch=$((patch + 1))
    ;;
esac

new="$major.$minor.$patch"
new_tag="v$new"

echo "Release: $prev_tag -> $new_tag ($BUMP_TYPE)"

if [ "$DRY_RUN" = true ]; then
  echo ""
  echo "=== DRY RUN ==="
  echo "Would bump: $prev_tag -> $new_tag"
  echo ""
  echo "Changelog entry:"
  generate_entry "$new_tag" "$prev_tag"
  exit 0
fi

entry="$(generate_entry "$new_tag" "$prev_tag")"

if [ ! -f "$CHANGELOG_FILE" ]; then
  {
    echo "# Changelog"
    echo ""
    echo "All notable changes to this project will be documented in this file."
    echo "The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)."
    echo ""
  } > "$CHANGELOG_FILE"
fi

head -4 "$CHANGELOG_FILE" > "$CHANGELOG_FILE.tmp"
{
  echo ""
  echo "$entry"
  echo ""
  tail -n +5 "$CHANGELOG_FILE"
} >> "$CHANGELOG_FILE.tmp"
mv "$CHANGELOG_FILE.tmp" "$CHANGELOG_FILE"

echo "Updated $CHANGELOG_FILE"

if [ -f "$CONFIG_FILE" ]; then
  jq --arg v "$new" '.version = $v' "$CONFIG_FILE" > "$CONFIG_FILE.tmp"
  mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"
  echo "Updated $CONFIG_FILE version to $new"
fi

git add "$CHANGELOG_FILE" "$CONFIG_FILE"
git commit -m "chore: release v${new}"
git tag -a "$new_tag" -m "Release $new_tag"

echo ""
echo "Done: $new_tag created locally."
echo ""
echo "Push to trigger release:"
echo "  git push origin HEAD --tags"