#!/usr/bin/env bash
# Open the commit that last touched a line, on the web.
# Usage: git-line-commit.sh [-n] <file> <line>
#
# `git blame` names the commit; `gh browse` turns it into a URL. With -n the
# URL is printed instead of opened, for hosts without a browser.

set -eu

browse_args=()
if [ "${1:-}" = "-n" ]; then
    browse_args+=(-n)
    shift
fi

if [ $# -lt 2 ]; then
    echo "Usage: $0 [-n] <file> <line>" >&2
    exit 2
fi

file="$1"
line="$2"

sha=$(git blame -L "$line,$line" --porcelain -- "$file" | awk 'NR==1 {print $1}')

case "$sha" in
    "") echo "no blame for $file:$line" >&2; exit 1 ;;
    *[!0]*) ;;
    *) echo "$file:$line is not committed yet" >&2; exit 1 ;;
esac

gh browse "${browse_args[@]}" "$sha"
