#!/usr/bin/env bash
# Claude Code hook handler: persist agent status to ~/.cache/agents/<session_id>.json
# and mirror it in the name of the zellij tab the agent runs in:
#   "● claude: name" working, "○ claude: name" needs input, "✓ claude: name" done.
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

# Print "id<TAB>name" for the session's active tab, stripping any status prefix
# this script previously added to the name.
capture_tab() {
    local info id name
    info=$(zellij action current-tab-info 2>/dev/null) || return 1
    id=$(sed -n 's/^id: //p' <<<"$info")
    name=$(sed -n 's/^name: //p' <<<"$info" | sed -E 's/^[●○✓] claude(: )?//')
    [ -n "$id" ] || return 1
    printf '%s\t%s\n' "$id" "$name"
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

    local prev_status tab_id orig_name
    prev_status=$(jq -r '.status // empty' <<<"$old")
    tab_id=$(jq -r '.zellij.tab_id // empty' <<<"$old")
    orig_name=$(jq -r '.zellij.original_tab_name // empty' <<<"$old")

    # Capture the agent's tab at session start and re-capture whenever the user
    # submits a prompt (they must be in this tab to type). Later renames target
    # the stable tab id, so they land here even while another tab is focused.
    if [ -n "${ZELLIJ:-}" ]; then
        local recapture="" cap
        case "$event" in
            SessionStart | UserPromptSubmit) recapture=1 ;;
            *) [ -n "$tab_id" ] || recapture=1 ;;
        esac
        if [ -n "$recapture" ] && cap=$(capture_tab); then
            tab_id=${cap%%$'\t'*}
            orig_name=${cap#*$'\t'}
        fi
    fi

    local now cwd msg new tmp
    now=$(date -Is)
    cwd=$(jq -r '.cwd // empty' <<<"$input")
    msg=""
    [ "$event" = Notification ] && msg=$(jq -r '.message // empty' <<<"$input")
    new=$(jq -n \
        --argjson old "$old" \
        --arg session_id "$session_id" \
        --arg status "$status" \
        --arg event "$event" \
        --arg now "$now" \
        --arg cwd "$cwd" \
        --arg msg "$msg" \
        --arg zsession "${ZELLIJ_SESSION_NAME:-}" \
        --arg zpane "${ZELLIJ_PANE_ID:-}" \
        --arg tab_id "$tab_id" \
        --arg orig "$orig_name" '
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
        | if $tab_id != "" then
              .zellij = {
                  session: $zsession,
                  pane_id: $zpane,
                  tab_id: ($tab_id | tonumber),
                  original_tab_name: $orig
              }
          else . end')
    tmp=$(mktemp "$state_dir/.${session_id}.XXXXXX")
    printf '%s\n' "$new" >"$tmp" && mv "$tmp" "$state_file"

    # PostToolUse fires on every tool call; only rename on an actual transition.
    if [ -n "${ZELLIJ:-}" ] && [ -n "$tab_id" ] && [ "$status" != "$prev_status" ]; then
        local label
        label="$(glyph_for "$status") claude"
        [ -n "$orig_name" ] && label="$label: $orig_name"
        zellij action rename-tab --tab-id "$tab_id" "$label" 2>/dev/null || true
    fi
    return 0
}

main || true
exit 0
