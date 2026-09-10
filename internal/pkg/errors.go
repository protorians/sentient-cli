package pkg

import "fmt"

// Exit codes per spec §11.1.
const (
	ExitOK             = 0
	ExitError          = 1 // erreur générique
	ExitAuth           = 2 // erreur d'authentification
	ExitModuleNotFound = 3 // module non trouvé
	ExitManifest       = 4 // manifest invalide
	ExitNetwork        = 5 // erreur réseau
	ExitPermission     = 6 // erreur de permission
	ExitMFA            = 7 // MFA requis / échoué
	ExitBuild          = 10
	ExitPublish        = 11
	ExitSigning        = 12
)

// Error is a categorized CLI error rendered as
// `✗ <Categorie> : <Message>` + `→ <Fix>` (spec §11.2).
type Error struct {
	Category string
	Message  string
	Fix      string
	Code     int
}

var _ error = (*Error)(nil)

func (e *Error) Error() string {
	return e.Message
}

// ExitCode returns the process exit code associated with the error.
func (e *Error) ExitCode() int {
	if e.Code == 0 {
		return ExitError
	}
	return e.Code
}

// NewError builds a categorized error.
func NewError(category, message string, code int) *Error {
	return &Error{Category: category, Message: message, Code: code}
}

// NewErrorWithFix builds a categorized error with a corrective action hint.
func NewErrorWithFix(category, message, fix string, code int) *Error {
	return &Error{Category: category, Message: message, Fix: fix, Code: code}
}

// ExitCodeFor returns the exit code implied by an arbitrary error, defaulting
// to generic error.
func ExitCodeFor(err error) int {
	if e, ok := err.(*Error); ok {
		return e.ExitCode()
	}
	return ExitError
}

// FormatError renders an error per the French spec format.
func FormatError(err error) string {
	if e, ok := err.(*Error); ok {
		out := fmt.Sprintf("✗ %s : %s", e.Category, e.Message)
		if e.Fix != "" {
			out += fmt.Sprintf("\n  → %s", e.Fix)
		}
		return out
	}
	return fmt.Sprintf("✗ Erreur : %s", err)
}
