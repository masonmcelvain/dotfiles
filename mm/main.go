// mm is the zellij and Claude Code tooling from these dotfiles: pickers over
// agents and sessions, fresh sessions in the Code/ slots, and the hook that
// tracks agent status.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/charmbracelet/x/term"

	"github.com/masonmcelvain/dotfiles/mm/internal/agents"
	"github.com/masonmcelvain/dotfiles/mm/internal/picker"
	"github.com/masonmcelvain/dotfiles/mm/internal/sessions"
)

const usage = `usage: mm COMMAND

  agents              jump to a Claude Code agent's zellij tab
  sessions            pick a zellij session by name, branch and directory
  sessions branch B   go to the session on branch B, or start one on it
  new [BRANCH]        start a session in the least recently used Code slot
  agent-status        Claude Code hook: record agent state, label its tab
  agent-status name ID  ask Haiku for a descriptive name for agent ID's tab

With stdout not a terminal, agents and sessions print a plain table.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "agents":
		runAgents()
	case "agent-status":
		if len(args) == 2 && args[0] == "name" {
			// The hook starts this in the background; run by hand, it shows
			// why a name didn't land.
			if err := agents.RunNamer(args[1]); err != nil {
				die("%v", err)
			}
			return
		}
		// A hook must never fail the agent.
		agents.RunHook(os.Stdin)
	case "sessions":
		runSessions(args)
	case "new":
		requireZellij()
		branch := ""
		if len(args) > 0 {
			branch = args[0]
		}
		argv, msg, err := sessions.New(branch)
		if msg != "" {
			fmt.Println("mm: " + msg)
		}
		finish(argv, err)
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		die("unknown command %s\n\n%s", cmd, usage)
	}
}

func runAgents() {
	src := agents.NewSource(agents.StateDir())
	if !term.IsTerminal(os.Stdout.Fd()) {
		src.PrintTable(os.Stdout)
		return
	}
	argv, err := picker.Run(agents.Config, src)
	finish(argv, err)
}

func runSessions(args []string) {
	requireZellij()
	if len(args) > 0 {
		switch args[0] {
		case "b", "branch":
			if len(args) < 2 {
				die("usage: mm sessions branch BRANCH")
			}
			argv, msg, err := sessions.GoToBranch(args[1])
			if msg != "" {
				fmt.Println("mm: " + msg)
			}
			finish(argv, err)
			return
		default:
			die("unknown sessions command %s; try branch BRANCH", args[0])
		}
	}
	if !term.IsTerminal(os.Stdout.Fd()) {
		sessions.PrintTable(os.Stdout, sessions.Collect())
		return
	}
	src, cfg := sessions.NewSource()
	argv, err := picker.Run(cfg, src)
	finish(argv, err)
}

// finish dies on err, or else replaces this process with argv if there is
// one.
func finish(argv []string, err error) {
	if err != nil {
		die("%v", err)
	}
	if argv == nil {
		return
	}
	path, err := exec.LookPath(argv[0])
	if err != nil {
		die("%v", err)
	}
	err = syscall.Exec(path, argv, os.Environ())
	die("exec %s: %v", argv[0], err)
}

func requireZellij() {
	if _, err := exec.LookPath("zellij"); err != nil {
		die("zellij is not installed")
	}
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "mm: "+format+"\n", args...)
	os.Exit(1)
}
