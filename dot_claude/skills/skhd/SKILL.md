---
name: skhd
description: Add, change or debug global macOS hotkeys with skhd, the hotkey daemon on the Mac. Use when I want a keyboard shortcut that works in every app, to run a command from a key, or when an skhd binding isn't firing.
---

# skhd

skhd (https://github.com/asmvik/skhd, in maintenance mode) runs as a launchd
service and runs shell commands on hotkeys system-wide, ahead of any app's own
shortcuts. Only the Mac has it.

## Where the config lives

Edit the chezmoi source `private_dot_config/skhd/skhdrc`, then `chezmoi apply
~/.config/skhd/skhdrc`; never edit `~/.config/skhd/skhdrc` directly. skhd
hotloads the file when it changes, so no restart is needed. `skhd --reload`
forces a reload if a change doesn't take.

skhd uses the first of `$XDG_CONFIG_HOME/skhd/skhdrc`, `~/.config/skhd/skhdrc`
and `~/.skhdrc` that exists, so a stray `~/.skhdrc` is ignored.

## Syntax

```
# <mods> - <key> : <command>
cmd - a : open -na /Applications/Alacritty.app
cmd + shift - return : some-script
ctrl + alt - 0x24 : echo keycode, from kVK_* values

# `->` runs the command but still passes the key to the app
cmd - k -> : some-logger

# per-app: `~` passes the key through unchanged, `*` is every other app
cmd - a [
  "Safari" ~
  *        : open -na /Applications/Alacritty.app
]

# continue a long command with a trailing backslash
cmd - t : first-command \
          || fallback-command

# include another file; stop watching keys in some apps
.load "partial_skhdrc"
.blacklist [ "kitty" ]
```

- Modifiers: `cmd`, `alt`, `ctrl`, `shift`, `fn`, `hyper` (cmd+alt+ctrl+shift)
  and `meh` (alt+ctrl+shift), and side-specific `lcmd`, `rcmd`, `lalt`,
  `ralt`, `lctrl`, `rctrl`, `lshift`, `rshift`.
- Named keys: `return`, `tab`, `space`, `backspace`, `escape`, `delete`,
  `home`, `end`, `pageup`, `pagedown`, `insert`, `left`, `right`, `up`,
  `down`, `f1`-`f20`, and media keys like `sound_up`, `brightness_down` and
  `play`. Full list: https://github.com/asmvik/skhd/issues/1
- Commands run through `$SHELL -c`, which is `/bin/bash` when launchd doesn't
  set `SHELL`. The service doesn't get my interactive PATH (no Homebrew, mise
  or fnm), so use absolute paths such as `/opt/homebrew/bin/...` or
  `/Applications/X.app/Contents/MacOS/x`.
- Per-app names are matched against the process name. Print the frontmost
  one with `osascript -e 'tell application "System Events" to name of first
  process whose frontmost is true'`.
- Modes (`:: name` and `key ; name`) exist for vim-style layers. Read the
  README before using them.

## Debugging

1. `skhd -o` prints the keysym and modifiers of each keypress (Ctrl-C to quit),
   which is the quickest way to get a binding's left-hand side right.
2. Logs: `/tmp/skhd_$USER.out.log` and `/tmp/skhd_$USER.err.log`. A parse
   error lands in the err log and leaves the previous config running.
3. Run the command by hand with `env -i HOME=$HOME /bin/bash -c '<command>'`
   to see it with the service's bare environment.
4. Nothing fires at all: check that skhd is on under System Settings → Privacy
   & Security → Accessibility (after a `brew upgrade` the binary changes, so
   toggle it off and on), then `skhd --restart-service`. Secure Keyboard
   Entry, which some terminals and password fields turn on, blocks every
   hotkey while it's active.
5. Service control: `skhd --start-service`, `--restart-service` and
   `--stop-service`.
