package cmd

import (
	"fmt"
	"os"

	"github.com/protorians/sentient-cli/internal/audit"
	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

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

	printAuditResult(result)
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
