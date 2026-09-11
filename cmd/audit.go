package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/protorians/sentient-cli/internal/audit"
	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

var auditOutput string

func init() {
	auditCmd.Flags().StringVar(&auditOutput, "output", "", "format de sortie (table | json)")
	_ = auditCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json"}, cobra.ShellCompDirectiveDefault
	})
}

// resolveAuditOutput maps the --output flag to the render mode, rejecting
// unknown values (spec §5.10 : `--output json` est le seul format machine).
func resolveAuditOutput() (string, error) {
	switch auditOutput {
	case "", "table":
		return "table", nil
	case "json":
		return "json", nil
	default:
		return "", pkg.NewErrorWithFix("Audit",
			fmt.Sprintf("format de sortie inconnu %q", auditOutput),
			"Formats valides : table, json.", pkg.ExitError)
	}
}

var auditCmd = &cobra.Command{
	Use:   "audit [module]",
	Short: "Auditer la conformité d'un module",
	Long: `Audite la conformité d'un (ou de tous les) module(s) par rapport aux
règles Sentient : Clean Architecture, manifest.json et index.tsx.

Sans argument, tous les modules de external_modules/ sont audités.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAudit(cmd, args)
	},
}

func runAudit(cmd *cobra.Command, args []string) error {
	root, err := requireProjectRoot()
	if err != nil {
		return err
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	auditor := &audit.Auditor{Root: root}

	result, err := tui.RunWithSpinner("Audit des modules…", func() (*audit.AuditResult, error) {
		return auditor.AuditModules(name)
	})
	if err != nil {
		return pkg.NewError("Audit", err.Error(), pkg.ExitError)
	}

	if auditOutput == "json" {
		return printAuditJSON(result)
	}

	printAuditResult(result)
	return nil
}

// printAuditJSON exports the audit report as JSON (spec §5.10).
func printAuditJSON(result *audit.AuditResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return pkg.NewError("Audit", "sérialisation JSON impossible : "+err.Error(), pkg.ExitError)
	}
	fmt.Println(string(data))
	return nil
}

func printAuditResult(result *audit.AuditResult) {
	s := tui.NewStyles()
	fmt.Println()

	for _, mod := range result.Modules {
		fmt.Println(s.SubHeader.Render("Audit : " + mod.Module))

		t := tui.NewTable([]string{"Catégorie", "Règle", "Statut"})
		for _, f := range mod.Findings {
			status := s.Success.Render("✓ " + f.Message)
			switch f.Severity {
			case module.LevelError:
				status = s.Error.Render("✗ " + f.Message)
			case module.LevelWarning:
				status = s.Warning.Render("⚠ " + f.Message)
			}
			t.AddRow(f.Category, f.Rule, status)
		}

		if len(mod.Findings) > 0 {
			fmt.Print(t.Render())
		} else {
			fmt.Println(s.Muted.Render("  Aucune vérification effectuée"))
		}
		fmt.Println()
	}

	errors := result.TotalErrors()
	warnings := result.TotalWarnings()

	if errors == 0 && warnings == 0 {
		fmt.Println(s.Success.Render("✓ Audit réussi — aucune erreur, aucun avertissement"))
	} else {
		parts := []string{}
		if errors > 0 {
			parts = append(parts, fmt.Sprintf("%d erreur(s)", errors))
		}
		if warnings > 0 {
			parts = append(parts, fmt.Sprintf("%d avertissement(s)", warnings))
		}
		header := "Résumé : "
		for i, p := range parts {
			if i > 0 {
				header += ", "
			}
			header += p
		}
		if errors > 0 {
			fmt.Fprintln(os.Stderr, s.Error.Render(header))
		} else {
			fmt.Println(s.Warning.Render(header))
		}
	}
}
