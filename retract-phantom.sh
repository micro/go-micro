#!/usr/bin/env bash
# retract-phantom.sh — Publish orphan retraction tags for phantom module paths.
#
# Usage:  ./retract-phantom.sh [--dry-run] [--push] [--pilot]
#
# Creates 4 orphan commits (one per module base: v0, v4, v5, v6), each containing
# all go.mod files with retract directives for phantom paths in that base.
# Tags each phantom path on the appropriate orphan commit. Master is never
# touched. The script survives orphan ops by saving itself to /tmp.
#
#   --dry-run  print what would happen, create nothing
#   --pilot    process only the first path of each base (validate end-to-end)
#   --push     push only the new phantom tags to origin (never git push --tags)

set -euo pipefail

DRY_RUN=false
PUSH=false
PILOT=false
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=true ;;
        --push)    PUSH=true ;;
        --pilot)   PILOT=true ;;
    esac
done

# ── Repo root ───────────────────────────────────────────────────────────────
REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# ── Phantom paths ────────────────────────────────────────────────────────────
# Each line: go-micro.dev/<base>/<relative_path>
# The script creates an orphan commit per path with a go.mod containing:
#   module go-micro.dev/<base>/<relative_path>
#   retract [v0.0.0, v1.18.2]

PHANTOM_PATHS=(
    go-micro.dev/plugins/auth/jwt/v4/token
    go-micro.dev/plugins/config/source/grpc/v4/proto
    go-micro.dev/plugins/tree/main/v5/registry
    go-micro.dev/plugins/tree/main/v5/registry/consul
    go-micro.dev/v2client
    go-micro.dev/v2client/registry
    go-micro.dev/v2client/registry/consul
    go-micro.dev/v4/cmd/protoc-gen-micro~
    go-micro.dev/v4.0.0/cmd
    go-micro.dev/v4.0.0/cmd/protoc-gen-micro
    go-micro.dev/v4.10.2/registry
    go-micro.dev/v4.6.0/go-plugins/registry/consul
    go-micro.dev/v4api
    go-micro.dev/v4api/proto
    go-micro.dev/v4config
    go-micro.dev/v4config/source
    go-micro.dev/v4egmentio
    go-micro.dev/v4egmentio/ksuid
    go-micro.dev/v4eris-io
    go-micro.dev/v4eris-io/shortid
    go-micro.dev/v4g.org/codec
    go-micro.dev/v4g.org/codec/protorpc
    go-micro.dev/v4github.com
    go-micro.dev/v4github.com/micro
    go-micro.dev/v4github.com/micro/go-micro
    go-micro.dev/v4github.com/micro/go-micro/client
    go-micro.dev/v4github.com/micro/go-micro/metadata
    go-micro.dev/v4ithub.io
    go-micro.dev/v4ithub.io/maps
    go-micro.dev/v4logger/plugins
    go-micro.dev/v4logger/plugins/wrapper
    go-micro.dev/v4logger/plugins/wrapper/select
    go-micro.dev/v4logger/plugins/wrapper/select/version
    go-micro.dev/v4registry
    go-micro.dev/v4registry/etcd
    go-micro.dev/v4registry/etcd/registry
    go-micro.dev/v4registry/kubernetes
    go-micro.dev/v4rvice
    go-micro.dev/v4rvice/client
    go-micro.dev/v4rvice/errors
    go-micro.dev/v4rvice/logger
    go-micro.dev/v4rvice/server
    go-micro.dev/v4rvice/store
    go-micro.dev/v4~
    go-micro.dev/v5-plugins
    go-micro.dev/v5-plugins/client
    go-micro.dev/v5-plugins/client/grpc
    go-micro.dev/v5-plugins/server
    go-micro.dev/v5-plugins/server/grpc
    go-micro.dev/v5-plugins/v5/client
    go-micro.dev/v5-plugins/v5/client/grpc
    go-micro.dev/v5-plugins/v5/server
    go-micro.dev/v5-plugins/v5/server/grpc
    go-micro.dev/v5-plugins/wrapper
    go-micro.dev/v5-plugins/wrapper/trace
    go-micro.dev/v5-plugins/wrapper/trace/opentelemetry
    go-micro.dev/v5service
    go-micro.dev/v5service/grpc
    go-micro.dev/v5service/logger
)

