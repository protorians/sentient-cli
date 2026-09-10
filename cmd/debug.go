package cmd

import (
	"fmt"

	"github.com/protorians/sentient-cli/internal/debug"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug [module]",
	Short: "Déboguer un module ou tous les modules",
	Long: `Lance le debug d'un (ou de tous les) modules dans external_modules/.

Vérifie la conformité du module, tente de compiler en mode debug,
et affiche les logs et erreurs en temps réel.

Sans argument, tous les modules sont débuggués.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDebug(cmd, args)
	},
}

func runDebug(cmd *cobra.Command, args []string) error {
	root, err := requireProjectRoot()
	if err != nil {
		return err
	}

	debugger := &debug.Debugger{Root: root}

	if len(args) > 0 {
		return runDebugSingle(debugger, args[0])
	}
	return runDebugAll(debugger)
}

func runDebugSingle(debugger *debug.Debugger, name string) error {
	result, err := tui.RunWithSpinner(fmt.Sprintf("Debug : %s", name), func() (*debug.DebugResult, error) {
		return debugger.DebugModule(name)
	})
	if err != nil {
		return pkg.NewError("Debug", err.Error(), pkg.ExitBuild)
	}

	printDebugResult(result)
	return nil
}

func runDebugAll(debugger *debug.Debugger) error {
	results, err := tui.RunWithSpinner("Debug de tous les modules", func() ([]*debug.DebugResult, error) {
		return debugger.DebugAll()
	})
	if err != nil {
		return pkg.NewError("Debug", err.Error(), pkg.ExitBuild)
	}

	s := tui.NewStyles()
	fmt.Println()

	t := tui.NewTable([]string{"Module", "Statut", "Erreurs"})
	for _, r := range results {
		status := s.Success.Render("✓ " + r.Status)
		if r.Status == "ERREUR" {
			status = s.Error.Render("✗ " + r.Status)
		} else if r.Status == "AVERTISSEMENT" {
			status = s.Warning.Render("⚠ " + r.Status)
		}
		t.AddRow(r.Module, status, fmt.Sprintf("%d", r.Errors))
	}

	fmt.Print(t.Render())

	// Print logs for modules with errors
	for _, r := range results {
		if len(r.Logs) > 0 {
			fmt.Println(s.SubHeader.Render("  Logs de " + r.Module + ":"))
			logs := debug.FormatDebugLogs(r.Module, r.Logs)
			for _, log := range logs {
				fmt.Println(s.Muted.Render("  " + log))
			}
			fmt.Println()
		}
	}

	return nil
}

func printDebugResult(result *debug.DebugResult) {
	s := tui.NewStyles()
	fmt.Println()

	status := s.Success.Render("✓ " + result.Status)
	if result.Status == "ERREUR" {
		status = s.Error.Render("✗ " + result.Status)
	} else if result.Status == "AVERTISSEMENT" {
		status = s.Warning.Render("⚠ " + result.Status)
	}

	fmt.Printf("  %s : %s\n", s.Muted.Render("Module"), result.Module)
	fmt.Printf("  %s : %s\n", s.Muted.Render("Statut"), status)
	if result.Errors > 0 {
		fmt.Printf("  %s : %d\n", s.Muted.Render("Erreurs"), result.Errors)
	}

	if len(result.Logs) > 0 {
		fmt.Println()
		fmt.Println(s.SubHeader.Render("  Logs :"))
		logs := debug.FormatDebugLogs(result.Module, result.Logs)
		for _, log := range logs {
			fmt.Println(s.Muted.Render("  " + log))
		}
	}

	fmt.Println()
}
