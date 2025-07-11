# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is a chezmoi-managed dotfiles repository for personal development environment configuration. Chezmoi is a cross-platform dotfile manager that uses templating to handle machine-specific variations.

## Key Commands

### Daily Chezmoi Usage
- `chezmoi edit $FILE` - Edit a managed file (e.g., `chezmoi edit ~/.bashrc`)
- `chezmoi diff` - Preview changes before applying
- `chezmoi apply` - Apply changes to your home directory
- `chezmoi edit --apply $FILE` - Edit and apply in one step
- `chezmoi status` - See what changes would be made
- `chezmoi update` - Pull latest changes from git and apply

### Adding New Configurations
- `chezmoi add $FILE` - Add a new file to chezmoi management
- `chezmoi add --template $FILE` - Add as a template file
- `chezmoi data` - View available template variables

### Testing and Debugging
- `chezmoi doctor` - Check for configuration issues
- `chezmoi execute-template < $FILE.tmpl` - Test template rendering

## Architecture and Structure

### File Naming Conventions
- `dot_*` - Files/directories that become `.` prefixed in home directory
- `private_*` - Files with restricted permissions (600)
- `*.tmpl` - Template files processed with Go templating
- `exact_*` - Directories where chezmoi removes unmanaged files

### Key Directories
- `dot_claude/` - Claude AI configuration and custom commands
- `dot_vim/` - Vim configuration with language-specific settings
- `private_dot_config/` - Application configs (Alacritty, Helix, Starship, etc.)
- `dot_fonts/` - Font collections (SF Mono, SF Pro, Liga SF Mono Nerd Font)

### Template Variables
Templates use Go syntax and can access:
- `.chezmoi.hostname` - Machine hostname (e.g., "mason-xps")
- `.chezmoi.os` - Operating system
- `.chezmoi.arch` - Architecture
- Custom data from `.chezmoidata.*` files

Example template usage found in configs:
```
{{ if eq .chezmoi.hostname "mason-xps" }}
# XPS-specific configuration
{{ end }}
```

## Development Guidelines

### Git Workflow
- Follow conventional commits: `feat:`, `fix:`, `docs:`, `chore:`, etc.
- Use `gh` CLI for GitHub operations
- Recent commits show focus on Claude integration, shell configs, and tool permissions

### Tool Preferences
- Use `rg` instead of `grep` for searching
- Use `gh` for all GitHub-related tasks
- Prefer editing existing files over creating new ones

### Testing Changes
1. Make edits with `chezmoi edit $FILE`
2. Review with `chezmoi diff`
3. Apply with `chezmoi apply`
4. Commit changes in the chezmoi source directory

### Common Tasks
- **Update shell aliases**: `chezmoi edit ~/.bash_aliases`
- **Modify git config**: `chezmoi edit ~/.gitconfig`
- **Update vim settings**: `chezmoi edit ~/.vimrc`
- **Configure terminal**: `chezmoi edit ~/.config/alacritty/alacritty.toml`