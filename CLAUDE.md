# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A chezmoi-managed dotfiles repository for two machines: `mason-xps` (Ubuntu desktop) and `cominor` (remote CentOS). The chezmoi source directory is the repo root; there is no `.chezmoiroot`.

## Chezmoi Commands

```bash
chezmoi apply              # Apply all changes to home directory
chezmoi apply ~/.bashrc    # Apply a single target file
chezmoi diff               # Preview changes before applying
chezmoi edit ~/.bashrc     # Edit the source file for a target
chezmoi cd                 # cd into this source directory
```

## Chezmoi Naming Conventions

Files use chezmoi's source-state naming:

- `dot_` prefix -> dotfile (e.g., `dot_bashrc.tmpl` -> `~/.bashrc`)
- `private_` prefix -> restricted permissions (e.g., `private_dot_config/` -> `~/.config/`)
- `executable_` prefix -> executable permission
- `.tmpl` suffix -> Go template, rendered at apply time

## Templating

Templates use Go's `text/template` syntax with chezmoi data. The primary branching variable is `.chezmoi.hostname`:

```
{{ if eq .chezmoi.hostname "mason-xps" -}}
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
- **Editors**: `private_dot_config/helix/` (primary editor), `dot_vimrc.tmpl` (fallback)
- **Terminals**: `private_dot_config/alacritty/` (emulator), `private_dot_config/zellij/` (multiplexer), `dot_tmux.conf.tmpl` (alt multiplexer)
- **Fonts**: `dot_fonts/` - SF Mono, SF Pro, Liga SF Mono Nerd Font, Apple Color Emoji
- **Scripts**: `bin/executable_vnstat_graph.sh`, `private_dot_local/bin/executable_hx-theme.sh`
- **Notifications**: `private_dot_local/bin/executable_notify.tmpl` picks whatever transport the host can reach - `notify-send` when a D-Bus session exists (mason-xps), an ntfy.sh push when `.ntfy.topic` is set (cominor). Its two callers are `executable_yo` (`yo <slow command>` notifies when the command finishes, and adds a terminal bell that zellij turns into a `[!]` tab flag) and the Claude Code hooks. Routing both through `notify` is what keeps the ntfy topic out of `~/.claude/settings.json`
- **Claude Code config**: `dot_claude/` - global CLAUDE.md, keybindings, custom agents and commands
- **Agent status**: `private_dot_local/bin/executable_agent-status.sh` - Claude Code hook handler that writes agent state to `~/.cache/agents/<session_id>.json` and renames the agent's zellij tab (`● name` working, `○` needs input, `✓` done)
- **Claude hooks**: `dot_claude/modify_settings.json` (a chezmoi modify script) merges the agent-status and `notify` hook registrations into `~/.claude/settings.json`; the rest of that file stays unmanaged because Claude Code live-edits it. Both hook commands are identical on every machine, so no host templating is needed. The script also removes the hooks it owns before re-adding them, which migrates any hand-written `notify-send`/`curl ... ntfy.sh` hooks in place
