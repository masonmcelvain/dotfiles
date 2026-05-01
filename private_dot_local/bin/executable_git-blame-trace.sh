#!/usr/bin/env bash
# Trace a line's history across file moves and renames.
# Usage: git-blame-trace <file> <line_start> [<line_end>]
#
# Walks `git log -L` for the given file/line range (which already follows
# whole-file renames), then runs `git blame -CCC` at the introducing commit
# to detect lines that were moved/copied from another file. Recurses into
# the source file when a move is found, up to a max depth.

set -eu

if [ $# -lt 2 ]; then
    echo "Usage: $0 <file> <line_start> [<line_end>]" >&2
    exit 2
fi

file="$1"
ls="$2"
le="${3:-$2}"

max_hops=5

trace() {
    local f="$1"
    local ls="$2"
    local le="$3"
    local until="$4"
    local hop="$5"

    if [ "$hop" -ge "$max_hops" ]; then
        printf '\n--- max hops (%d) reached ---\n' "$max_hops"
        return
    fi

    printf '\n=== %s lines %s-%s ===\n' "$f" "$ls" "$le"

    local log_args=(-L "$ls,$le:$f")
    [ -n "$until" ] && log_args+=("$until")

    git log "${log_args[@]}" --no-patch \
        --pretty=format:'%h %ad %an: %s' --date=short 2>/dev/null || return 0
    echo

    # Capture the oldest commit's sha, file path at that commit, and +line
    # range. We walk the patch output and let awk's END block keep only the
    # last commit's metadata (oldest, since log walks newest -> oldest).
    local meta
    meta=$(git log "${log_args[@]}" --pretty=format:'__C__ %H' 2>/dev/null \
        | awk '
            /^__C__ / { sha=$2; ff=""; hh=""; next }
            /^\+\+\+ b\// && ff=="" { ff=substr($0, 7) }
            /^@@ / && hh=="" { hh=$0 }
            END { printf "%s\t%s\t%s\n", sha, ff, hh }
        ')

    local oldest oldest_file oldest_hunk
    IFS=$'\t' read -r oldest oldest_file oldest_hunk <<< "$meta"
    [ -n "$oldest" ] && [ -n "$oldest_file" ] && [ -n "$oldest_hunk" ] || return 0

    git rev-parse --verify "$oldest^" >/dev/null 2>&1 || return 0

    local at_start at_count
    at_start=$(printf '%s' "$oldest_hunk" | sed -nE 's/^@@ -[0-9]+(,[0-9]+)? \+([0-9]+).*/\2/p')
    at_count=$(printf '%s' "$oldest_hunk" | sed -nE 's/^@@ -[0-9]+(,[0-9]+)? \+[0-9]+,([0-9]+).*/\2/p')
    at_count="${at_count:-1}"
    [ -n "$at_start" ] && [ "$at_count" -gt 0 ] || return 0

    local at_end=$((at_start + at_count - 1))

    local porcelain
    porcelain=$(git blame -CCC --porcelain -L "$at_start,$at_end" "$oldest" -- "$oldest_file" 2>/dev/null) || return 0

    local src_file src_line src_sha
    src_sha=$(printf '%s\n' "$porcelain" | awk 'NR==1 {print $1}')
    src_line=$(printf '%s\n' "$porcelain" | awk 'NR==1 {print $2}')
    src_file=$(printf '%s\n' "$porcelain" | awk '/^filename / { sub(/^filename /, ""); print; exit }')

    if [ -z "$src_file" ] || [ "$src_file" = "$oldest_file" ]; then
        return 0
    fi

    printf '\n--- moved from %s line %s (commit %s) ---\n' \
        "$src_file" "$src_line" "${src_sha:0:7}"

    trace "$src_file" "$src_line" "$src_line" "$src_sha" "$((hop + 1))"
}

trace "$file" "$ls" "$le" "" 0
