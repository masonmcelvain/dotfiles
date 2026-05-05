#!/bin/bash
# Watch GNOME color-scheme and sync alacritty + all zellij sessions to match.
set -uo pipefail

mode_from_value() {
	case "$1" in
		*"'default'"*) echo "light" ;;
		*"'prefer-light'"*) echo "light" ;;
		*) echo "dark" ;;
	esac
}

apply_theme() {
	local mode="$1"
	ln -sf "$HOME/.config/alacritty/everforest_${mode}.toml" \
		"$HOME/.config/alacritty/theme.toml"
	while IFS= read -r session; do
		[ -z "$session" ] && continue
		zellij -s "$session" action "set-${mode}-theme" >/dev/null 2>&1 || true
	done < <(zellij list-sessions -n 2>/dev/null | grep -v "EXITED" | awk '{print $1}')
}

apply_theme "$(mode_from_value "$(gsettings get org.gnome.desktop.interface color-scheme)")"

gsettings monitor org.gnome.desktop.interface color-scheme | while IFS= read -r line; do
	apply_theme "$(mode_from_value "$line")"
done
