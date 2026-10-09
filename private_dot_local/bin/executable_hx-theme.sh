#!/bin/bash
# hx-theme - launch helix with a temporary light/dark theme override

if [[ $# -lt 3 ]]; then
    echo "Usage: hx-theme <light-theme> <dark-theme> <file...>"
    exit 1
fi

# Pick by GNOME's setting at launch rather than helix's own light/dark
# detection, the same check the bat wrapper makes
theme="$1"
if [ "$(gsettings get org.gnome.desktop.interface color-scheme 2>/dev/null)" = "'prefer-dark'" ]; then
    theme="$2"
fi
shift 2

tmp=$(mktemp)
config="$HOME/.config/helix/config.toml"

{
    echo "theme = \"$theme\""
    grep -v '^theme\s*=' "$config"
} > "$tmp"

hx -c "$tmp" "$@"
rm "$tmp"
