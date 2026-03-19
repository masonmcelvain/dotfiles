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

Custom data variables (from `chezmoidata`): `.gitconfig.email`, `.gitconfig.signingkey`.

## Architecture

- **Shell**: `dot_bashrc.tmpl`, `dot_bash_aliases.tmpl`, `dot_bash_keybindings` - hostname-conditional shell setup with fnm, zoxide, starship, fzf, direnv
- **Git**: `dot_gitconfig.tmpl`, `dot_gitmessage` - templated for per-machine GPG keys and email
- **Editors**: `private_dot_config/helix/` (primary editor), `dot_vimrc.tmpl` (fallback)
- **Terminals**: `private_dot_config/alacritty/` (emulator), `private_dot_config/zellij/` (multiplexer), `dot_tmux.conf.tmpl` (alt multiplexer)
- **Fonts**: `dot_fonts/` - SF Mono, SF Pro, Liga SF Mono Nerd Font, Apple Color Emoji
- **Scripts**: `bin/executable_vnstat_graph.sh`, `private_dot_local/bin/executable_hx-theme.sh`
- **Claude Code config**: `dot_claude/` - global CLAUDE.md, keybindings, custom agents and commands
