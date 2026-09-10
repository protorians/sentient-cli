package tui

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// IsInteractive reports whether stdin and stdout are terminals, i.e. whether
// Bubble Tea prompts and spinners can run.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// RequireInteractive returns an explicit French error when the environment is
// not a terminal (helpful in CI or piped contexts).
func RequireInteractive(what string) error {
	return fmt.Errorf("%s nécessite un terminal interactif", what)
}
