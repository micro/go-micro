#!/usr/bin/env bash
# check-versions.sh — Run `go list -m -versions` for every phantom module path
# declared in retract-phantom.sh.
#
# Usage:  ./check-versions.sh [--dry-run]
#
#   --dry-run  print the paths only, without running go list

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE="$SCRIPT_DIR/retract-phantom.sh"

DRY_RUN=false
for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=true ;;
    esac
done

if [[ ! -f "$SOURCE" ]]; then
    echo "error: $SOURCE not found" >&2
    exit 1
fi

mapfile -t PATHS < <(
    awk '
        /^PHANTOM_PATHS=\(/ { in_arr = 1; next }
        in_arr && /^\)/     { exit }
        in_arr {
            gsub(/^[[:space:]]+|[[:space:]]+$/, "")
            if ($0 != "" && $0 !~ /^#/) print
        }
    ' "$SOURCE"
)

if [[ ${#PATHS[@]} -eq 0 ]]; then
    echo "error: no phantom paths parsed from $SOURCE" >&2
    exit 1
fi

echo "checking ${#PATHS[@]} phantom module paths"
for path in "${PATHS[@]}"; do
    if $DRY_RUN; then
        echo "$path"
        continue
    fi
    printf '%-70s ' "$path"
    go list -m -versions "$path" 2>/dev/null || printf '(no versions found)'
    echo
done
