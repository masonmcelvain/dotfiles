# picker.sh: the full-screen list picker shared by agents and zellij-sessions.
# Sourced, not run. Draws a header, the rows, a detail pane about the selected
# row and a key hint, reloading the rows on every refresh tick.
#
# The sourcing script sets PICKER_TITLE, PICKER_HELP, PICKER_EMPTY and
# PICKER_DETAIL_LINES, and defines:
#   picker_load            reload the rows and set PICKER_N to their count
#   picker_render WIDTH    set PICKER_ROWS[i] to row i's text, at most WIDTH
#                          columns, and PICKER_DIM[i]=1 for rows to dim
#   picker_detail I WIDTH  set PICKER_DETAIL to up to PICKER_DETAIL_LINES
#                          lines about row I
#   picker_open I          act on row I; set PICKER_QUIT=1 to leave
#   picker_key CHAR        optional: handle a key the picker doesn't bind
# Either hook can set PICKER_FLASH to show a message in place of the hint, or
# call picker_ask to put a question there and read the answer.
#
# Keys: 1-9 open, j/k or arrows move (wrapping around the ends), g/G
# top/bottom, enter open, r refresh, ctrl+l redraw, q or esc quit.

PICKER_N=0 PICKER_SEL=0 PICKER_SCROLL=0 PICKER_FLASH='' PICKER_QUIT=''
PICKER_ROWS=() PICKER_DIM=() PICKER_DETAIL=()

C_RESET=$'\e[0m' C_DIM=$'\e[2m' C_UNDIM=$'\e[22m' C_FG=$'\e[39m'
C_FLASH=$'\e[38;5;208m'

# Truncate $1 to $2 chars (adding …) and pad to exactly $2. printf's %-*s pads
# by bytes, so pad manually with char counts to keep unicode columns aligned.
fit() {
    local s=$1 w=$2
    ((${#s} > w)) && s="${s:0:w-1}…"
    printf '%s%*s' "$s" $((w - ${#s})) ''
}

# Seconds -> the largest whole unit: 42s, 5m, 3h, 2d, 4mo, 1y.
age_str() {
    local s=$1
    ((s < 0)) && s=0
    if ((s < 60)); then printf '%ds' "$s"
    elif ((s < 3600)); then printf '%dm' $((s / 60))
    elif ((s < 86400)); then printf '%dh' $((s / 3600))
    elif ((s < 2592000)); then printf '%dd' $((s / 86400))
    elif ((s < 31536000)); then printf '%dmo' $((s / 2592000))
    else printf '%dy' $((s / 31536000)); fi
}

picker_reload() {
    picker_load
    ((PICKER_SEL >= PICKER_N)) && PICKER_SEL=$((PICKER_N - 1))
    ((PICKER_SEL < 0)) && PICKER_SEL=0
}

picker_draw() {
    local cols lines n=$PICKER_N buf='' i
    cols=$(tput cols)
    lines=$(tput lines)

    local host=${HOSTNAME:-$(hostname)}
    local head="  $PICKER_TITLE · $n"
    buf+="${C_DIM}${head}"
    buf+="$(printf '%*s' $((cols - ${#head} - ${#host} - 2)) '')${host}${C_RESET}"
    buf+=$'\e[K\n\e[K\n'

    # Rows sit behind an 8-column "  -> 1  " gutter.
    PICKER_ROWS=() PICKER_DIM=()
    ((n > 0)) && picker_render $((cols - 8))

    # Everything but the list: header and its gap, the gap after the list,
    # two separators, the detail lines and the hint.
    local list_h=$((lines - 6 - PICKER_DETAIL_LINES))
    ((list_h < 1)) && list_h=1
    ((PICKER_SEL < PICKER_SCROLL)) && PICKER_SCROLL=$PICKER_SEL
    ((PICKER_SEL >= PICKER_SCROLL + list_h)) && PICKER_SCROLL=$((PICKER_SEL - list_h + 1))

    local marker num numseq attrs
    for ((i = PICKER_SCROLL; i < n && i < PICKER_SCROLL + list_h; i++)); do
        marker='  ' && ((i == PICKER_SEL)) && marker='->'
        num=' ' && ((i < 9)) && num=$((i + 1))
        if [ -n "${PICKER_DIM[i]:-}" ]; then
            attrs=$C_DIM
            numseq=$num
        else
            attrs=''
            numseq="${C_DIM}${num}${C_UNDIM}"
        fi
        buf+="${attrs}  ${marker} ${numseq}  ${PICKER_ROWS[i]}${C_RESET}"$'\e[K\n'
    done
    if ((n == 0)); then
        buf+="  ${C_DIM}${PICKER_EMPTY}${C_RESET}"$'\e[K\n'
    fi

    local sep_line
    printf -v sep_line '%*s' $((cols - 2)) ''
    sep_line="  ${C_DIM}${sep_line// /─}${C_RESET}"$'\e[K\n'

    PICKER_DETAIL=()
    ((n > 0)) && picker_detail "$PICKER_SEL" $((cols - 4))
    buf+=$'\e[K\n'"$sep_line"
    for ((i = 0; i < PICKER_DETAIL_LINES; i++)); do
        buf+="   ${PICKER_DETAIL[i]:-}${C_RESET}"$'\e[K\n'
    done
    buf+="$sep_line"
    if [ -n "$PICKER_FLASH" ]; then
        buf+="   ${C_FLASH}${PICKER_FLASH}${C_RESET}"$'\e[K'
    else
        buf+="   ${C_DIM}${PICKER_HELP}${C_RESET}"$'\e[K'
    fi
    printf '\e[H%s\e[J' "$buf"
}

# Sets PICKER_KEY; returns 1 on timeout (refresh tick).
picker_get_key() {
    local k rest
    PICKER_KEY=''
    IFS= read -rsn1 -t 1 k || return 1
    if [[ $k == $'\e' ]]; then
        IFS= read -rsn2 -t 0.05 rest || {
            PICKER_KEY=quit
            return 0
        }
        case $rest in
            '[A') PICKER_KEY=up ;;
            '[B') PICKER_KEY=down ;;
            '[Z') PICKER_KEY=up ;; # shift+tab, undocumented alias for k
            *) PICKER_KEY=other ;;
        esac
        return 0
    fi
    case $k in
        '') PICKER_KEY=enter ;;
        $'\t') PICKER_KEY=down ;; # tab, undocumented alias for j
        $'\f') PICKER_KEY=redraw ;; # ctrl+l
        j) PICKER_KEY=down ;;
        k) PICKER_KEY=up ;;
        g) PICKER_KEY=top ;;
        G) PICKER_KEY=bottom ;;
        q) PICKER_KEY=quit ;;
        r) PICKER_KEY=reload ;;
        [1-9]) PICKER_KEY=num:$k ;;
        *) PICKER_KEY=key:$k ;;
    esac
}

