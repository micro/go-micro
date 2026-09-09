#!/usr/bin/env bash
# retract-tilde.sh — Publish root retraction tags for the two phantom paths
# whose names contain `~`, which retract-phantom.sh must skip (git rejects `~`
# in ref names). Each retraction is published as a root tag instead:
#
#   go-micro.dev/v4/cmd/protoc-gen-micro~  →  root tag v1.18.1
#   go-micro.dev/v4~                       →  root tag v1.18.2
#
# Both paths are keyed to root tags (their cached versions are the repo's v1.x
# root tags), so a root tag is the natural home for the retraction.
#
# Usage:  ./retract-tilde.sh [--dry-run] [--push]
#
#   --dry-run  print what would happen, create nothing
#   --push     push the new root tags to origin (never git push --tags)

set -euo pipefail

DRY_RUN=false
PUSH=false
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=true ;;
        --push)    PUSH=true ;;
    esac
done

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# ── Tilde paths ───────────────────────────────────────────────────────────────
# Each entry: <module path>:<root tag>
# The natural retraction tag (<path>/v1.18.2) is an invalid git ref name, so
# each path gets its own orphan commit whose go.mod declares the module path
# and retracts every cached version, tagged with the root tag.
TILDE_PATHS=(
    "go-micro.dev/v4/cmd/protoc-gen-micro~:v1.18.1"
    "go-micro.dev/v4~:v1.18.2"
)

RETRACT_MAX="v1.18.0"

# ── Existing tags ─────────────────────────────────────────────────────────────
ident="$(git var GIT_COMMITTER_IDENT)"
declare -A EXISTING_TAGS=()
while IFS= read -r t; do
    [[ -n "$t" ]] && EXISTING_TAGS["$t"]=1
done < <(git for-each-ref --format='%(refname:short)' refs/tags)
declare -A REMOTE_TAGS=()
while read -r _ ref; do
    [[ -n "$ref" ]] && REMOTE_TAGS["${ref#refs/tags/}"]=1
done < <(git ls-remote --tags --refs origin 2>/dev/null || true)

# ── Helper ────────────────────────────────────────────────────────────────────
make_orphan_commit() {
    # make_orphan_commit <label> <content> <tag>
    # Runs ONE `git fast-import` that creates a parentless commit holding the
    # go.mod, tags it, then deletes the temporary branch. Prints the SHA.
    local label="$1" content="$2" tag="$3"
    local branch="refs/heads/phantom-retract-$tag"
    local msg="retract $label (phantom, tilde)"
    local bytes
    bytes="$(printf %s "$content" | wc -c | tr -d ' ')"
    local tmpdir="$(mktemp -d)"
    local stream="$tmpdir/stream"
    {
        printf 'blob\nmark :1\ndata %d\n%s' "$bytes" "$content"
        printf 'commit %s\nmark :1000000\nauthor %s\ncommitter %s\ndata %d\n%s\n' \
            "$branch" "$ident" "$ident" "$(( ${#msg} + 1 ))" "$msg"
        printf 'M 100644 :1 go.mod\n'
        printf '\n'
        printf 'tag %s\nfrom :1000000\ntagger %s\ndata %d\n%s\n\n' \
            "$tag" "$ident" "$(( ${#msg} + 1 ))" "$msg"
    } > "$stream"
    if ! git fast-import --force < "$stream"; then
        echo "ERROR: fast-import failed for $tag" >&2
        exit 1
    fi
    local sha
    sha="$(git rev-parse "$branch")"
    git update-ref -d "$branch"
    rm -rf "$tmpdir"
    echo "$sha"
}

# ── Execute ───────────────────────────────────────────────────────────────────
CREATED_TAGS=()
echo "Retracting ${#TILDE_PATHS[@]} tilde phantom paths as root tags..."
for entry in "${TILDE_PATHS[@]}"; do
    IFS=: read -r path tag <<< "$entry"
    if [[ -n "${REMOTE_TAGS[$tag]+x}" ]]; then
        echo "  exists on origin: $tag (skip)"
        continue
    fi
    if [[ -n "${EXISTING_TAGS[$tag]+x}" ]]; then
        echo "  exists locally: $tag (queue for push)"
        CREATED_TAGS+=("refs/tags/$tag")
        continue
    fi
    if $DRY_RUN; then
        echo "  [dry-run] would tag $tag with retract for $path"
        continue
    fi
    content="module ${path}

go 1.24

// Phantom path (tilde). The natural retraction tag $(basename "$path")/v1.18.2
// is an invalid git ref name, so this retraction is published as the root tag
// ${tag} instead — which is how this path is already keyed (its versions are
// the repo's v1.x root tags).
retract [v0.0.0, ${RETRACT_MAX}]
"
    sha="$(make_orphan_commit "$path" "$content" "$tag")"
    EXISTING_TAGS["$tag"]=1
    CREATED_TAGS+=("refs/tags/$tag")
    echo "  tagged: $tag (commit $sha)"
done

# ── Push ──────────────────────────────────────────────────────────────────────
if $PUSH && ! $DRY_RUN; then
    if ((${#CREATED_TAGS[@]})); then
        echo "Pushing ${#CREATED_TAGS[@]} root retraction tags to origin..."
        git push origin "${CREATED_TAGS[@]}"
    fi
    echo "Done. Verify with:"
    echo "  go list -m -versions go-micro.dev/v4/cmd/protoc-gen-micro~"
    echo "  go list -m -versions go-micro.dev/v4~"
elif ! $DRY_RUN; then
    echo ""
    echo "Tags created locally. Re-run with --push to publish to origin."
    echo ""
    echo "After push, verify with:"
    echo "  go list -m -versions go-micro.dev/v4/cmd/protoc-gen-micro~"
    echo "  go list -m -versions go-micro.dev/v4~"
fi
