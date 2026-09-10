package cmd

import (
	"fmt"
	"strings"

	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create module [nom]",
	Short: "Créer un nouveau module",
	Long: `Créer un nouveau module dans external_modules/ avec la structure
standardisée : manifest.json, index.tsx, components/, hooks/, services/.

Le token UUID unique du module est généré automatiquement.

Usage : sentient create module [nom]`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCreate(cmd, args)
	},
}

func runCreate(cmd *cobra.Command, args []string) error {
	// Accept both `create module <name>` and `create <name>`.
	if len(args) > 0 && args[0] == "module" {
		args = args[1:]
	}
	if len(args) > 1 {
		return pkg.NewError("Module", "trop d'arguments — utilisez 'sentient create module [nom]'", pkg.ExitError)
	}

	root, err := requireProjectRoot()
	if err != nil {
		return err
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		if !tui.IsInteractive() {
			return pkg.NewError("Critère de saisie", "fournissez le nom du module en argument", pkg.ExitError)
		}
		n, err := tui.AskText("Nom du module", "")
		if err != nil {
			return err
		}
		name = strings.TrimSpace(n)
	}
	if err := module.ValidateName(name); err != nil {
		return pkg.NewError("Module", err.Error(), pkg.ExitError)
	}

	description := ""
	if tui.IsInteractive() {
		d, err := tui.AskText("Description du module", "")
		if err != nil {
			return err
		}
		description = strings.TrimSpace(d)
	}

	creator := &module.Creator{Root: root}
	result, err := creator.Create(name, description)
	if err != nil {
		if strings.Contains(err.Error(), "existe déjà") {
			return pkg.NewError("Module", err.Error(), pkg.ExitError)
		}
		return err
	}

	s := tui.NewStyles()
	fmt.Println()
	fmt.Println(s.Success.Render(fmt.Sprintf("✓ Module créé : %s/", result.Dir)))
	fmt.Println(s.Success.Render("✓ Token généré : " + result.Token))
	fmt.Println(s.Success.Render("✓ manifest.json initialisé"))
	fmt.Println(s.Success.Render("✓ index.tsx initialisé"))
	fmt.Println()
	fmt.Println(s.SubHeader.Render("Prochaines étapes :"))
	fmt.Println(s.Info.Render("  sentient connect"))
	fmt.Println(s.Info.Render("  sentient pack " + result.Name))
	fmt.Println(s.Info.Render("  sentient publish"))
	return nil
}
