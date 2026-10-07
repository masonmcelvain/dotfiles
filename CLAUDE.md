# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A chezmoi-managed dotfiles repository for two machines: `masons-xps` (Ubuntu desktop) and `cominor` (remote CentOS). The chezmoi source directory is the repo root; there is no `.chezmoiroot`.

## Chezmoi Commands

```bash
chezmoi apply              # Apply all changes to home directory
chezmoi apply ~/.bashrc    # Apply a single target file
chezmoi diff               # Preview changes before applying
chezmoi edit ~/.bashrc     # Edit the source file for a target
chezmoi cd                 # cd into this source directory
```

## Linting

`.mise.toml` pins shellcheck and chezmoi as dev tools for this repo. Run `mise run lint` (`.mise/tasks/lint`) after changing a shell script or shell template; CI (`.github/workflows/lint.yml`) runs the same task. mise only activates in interactive shells, so from a non-interactive one call `~/.local/bin/mise run lint`, or `mise install` first on a fresh checkout.

Plain scripts are found by shebang, plus the sourced `picker.sh`. Shell templates (`dot_bash*`, `*.sh.tmpl`, or a shebang on line 1) are rendered once per host with `chezmoi execute-template --override-data` and checked as `<file>@<host>`, so both sides of every hostname branch get linted. Line numbers in those findings are the rendered file's, not the template's; find the line by its text.

## Chezmoi Naming Conventions

Files use chezmoi's source-state naming:

- `dot_` prefix -> dotfile (e.g., `dot_bashrc.tmpl` -> `~/.bashrc`)
- `private_` prefix -> restricted permissions (e.g., `private_dot_config/` -> `~/.config/`)
- `executable_` prefix -> executable permission
- `.tmpl` suffix -> Go template, rendered at apply time

## Templating

Templates use Go's `text/template` syntax with chezmoi data. The primary branching variable is `.chezmoi.hostname`:

```
{{ if eq .chezmoi.hostname "masons-xps" -}}
# Ubuntu-specific
{{- else if eq .chezmoi.hostname "cominor" -}}
# CentOS-specific
{{- end }}
```

Custom data variables live in `~/.config/chezmoi/chezmoi.toml`, which is *not* in this repo — the repo is public, so secrets belong there:

- `.gitconfig.email`, `.gitconfig.signingkey`
- `.ntfy.topic` — ntfy.sh topic used by `yo` for push notifications (optional; guard reads with `hasKey`)

## Architecture

