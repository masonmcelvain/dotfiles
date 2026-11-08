#!/bin/bash
# hx-theme - launch helix with a temporary light/dark theme override

if [[ $# -lt 3 ]]; then
    echo "Usage: hx-theme <light-theme> <dark-theme> <file...>"
    exit 1
fi

# Pick by the desktop's setting at launch rather than helix's own light/dark
# detection: GNOME's color scheme, the same check the bat wrapper makes, or
# macOS's appearance, which only sets AppleInterfaceStyle when it's dark
theme="$1"
if [[ $OSTYPE == darwin* ]]; then
    if [ "$(defaults read -g AppleInterfaceStyle 2>/dev/null)" = "Dark" ]; then
        theme="$2"
    fi
elif [ "$(gsettings get org.gnome.desktop.interface color-scheme 2>/dev/null)" = "'prefer-dark'" ]; then
    theme="$2"
fi
shift 2

tmp=$(mktemp)
config="$HOME/.config/helix/config.toml"

{
    echo "theme = \"$theme\""
    grep -v '^theme[[:space:]]*=' "$config"
} > "$tmp"

hx -c "$tmp" "$@"
rm "$tmp"
