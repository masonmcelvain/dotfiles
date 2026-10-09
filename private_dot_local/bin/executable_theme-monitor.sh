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

# Panes hx has exited from keep the background it pinned with OSC 11 (see
# ~/.bashrc), which shells clear at their next prompt. This catches the rest,
# like a claude pane that ran hx. A pane hx is still running in is reset too,
# but its everforest background matches alacritty's, so nothing changes.
reset_pane_colors() {
	local session="$1" id
	zellij -s "$session" action list-panes --json 2>/dev/null |
		jq -r '.[] | select((.is_plugin | not) and .default_bg != null) | .id' |
		while IFS= read -r id; do
			zellij -s "$session" action set-pane-color --pane-id "terminal_$id" --reset >/dev/null 2>&1 || true
		done
}

apply_theme() {
	local mode="$1"
	ln -sf "$HOME/.config/alacritty/everforest_${mode}.toml" \
		"$HOME/.config/alacritty/theme.toml"
	while IFS= read -r session; do
		[ -z "$session" ] && continue
		zellij -s "$session" action "set-${mode}-theme" >/dev/null 2>&1 || true
		reset_pane_colors "$session"
	done < <(zellij list-sessions -n 2>/dev/null | grep -v "EXITED" | awk '{print $1}')

	# delta's terminal-background detection is unreliable inside zellij, pin it
	mkdir -p "$HOME/.config/git"
	printf '[delta]\n\t%s = true\n' "$mode" >"$HOME/.config/git/theme.gitconfig"
}

apply_theme "$(mode_from_value "$(gsettings get org.gnome.desktop.interface color-scheme)")"

gsettings monitor org.gnome.desktop.interface color-scheme | while IFS= read -r line; do
	apply_theme "$(mode_from_value "$line")"
done