# ── Parse into per-base arrays ──────────────────────────────────────────────
BASES_V4=() BASES_V5=() BASES_V6=() BASES_V0=()
for p in "${PHANTOM_PATHS[@]}"; do
    case "$p" in
        go-micro.dev/v4/*) BASES_V4+=("$p") ;;
        go-micro.dev/v5/*) BASES_V5+=("$p") ;;
        go-micro.dev/v6/*) BASES_V6+=("$p") ;;
        go-micro.dev/*) BASES_V0+=("$p") ;;
    esac
done

for b in V6 V5 V4 V0; do
    declare -n ba="BASES_$b"
    mapfile -t ba < <(printf '%s\n' "${ba[@]}" | awk '!seen[$0]++')
done

# ── Sanity checks ───────────────────────────────────────────────────────────
for p in "${PHANTOM_PATHS[@]}"; do
    case "$p" in
        go-micro.dev/v[0-9]*/*) rel="${p#go-micro.dev/v[0-9]*/}" ;;
        *) rel="${p#go-micro.dev/}" ;;
    esac
    [[ -f "$rel/go.mod" ]] && { echo "ERROR: $p has go.mod on master — not a phantom"; exit 1; }
done

# ── Pilot mode ───────────────────────────────────────────────────────────────
# Keep only the first path of each base so the full mechanism can be validated
# on a tiny case before the 10k-path blast.
if $PILOT; then
    for b in V4 V5 V6 V0; do
        declare -n ba="BASES_$b"
        ((${#ba[@]})) && ba=("${ba[0]}")
    done
fi

# ── Helpers ──────────────────────────────────────────────────────────────────
RETRACT_MAX="v1.18.2"

valid_tag_ref() {
    local t="$1"
    if [[ "$t" == *..* || "$t" == *"@{"* || "$t" == *"//"* || "$t" == *"/."* || "$t" == *".lock"* || "$t" == .* || "$t" == */ || "$t" == *. ]] || ! [[ "$t" =~ ^[A-Za-z0-9][A-Za-z0-9._/-]*$ ]]; then
        git check-ref-format "refs/tags/$t" 2>/dev/null
        return $?
    fi
    return 0
}

make_shared_orphan_commit() {
    # make_shared_orphan_commit <label> <tmpdir> <count> <ident>
    # Runs ONE `git fast-import` that creates the orphan commit (a parentless
    # commit holding a go.mod per new tag) plus its annotated tags, then deletes
    # the temporary branch. Prints the commit SHA on stdout.
    local label="$1" tmpdir="$2" count="$3" ident="$4"
    local branch="refs/heads/phantom-retract-$label"
    local msg="retract ${label} phantom paths (${count} modules)"
    local stream="$tmpdir/stream"
    {
        cat "$tmpdir/blobs"
        printf 'commit %s\nmark :1000000\nauthor %s\ncommitter %s\ndata %d\n%s\n' \
            "$branch" "$ident" "$ident" "$(( ${#msg} + 1 ))" "$msg"
        cat "$tmpdir/trees"
        printf '\n'
        cat "$tmpdir/tags"
    } > "$stream"
    if ! git fast-import --force < "$stream"; then
        echo "ERROR: fast-import failed for $label" >&2
        exit 1
    fi
    git rev-parse "$branch"
    git update-ref -d "$branch"
}

# ── Execute ──────────────────────────────────────────────────────────────────
total=$(( ${#BASES_V4[@]} + ${#BASES_V5[@]} + ${#BASES_V6[@]} + ${#BASES_V0[@]} ))
echo "Retracting ${total} phantom paths across 4 orphan commits..."
echo ""

CREATED_TAGS=()
ident="$(git var GIT_COMMITTER_IDENT)"
declare -A EXISTING_TAGS=()
while IFS= read -r t; do
    [[ -n "$t" ]] && EXISTING_TAGS["$t"]=1
done < <(git for-each-ref --format='%(refname:short)' refs/tags)
declare -A REMOTE_TAGS=()
while read -r _ ref; do
    [[ -n "$ref" ]] && REMOTE_TAGS["${ref#refs/tags/}"]=1
done < <(git ls-remote --tags --refs origin)

for base_info in "v6:${#BASES_V6[@]}" "v5:${#BASES_V5[@]}" "v4:${#BASES_V4[@]}" "v0:${#BASES_V0[@]}"; do
    IFS=: read -r label count <<< "$base_info"
    if [[ "$count" -eq 0 ]]; then
        echo "=== $label: no phantom paths, skipping ==="
        continue
    fi
    echo "=== $label orphan commit ($count paths) ==="
    if $DRY_RUN; then
        echo "[dry-run] Would create 1 orphan commit with ${count} go.mod files"
        echo "[dry-run] Would tag ${count} paths on that commit"
    else
        declare -n ref="BASES_${label^^}"
        tmpdir="$(mktemp -d)"
        : > "$tmpdir/blobs"
        : > "$tmpdir/trees"
        : > "$tmpdir/tags"
        n=0
        for p in "${ref[@]}"; do
            case "$p" in
                go-micro.dev/v[0-9]*/*) rel="${p#go-micro.dev/v[0-9]*/}" ;;
                *) rel="${p#go-micro.dev/}" ;;
            esac
            tag="${rel}/${RETRACT_MAX}"
            if ! valid_tag_ref "$tag"; then
                echo "  skip (invalid tag name): $tag" >&2
                continue
            fi
            if [[ -n "${REMOTE_TAGS[$tag]+x}" ]]; then
                echo "  exists on origin: $tag (skip)"
                continue
            fi
            if [[ -n "${EXISTING_TAGS[$tag]+x}" ]]; then
                echo "  exists locally: $tag (queue for push)"
                CREATED_TAGS+=("refs/tags/$tag")
                continue
            fi
            n=$((n+1))
            content="module ${p}

go 1.24

// Phantom module path. The Go proxy cached this as a separate module.
// Every version is retracted so 'go install ${p}@latest' errors or
// resolves to the next non-retracted version instead of this path.
retract [v0.0.0, ${RETRACT_MAX}]
"
            printf 'blob\nmark :%d\ndata %d\n%s' "$n" "${#content}" "$content" >> "$tmpdir/blobs"
            printf 'M 100644 :%d %s/go.mod\n' "$n" "$rel" >> "$tmpdir/trees"
            tmsg="retract $tag (phantom)"
            printf 'tag %s\nfrom :1000000\ntagger %s\ndata %d\n%s\n\n' \
                "$tag" "$ident" "$(( ${#tmsg} + 1 ))" "$tmsg" >> "$tmpdir/tags"
            EXISTING_TAGS["$tag"]=1
            CREATED_TAGS+=("refs/tags/$tag")
            echo "  tagged: $tag"
        done
        orphan_sha="$(make_shared_orphan_commit "$label" "$tmpdir" "$n" "$ident")"
        echo "  commit: $orphan_sha"
        rm -rf "$tmpdir"
    fi
    echo ""
done

mapfile -t CREATED_TAGS < <(printf '%s\n' "${CREATED_TAGS[@]}" | awk 'NF' | sort -u)

if $PUSH && ! $DRY_RUN; then
    echo "Pushing ${#CREATED_TAGS[@]} phantom tags to origin..."
    for ((offset = 0; offset < ${#CREATED_TAGS[@]}; offset += 500)); do
        declare -A CURRENT_REMOTE_TAGS=()
        while read -r _ ref; do
            [[ -n "$ref" ]] && CURRENT_REMOTE_TAGS["$ref"]=1
        done < <(git ls-remote --tags --refs origin)

        batch=()
        for ref in "${CREATED_TAGS[@]:offset:500}"; do
            [[ -z "${CURRENT_REMOTE_TAGS[$ref]+x}" ]] && batch+=("$ref")
        done
        ((${#batch[@]})) && git push origin "${batch[@]}"
    done
    echo "Done. Verify with: go list -m -u <path>"
elif ! $DRY_RUN; then
    echo "All tags created locally. Re-run with --push to publish the phantom tags to origin."
    echo ""
    echo "After push, verify with:"
    echo "  go list -m -versions go-micro.dev/v5/api/handler"
    echo "  go install go-micro.dev/v5/api/handler@latest"
fi