- **Shell**: `dot_bashrc.tmpl`, `dot_bash_aliases.tmpl`, `dot_bash_keybindings` - hostname-conditional shell setup with fnm, zoxide, starship, fzf, direnv
- **Git**: `dot_gitconfig.tmpl`, `dot_gitmessage` - templated for per-machine GPG keys and email
- **SSH**: `private_dot_ssh/private_config` (masons-xps only) forwards the laptop's gpg-agent to cominor, which is the only way commits get signed there. Every connection re-binds that socket and leaves it dead when it closes, so all sessions multiplex over one persistent `ControlMaster`; `ssh -O exit cominor` drops signing on cominor until the next `ssh`
- **Editors**: `private_dot_config/helix/` (primary editor), `dot_vimrc.tmpl` (fallback)
- **Terminals**: `private_dot_config/alacritty/` (emulator, made the GNOME default by `private_dot_config/xdg-terminals.list`, which `xdg-terminal-exec` reads; masons-xps only), `private_dot_config/zellij/` (multiplexer), `dot_tmux.conf.tmpl` (alt multiplexer)
- **GNOME extensions**: `private_dot_local/private_share/private_gnome-shell/extensions/` (masons-xps only, via `.chezmoiignore`). `spotify-fullscreen@masons-xps` fullscreens Spotify when it opens, because Spotify ignores `--start-fullscreen`. `hide-date-menu-cards@masons-xps` hides the Today and World Clocks cards in the clock dropdown, because GNOME has no setting for them; it re-hides them whenever the shell re-shows them. Enabling either is a `gsettings` change to `org.gnome.shell enabled-extensions`, not a file, and on Wayland the shell only picks up a new or edited extension at the next login
- **Fonts**: `dot_fonts/` - SF Mono, SF Pro, Liga SF Mono Nerd Font, Apple Color Emoji
- **Scripts**: `bin/executable_vnstat_graph.sh`, `private_dot_local/bin/executable_hx-theme.sh`
- **Session picker**: `private_dot_local/bin/executable_zellij-sessions` - a picker over zellij sessions listing each one's git branch and directory, so a session is recognizable by the work in it rather than by its name alone. `zf` (bash) and `Alt f` (zellij, a floating pane) both open it; it attaches from a plain shell and `switch-session`s from inside one, and either path resurrects an exited session. `d` deletes the selected session after a `y` confirmation, killing it first if it's live (`zellij delete-session --force`), then detaches HEAD in its checkout so another Code slot can check out the branch. It skips this when the tree is dirty or another live session shares the checkout, and keeps the index mtime so `zellij-new` doesn't count the slot as freshly used. Where `zellij-new` is installed (cominor), `n` runs it to start a fresh `sN` session; the key and its hint are gated on the command existing, so the script needs no host templating. `b` (or `zellij-sessions branch NAME`, aliased to `zb`) prompts for a branch and goes to the session that has it checked out, live ones first; failing that, it runs `zellij-new NAME`, and without `zellij-new` it's an error. Session cwds are read from zellij's own cache (`~/.cache/zellij/*/session_info/<name>/session-layout.kdl`) rather than from `zellij action dump-layout`, which needs a running server per session and hangs on exited ones; branches come straight out of `.git/HEAD`, since a `git` per session is what would make it feel slow
- **Fresh sessions**: `private_dot_local/bin/executable_zellij-new` (cominor only, via `.chezmoiignore`; aliased to `zn`) starts a zellij session named `sN` in the least recently used Code slot (`/home/mmcelvain/Code` is `s1`, `/home/mmcelvain-N/Code` is `sN`) with the `sh-hx-claude` layout, nothing launched in it. A slot's last use is its git index mtime, read through `rev-parse --absolute-git-dir` because slots 2-9 are worktrees; slots that a live session is rooted in, or whose `sN` is live, are skipped, and a stale exited `sN` is deleted first so it isn't resurrected instead. It attaches from a plain shell and `switch-session`s from inside one. `zellij-new BRANCH` starts the session with BRANCH checked out, fetching just that branch from origin when it isn't local and dying if it's on neither: a free slot already holding the branch is reused, otherwise the least recently used slot without uncommitted changes switches to it
- **Pickers**: `private_dot_local/bin/picker.sh` is the full-screen list TUI behind both `zellij-sessions` and `agents` (`Alt a`, aliased to `ag`, which jumps to a Claude Code agent's tab) - the draw loop, refresh tick, keys (1-9, wrapping j/k, g/G, enter, r, ctrl-l, q) and detail pane. It's sourced from the script's own directory, not run, so it carries no `executable_` prefix; each script supplies `picker_load`/`picker_render`/`picker_detail`/`picker_open` hooks, plus an optional `picker_key` for extra keys like `agents`' `d`, and can call `picker_ask` to confirm one with a single keypress or `picker_read` to read a line of text
- **Notifications**: `private_dot_local/bin/executable_notify.tmpl` picks whatever transport the host can reach - `notify-send` when a D-Bus session exists (masons-xps), an ntfy.sh push when `.ntfy.topic` is set (cominor). Its two callers are `executable_yo` (`yo <slow command>` notifies when the command finishes, and adds a terminal bell that zellij turns into a `[!]` tab flag) and the Claude Code hooks. Routing both through `notify` is what keeps the ntfy topic out of `~/.claude/settings.json`
- **Notification stacking**: `private_dot_local/bin/executable_notify-relay` sends notifications from one long-lived process, because GNOME stacks a notification center group per sender pid and app name, and every `notify-send` is a new pid. `notify` and the ntfy subscriber try it first and fall back to `notify-send`. It runs from `private_dot_config/systemd/user/notify-relay.socket` (masons-xps only, via `ConditionHost`), which has to be enabled once with `systemctl --user enable --now notify-relay.socket`. It deliberately sends no `desktop-entry` hint: that groups by app instead, but GNOME destroys an app's notifications as soon as the sender leaves the bus
- **Notification subscriber**: `private_dot_config/ntfy/private_client.yml.tmpl` closes the loop for pushes from cominor - `ntfy-client.service` (a user unit from the `ntfy` package, not managed here) streams the topic and replays each message through `notify-send`. Only managed on masons-xps, via a conditional in `.chezmoiignore`; restart the unit after changing it. Both halves read `.ntfy.topic`, so rotating the topic is one line in `~/.config/chezmoi/chezmoi.toml` plus a `chezmoi apply`
- **Claude Code config**: `dot_claude/` - global CLAUDE.md, keybindings, custom agents and commands
- **Local review loop**: review a diff without leaving the terminal - `hkg`/`hkr` (bash) open the working tree or the branch's own work in [hunk](https://www.hunk.dev/), `c` (hunk) leaves an inline comment on the selected hunk, and the `/resolve` command (`dot_claude/commands/resolve.md`) has Claude read them back over `hunk session comment list --type user`, fix or push back on each, and clear them. `hkr` diffs from the merge base rather than the branch tip, so upstream drift stays out of the changeset; `hxr`/`hxg` are the helix counterparts, for editing the same files rather than reviewing them. The comments live in the running TUI, not the tree - close it and they're gone
- **Agent review loop**: `dot_claude/skills/review-loop/SKILL.md` runs the same round-trip with a second agent instead of me, before a PR goes out. `/review-loop` spawns the read-only `code-reviewer` agent (`dot_claude/agents/code-reviewer.md`) on the `hkr` changeset, addresses each numbered finding (fix, or decline with `file:line`), then continues the same reviewer via `SendMessage` so it re-checks the fixes and contests declines at most once. It stops on `APPROVE`, on a stalemate, or after four rounds, and reports every finding's outcome; stalemates also land as `--author claude` comments in an open hunk session
- **Hunk config**: `private_dot_config/hunk/config.toml` - theme, `watch`, and the keybinding for the bundled review skill. `dot_claude/skills/symlink_hunk-review.tmpl` resolves that skill's path at apply time (it sits inside the fnm node install, so the path moves on every node upgrade) and `.chezmoiignore` skips it on a host without hunk, since the `output` call would otherwise fail the apply
- **Agent status**: `private_dot_local/bin/executable_agent-status.sh` - Claude Code hook handler that writes agent state to `~/.cache/agents/<session_id>.json` and renames the agent's zellij tab (`● name` working, `○` needs input, `✓` done)
- **Claude hooks**: `dot_claude/modify_settings.json` (a chezmoi modify script) merges the agent-status and `notify` hook registrations into `~/.claude/settings.json`; the rest of that file stays unmanaged because Claude Code live-edits it. Both hook commands are identical on every machine, so no host templating is needed. The script also removes the hooks it owns before re-adding them, which migrates any hand-written `notify-send`/`curl ... ntfy.sh` hooks in place
