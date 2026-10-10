// Package zellij reads zellij's session list and on-disk session info, and
// runs the zellij actions the pickers need.
package zellij

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/masonmcelvain/dotfiles/mm/internal/text"
)

// Session is one line of `zellij list-sessions -n`.
type Session struct {
	Name   string
	Exited bool
	// Created is how many seconds ago zellij created the session.
	Created int64
}

// ParseSessions reads `zellij list-sessions -n` output, whose lines look like
// "name [Created 1day 6h 27m 45s ago] (EXITED - attach to resurrect)".
func ParseSessions(out string) []Session {
	var sessions []Session
	for line := range strings.Lines(out) {
		line = strings.TrimRight(line, "\n")
		name, rest, _ := strings.Cut(line, " ")
		if name == "" {
			continue
		}
		s := Session{Name: name, Exited: strings.Contains(rest, "EXITED")}
		if _, created, ok := strings.Cut(rest, "[Created "); ok {
			created, _, _ = strings.Cut(created, " ago]")
			s.Created = text.ParseDuration(created)
		}
		sessions = append(sessions, s)
	}
	return sessions
}

// ListSessions lists every session zellij knows, live and exited. A failure
// reads as no sessions, as it does for zellij itself with no server running.
func ListSessions() []Session {
	out, _ := exec.Command("zellij", "list-sessions", "-n").Output()
	return ParseSessions(string(out))
}

// InCurrent reports whether this process runs inside a zellij session, and
// which one.
func InCurrent() (session string, inside bool) {
	return os.Getenv("ZELLIJ_SESSION_NAME"), os.Getenv("ZELLIJ") != ""
}

// InfoDir is where zellij keeps per-session layout and metadata. Its parent
// is a contract version ("contract_version_1") that was a plain zellij
// version in earlier releases, so take the freshest rather than hardcode it.
func InfoDir() string {
	cache := os.Getenv("XDG_CACHE_HOME")
	if cache == "" {
		cache = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	dirs, _ := filepath.Glob(filepath.Join(cache, "zellij", "*", "session_info"))
	var best string
	var newest time.Time
	for _, d := range dirs {
		fi, err := os.Stat(d)
		if err != nil || !fi.IsDir() {
			continue
		}
		if fi.ModTime().After(newest) {
			newest, best = fi.ModTime(), d
		}
	}
	return best
}

// ParseLayoutCWD returns the first `cwd "..."` node of a session layout. The
// layout carries exactly one top-level cwd, which every pane inherits, and
// that's what makes "the session's directory" well defined.
func ParseLayoutCWD(layout string) string {
	for line := range strings.Lines(layout) {
		trimmed := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(trimmed, "cwd ") {
			continue
		}
		start := strings.Index(trimmed, `"`)
		end := strings.LastIndex(trimmed, `"`)
		if start < 0 || end <= start {
			return ""
		}
		return trimmed[start+1 : end]
	}
	return ""
}

// SessionCWD is the directory session name is rooted in, or "" when zellij
// hasn't written its layout yet.
func SessionCWD(info, name string) string {
	if info == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(info, name, "session-layout.kdl"))
	if err != nil {
		return ""
	}
	return ParseLayoutCWD(string(data))
}

// SessionMtime is when zellij last rewrote session name's metadata. It stops
// when the session exits, which makes this when an exited session was last
// used. Live sessions share a rewrite tick, so it means nothing for them.
func SessionMtime(info, name string) (time.Time, bool) {
	if info == "" {
		return time.Time{}, false
	}
	fi, err := os.Stat(filepath.Join(info, name, "session-metadata.kdl"))
	if err != nil {
		return time.Time{}, false
	}
	return fi.ModTime(), true
}

// Action runs `zellij action ARGS...`, aimed at session when it's set and at
// the current one otherwise. A stale ZELLIJ_SESSION_NAME makes zellij hang
// rather than fail, hence the timeout.
func Action(timeout time.Duration, session string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "zellij", append([]string{"action"}, args...)...)
	if session != "" {
		cmd.Env = append(os.Environ(), "ZELLIJ_SESSION_NAME="+session)
	}
	return cmd.Output()
}

// Switch moves this client, which must be inside zellij, to session name,
// resurrecting it if it exited. It leaves the current session running.
func Switch(name string, args ...string) error {
	_, err := Action(10*time.Second, "", append([]string{"switch-session", name}, args...)...)
	return err
}

// Delete deletes session name, killing it first if it's live.
//
// `delete-session --force` can't be trusted with a live session: it removes
// the session's cache right after asking the server to die, but the dying
// server keeps rewriting that cache for a few milliseconds more, and a write
// that lands after the removal recreates it, bringing the session back as an
// exited one. The server's last write comes before it removes its socket, so
// kill it, wait for the socket to go, and only then delete. Should the wait
// time out, the forced delete still goes ahead.
//
// A killed session that never saved a layout leaves nothing to resurrect,
// so there's nothing left to delete and zellij reports it as not found;
// that's a success, as long as it's really gone.
func Delete(name string) error {
	if socketExists(name) && exec.Command("zellij", "kill-session", name).Run() == nil {
		for deadline := time.Now().Add(5 * time.Second); socketExists(name) && time.Now().Before(deadline); {
			time.Sleep(20 * time.Millisecond)
		}
	}
	err := exec.Command("zellij", "delete-session", "--force", name).Run()
	if err != nil && !slices.ContainsFunc(ListSessions(), func(s Session) bool { return s.Name == name }) {
		return nil
	}
	return err
}

// socketExists reports whether session name's server socket is still there,
// in any contract version's directory. The sockets live where zellij puts
// them: $ZELLIJ_SOCKET_DIR, else $XDG_RUNTIME_DIR/zellij, else
// $TMPDIR/zellij-UID.
func socketExists(name string) bool {
	root := os.Getenv("ZELLIJ_SOCKET_DIR")
	if root == "" {
		if rt := os.Getenv("XDG_RUNTIME_DIR"); rt != "" {
			root = filepath.Join(rt, "zellij")
		} else {
			root = filepath.Join(os.TempDir(), fmt.Sprintf("zellij-%d", os.Getuid()))
		}
	}
	dirs, _ := filepath.Glob(filepath.Join(root, "*"))
	for _, d := range dirs {
		if fi, err := os.Stat(filepath.Join(d, name)); err == nil && fi.Mode()&os.ModeSocket != 0 {
			return true
		}
	}
	return false
}

// AttachArgv is the command that attaches this terminal to session name,
// resurrecting it if it exited. Run it by replacing this process.
func AttachArgv(name string) []string {
	return []string{"zellij", "attach", name}
}