# Shows $1 in place of the hint and sets PICKER_ANSWER to the next key. Waits
# without a refresh tick, so the rows can't shift under the question.
picker_ask() {
    PICKER_FLASH=$1
    picker_draw
    PICKER_FLASH=''
    PICKER_ANSWER=''
    IFS= read -rsn1 PICKER_ANSWER
}

picker_cleanup() {
    tput rmcup
    tput cnorm
}

picker_run() {
    tput smcup
    tput civis
    trap picker_cleanup EXIT
    trap 'exit 130' INT TERM
    trap : WINCH # interrupts read -> immediate reload + redraw

    local i
    picker_reload
    while [ -z "$PICKER_QUIT" ]; do
        picker_draw
        if picker_get_key; then
            PICKER_FLASH=''
            case $PICKER_KEY in
                quit) PICKER_QUIT=1 ;;
                # Moving past either end wraps around to the other.
                down) ((PICKER_N > 0)) && PICKER_SEL=$(((PICKER_SEL + 1) % PICKER_N)) ;;
                up) ((PICKER_N > 0)) && PICKER_SEL=$(((PICKER_SEL - 1 + PICKER_N) % PICKER_N)) ;;
                top) PICKER_SEL=0 ;;
                bottom) ((PICKER_N > 0)) && PICKER_SEL=$((PICKER_N - 1)) ;;
                enter) ((PICKER_N > 0)) && picker_open "$PICKER_SEL" ;;
                num:*)
                    i=$((${PICKER_KEY#num:} - 1))
                    ((i < PICKER_N)) && picker_open "$i"
                    ;;
                reload) picker_reload ;;
                # Wipe the screen too, for when something has drawn over it.
                redraw) printf '\e[2J' && picker_reload ;;
                key:*) declare -F picker_key >/dev/null && picker_key "${PICKER_KEY#key:}" ;;
            esac
        else
            picker_reload
        fi
    done
}
