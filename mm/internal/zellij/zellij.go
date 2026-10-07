// Package zellij reads zellij's session list and on-disk session info, and
// runs the zellij actions the pickers need.
package zellij

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
func Delete(name string) error {
	return exec.Command("zellij", "delete-session", "--force", name).Run()
}

// AttachArgv is the command that attaches this terminal to session name,
// resurrecting it if it exited. Run it by replacing this process.
func AttachArgv(name string) []string {
	return []string{"zellij", "attach", name}
}
