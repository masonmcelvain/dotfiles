// mm is the zellij and Claude Code tooling from these dotfiles: pickers over
// agents and sessions, fresh sessions in the Code/ slots, and the hook that
// tracks agent status.
package main

import (
	"fmt"
	"os"
)

const usage = `usage: mm COMMAND
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch cmd := os.Args[1]; cmd {
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "mm: unknown command %s\n\n%s", cmd, usage)
		os.Exit(1)
	}
}
