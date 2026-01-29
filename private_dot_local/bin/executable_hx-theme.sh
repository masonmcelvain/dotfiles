#!/bin/bash
# hx-theme - launch helix with a temporary theme override

if [[ $# -lt 2 ]]; then
    echo "Usage: hx-theme <theme> <file...>"
    exit 1
fi

theme="$1"
shift

tmp=$(mktemp)
config="$HOME/.config/helix/config.toml"

{
    echo "theme = \"$theme\""
    grep -v '^theme\s*=' "$config"
} > "$tmp"

hx -c "$tmp" "$@"
rm "$tmp"
