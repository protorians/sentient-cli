package cmd

import (
	"fmt"

	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

var packCmd = &cobra.Command{
	Use:   "pack [module]",
	Short: "Construire l'archive d'un module (.smp)",
	Long: `Construit le build d'un module et crée une archive .smp compressée
(déplacée vers .sentients/build/).

Sans argument, un sélecteur permet de choisir le module.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPack(cmd, args)
	},
}

func runPack(cmd *cobra.Command, args []string) error {
	root, err := requireProjectRoot()
	if err != nil {
		return err
	}

	name, err := resolveModule(root, args)
	if err != nil {
		return err
	}

	result, err := tui.RunWithSpinner("Construction de l'archive…", func() (*module.PackResult, error) {
		p := &module.Packer{Root: root}
		return p.Pack(name)
	})
	if err != nil {
		if _, ok := err.(*pkg.Error); ok {
			return err
		}
		return pkg.NewError("Pack", err.Error(), pkg.ExitBuild)
	}

	s := tui.NewStyles()
	fmt.Println()
	fmt.Println(s.Success.Render("✓ Archive créée avec succès"))
	fmt.Printf("  %s : %s v%s\n", s.Muted.Render("Module"), name, result.Version)
	fmt.Printf("  %s : %s\n", s.Muted.Render("Fichier"), s.Info.Render(result.Path))
	fmt.Printf("  %s : %s\n", s.Muted.Render("Taille"), humanSize(result.Size))
	return nil
}

// humanSize renders a byte count in a human-friendly way.
func humanSize(bytes int64) string {
	const kb = 1024
	const mb = kb * 1024
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
