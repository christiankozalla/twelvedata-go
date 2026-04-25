#!/usr/bin/env bash
set -euo pipefail

CHANGELOG_FILE="CHANGELOG.md"
START_MARKER="<!-- AUTO-UNRELEASED:START -->"
END_MARKER="<!-- AUTO-UNRELEASED:END -->"

if [[ ! -f "$CHANGELOG_FILE" ]]; then
  echo "error: $CHANGELOG_FILE not found" >&2
  exit 1
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: not inside a git repository" >&2
  exit 1
fi

latest_tag="$(git tag --list --sort=-v:refname | head -n 1)"
if [[ -z "$latest_tag" ]]; then
  echo "error: no git tags found; create an initial tag before generating changelog" >&2
  exit 1
fi

range="${latest_tag}..HEAD"

# Insert markers once under [Unreleased] if they are missing.
if ! grep -Fq "$START_MARKER" "$CHANGELOG_FILE"; then
  tmp_file="$(mktemp)"
  awk -v start="$START_MARKER" -v end="$END_MARKER" '
    {
      print
      if ($0 ~ /^## \[Unreleased\]/) {
        print ""
        print start
        print end
      }
    }
  ' "$CHANGELOG_FILE" > "$tmp_file"
  mv "$tmp_file" "$CHANGELOG_FILE"
fi

clean_subject() {
  local subject="$1"

  # Drop the conventional commit prefix when present.
  if [[ "$subject" == *": "* ]]; then
    echo "${subject#*: }"
    return
  fi

  echo "$subject"
}

added_items=()
fixed_items=()
changed_items=()
chores_items=()
other_items=()

while IFS= read -r subject; do
  [[ -z "$subject" ]] && continue
  clean="$(clean_subject "$subject")"

  case "$subject" in
    feat* )
      added_items+=("- $clean")
      ;;
    fix* )
      fixed_items+=("- $clean")
      ;;
    refactor*|perf* )
      changed_items+=("- $clean")
      ;;
    docs*|test*|build*|ci*|chore*|style* )
      chores_items+=("- $clean")
      ;;
    * )
      other_items+=("- $clean")
      ;;
  esac
done < <(git log --no-merges --pretty=format:%s "$range")

generated_file="$(mktemp)"
{
  if [[ ${#added_items[@]} -gt 0 ]]; then
    echo "### Added"
    printf '%s\n' "${added_items[@]}"
    echo ""
  fi

  if [[ ${#fixed_items[@]} -gt 0 ]]; then
    echo "### Fixed"
    printf '%s\n' "${fixed_items[@]}"
    echo ""
  fi

  if [[ ${#changed_items[@]} -gt 0 ]]; then
    echo "### Changed"
    printf '%s\n' "${changed_items[@]}"
    echo ""
  fi

  if [[ ${#chores_items[@]} -gt 0 ]]; then
    echo "### Chore"
    printf '%s\n' "${chores_items[@]}"
    echo ""
  fi

  if [[ ${#other_items[@]} -gt 0 ]]; then
    echo "### Other"
    printf '%s\n' "${other_items[@]}"
    echo ""
  fi

  if [[ ${#added_items[@]} -eq 0 && ${#fixed_items[@]} -eq 0 && ${#changed_items[@]} -eq 0 && ${#chores_items[@]} -eq 0 && ${#other_items[@]} -eq 0 ]]; then
    echo "_No unreleased changes._"
  fi
} > "$generated_file"

# Replace marker block with generated content.
updated_file="$(mktemp)"
awk -v start="$START_MARKER" -v end="$END_MARKER" -v generated="$generated_file" '
  BEGIN { in_block = 0 }
  {
    if ($0 == start) {
      print
      while ((getline line < generated) > 0) {
        print line
      }
      close(generated)
      in_block = 1
      next
    }

    if ($0 == end) {
      in_block = 0
      print
      next
    }

    if (!in_block) {
      print
    }
  }
' "$CHANGELOG_FILE" > "$updated_file"
mv "$updated_file" "$CHANGELOG_FILE"

# Keep [Unreleased] compare link anchored to latest tag.
origin_url="$(git remote get-url origin 2>/dev/null || true)"
repo_slug=""
if [[ "$origin_url" =~ github.com[:/]([^/]+/[^/]+)(\.git)?$ ]]; then
  repo_slug="${BASH_REMATCH[1]%.git}"
fi

if [[ -n "$repo_slug" ]]; then
  unreleased_url="https://github.com/${repo_slug}/compare/${latest_tag}...HEAD"
  tmp_link="$(mktemp)"
  awk -v url="$unreleased_url" '
    {
      if ($0 ~ /^\[Unreleased\]: /) {
        print "[Unreleased]: " url
      } else {
        print
      }
    }
  ' "$CHANGELOG_FILE" > "$tmp_link"
  mv "$tmp_link" "$CHANGELOG_FILE"
fi

rm -f "$generated_file"

echo "Updated $CHANGELOG_FILE from commits in $range"
