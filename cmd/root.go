package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagVerbose bool
	flagNoColor bool
)

var rootCmd = &cobra.Command{
	Use:   "sentients",
	Short: "Sentient CLI — Outil de développement pour les modules Sentient",
	Long: `Sentient CLI est l'outil de développement unique pour créer, maintenir
et publier des modules dans l'écosystème Sentient.

Le cycle de vie complet : init → create → develop → debug → audit → pack → sign → publish.
`,
	Example: `  sentients init
  sentients create module
  sentients connect
  sentients pack blog-manager
  sentients sign blog-manager
  sentients publish blog-manager
  sentients audit`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "activer les logs détaillés")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "désactiver les couleurs")

	rootCmd.SetVersionTemplate("sentients {{.Version}}\n")

	rootCmd.AddCommand(
		initCmd,
		createCmd,
		connectCmd,
		disconnectCmd,
		packCmd,
		signCmd,
		publishCmd,
		linkCmd,
		unlinkCmd,
		debugCmd,
		auditCmd,
	)
}

// Execute runs the CLI. The version/commit/date build info comes from
// ldflags (see .goreleaser.yaml).
func Execute(version, commit, date string) {
	// rootCmd.Version is read by Cobra at execution time, so it is computed
	// here (not in init) to pick up the values injected via ldflags.
	rootCmd.Version = fmt.Sprintf("v%s (%s/%s) %s", version, runtime.GOOS, runtime.GOARCH, commit)

	if flagNoColor {
		lipgloss.SetColorProfile(termenv.Ascii)
	}

	// NFR-006: check for updates (non-blocking, cached daily)
	if updateMsg := pkg.CheckForUpdate(version); updateMsg != "" {
		s := tui.NewStyles()
		fmt.Fprintln(os.Stderr, s.Info.Render(updateMsg))
	}

	if err := rootCmd.Execute(); err != nil {
		printCmdError(err)
		os.Exit(pkg.ExitCodeFor(err))
	}
}

// printCmdError renders an error in the French spec format.
func printCmdError(err error) {
	s := tui.NewStyles()
	fmt.Fprintln(os.Stderr, s.Error.Render(pkg.FormatError(err)))
}

// debugf logs a verbose line to stderr when --verbose (or the debug env var)
// is enabled (spec NFR-005).
func debugf(format string, args ...any) {
	if !flagVerbose && os.Getenv("SENTIENT_CLI_DEBUG") == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "[sentients] "+format+"\n", args...)
}

// warn prints a warning to stderr.
func warn(message string) {
	s := tui.NewStyles()
	fmt.Fprintln(os.Stderr, s.Warning.Render("⚠ "+message))
}
