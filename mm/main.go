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
)

const usage = `usage: mm COMMAND

  agents              jump to a Claude Code agent's zellij tab
  agent-status        Claude Code hook: record agent state, label its tab

With stdout not a terminal, agents prints a plain table.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	switch cmd {
	case "agents":
		runAgents()
	case "agent-status":
		// A hook must never fail the agent.
		agents.RunHook(os.Stdin)
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

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "mm: "+format+"\n", args...)
	os.Exit(1)
}
