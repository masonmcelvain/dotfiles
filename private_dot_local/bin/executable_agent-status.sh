#!/usr/bin/env bash
# Claude Code hook handler: persist agent status to ~/.cache/agents/<session_id>.json
# and mirror it in the name of the zellij tab the agent runs in, labeled with
# the session's title (falling back to the tab's original name):
#   "● title" working, "○ title" needs input, "✓ title" done.
# Registered for hook events in ~/.claude/settings.json by dot_claude/modify_settings.json.
# Usage: receives hook-event JSON on stdin; must be fast and never fail the agent.

set -uo pipefail

state_dir="${XDG_CACHE_HOME:-$HOME/.cache}/agents"

glyph_for() {
    case "$1" in
        working) printf '●' ;;
        waiting) printf '○' ;;
        done) printf '✓' ;;
    esac
}

# Print "id<TAB>name" for the tab holding this agent's pane, stripping any status
# glyph this script previously added to the name. Strip only the glyph: the rest
# is the tab's real name, even when it starts with "claude".
#
# The tab is resolved from ZELLIJ_PANE_ID, never from the focused tab: hooks fire
# while the user is off in another tab, and `current-tab-info` would report
# *their* tab and rebind us to it. Filter out plugin panes -- they share the
# numeric id space with terminals (plugin_0 and terminal_0 are both id 0).
capture_tab() {
    local pane out
    pane=${ZELLIJ_PANE_ID:-}
    pane=${pane#terminal_}
    [ -n "$pane" ] || return 1
    out=$(zellij action list-panes -j -t 2>/dev/null |
        jq -r --arg pane "$pane" '
            first(.[] | select((.is_plugin | not) and (.id | tostring) == $pane)
                  | "\(.tab_id)\t\(.tab_name)")' |
        sed -E 's/\t[●○✓] ?/\t/')
    [ -n "$out" ] || return 1
    printf '%s\n' "$out"
}

# Latest session title Claude Code wrote to the transcript: an explicit name
# (claude -n, session rename) lands as "agent-name", the auto-generated title
# as "ai-title"; newest entry wins. tac reads from the end, so this stays fast
# on large transcripts. These entry types are undocumented internals: if they
# vanish, this prints nothing and callers fall back to the original tab name.
session_title() {
    local path=$1
    [ -n "$path" ] && [ -r "$path" ] || return 0
    tac "$path" 2>/dev/null |
        grep -m1 -e '"type":"agent-name"' -e '"type":"ai-title"' |
        jq -r '.agentName // .aiTitle // empty'
}

restore_tab() {
    local state=$1 tab_id orig
    tab_id=$(jq -r '.zellij.tab_id // empty' <<<"$state")
    orig=$(jq -r '.zellij.original_tab_name // empty' <<<"$state")
    [ -n "$tab_id" ] && [ -n "$orig" ] || return 0
    zellij action rename-tab --tab-id "$tab_id" "$orig" 2>/dev/null || true
}

main() {
    local input event session_id
    input=$(cat)
    event=$(jq -r '.hook_event_name // empty' <<<"$input")
    session_id=$(jq -r '.session_id // empty' <<<"$input")
    [ -n "$event" ] && [ -n "$session_id" ] || return 0

    local status
    case "$event" in
        SessionStart | Notification) status="waiting" ;;
        UserPromptSubmit | PostToolUse) status="working" ;;
        Stop) status="done" ;;
        SessionEnd) status="" ;;
        *) return 0 ;;
    esac

    mkdir -p "$state_dir"
    local state_file="$state_dir/$session_id.json"
    local old
    old=$(cat "$state_file" 2>/dev/null) || true
    [ -n "$old" ] || old='{}'

    if [ "$event" = SessionEnd ]; then
        [ -n "${ZELLIJ:-}" ] && restore_tab "$old"
        rm -f "$state_file"
        return 0
    fi

    local prev_status tab_id orig_name last_name prev_zsession prev_zpane
    prev_status=$(jq -r '.status // empty' <<<"$old")
    tab_id=$(jq -r '.zellij.tab_id // empty' <<<"$old")
    orig_name=$(jq -r '.zellij.original_tab_name // empty' <<<"$old")
    last_name=$(jq -r '.zellij.last_name // empty' <<<"$old")
    prev_zsession=$(jq -r '.zellij.session // empty' <<<"$old")
    prev_zpane=$(jq -r '.zellij.pane_id // empty' <<<"$old")

    # Capture the agent's tab at session start, re-capture on every prompt (the
    # user may have renamed the tab or moved the pane), and otherwise reuse the
    # stored tab id -- unless it was recorded for a different zellij session or
    # pane, as after `claude --resume` in a new window, where it now points at
    # some unrelated tab. Renames target the stable tab id, so they land on the
    # agent's tab even while another tab is focused.
    if [ -n "${ZELLIJ:-}" ]; then
        local recapture="" cap cap_name
        case "$event" in
            SessionStart | UserPromptSubmit) recapture=1 ;;
            *)
                [ -n "$tab_id" ] && [ "$prev_zsession" = "${ZELLIJ_SESSION_NAME:-}" ] &&
                    [ "$prev_zpane" = "${ZELLIJ_PANE_ID:-}" ] || recapture=1
                ;;
        esac
        if [ -n "$recapture" ] && cap=$(capture_tab); then
            tab_id=${cap%%$'\t'*}
            # Keep the stored name when the capture is empty (a renamed tab
            # with no name of its own) or is just the session title this
            # script set on a previous rename.
            cap_name=${cap#*$'\t'}
            if [ -n "$cap_name" ] && [ "$cap_name" != "$last_name" ]; then
                orig_name=$cap_name
            fi
        fi
    fi

    local now cwd msg title show_name new tmp
    now=$(date -Is)
    cwd=$(jq -r '.cwd // empty' <<<"$input")
    msg=""
    [ "$event" = Notification ] && msg=$(jq -r '.message // empty' <<<"$input")
    title=$(session_title "$(jq -r '.transcript_path // empty' <<<"$input")")
    show_name=${title:-$orig_name}
    [ "${#show_name}" -gt 50 ] && show_name="${show_name:0:49}…"
    new=$(jq -n \
        --argjson old "$old" \
        --arg session_id "$session_id" \
        --arg status "$status" \
        --arg event "$event" \
        --arg now "$now" \
        --arg cwd "$cwd" \
        --arg msg "$msg" \
        --arg title "$title" \
        --arg zsession "${ZELLIJ_SESSION_NAME:-}" \
        --arg zpane "${ZELLIJ_PANE_ID:-}" \
        --arg tab_id "$tab_id" \
        --arg orig "$orig_name" \
        --arg last_name "$show_name" '
        $old + {
            schema: 1,
            agent: "claude",
            session_id: $session_id,
            status: $status,
            last_event: $event,
            updated_at: $now
        }
        | if $cwd != "" then .cwd = $cwd else . end
        | if $msg != "" then .message = $msg else . end
        | if $title != "" then .title = $title else . end
        | if $tab_id != "" then
              .zellij = {
                  session: $zsession,
                  pane_id: $zpane,
                  tab_id: ($tab_id | tonumber),
                  original_tab_name: $orig,
                  last_name: $last_name
              }
          else . end')
    tmp=$(mktemp "$state_dir/.${session_id}.XXXXXX")
    printf '%s\n' "$new" >"$tmp" && mv "$tmp" "$state_file"

    # PostToolUse fires on every tool call; rename only when the glyph or the
    # name changes (the session title appears and updates mid-session).
    if [ -n "${ZELLIJ:-}" ] && [ -n "$tab_id" ] &&
        { [ "$status" != "$prev_status" ] || [ "$show_name" != "$last_name" ]; }; then
        local label
        label="$(glyph_for "$status")"
        [ -n "$show_name" ] && label="$label $show_name"
        zellij action rename-tab --tab-id "$tab_id" "$label" 2>/dev/null || true
    fi
    return 0
}

main || true
exit 0
