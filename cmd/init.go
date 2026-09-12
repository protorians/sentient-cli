package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/tui"
	"github.com/spf13/cobra"
)

// templateRepo is the repository cloned by `sentients init` (spec FR-002).
const templateRepo = "https://github.com/protorians/sentient-cms"

// packageManagers is the detection + install order for `sentients init`.
var packageManagers = []struct {
	name       string
	installCmd []string
}{
	{name: "bun", installCmd: []string{"bun", "install"}},
	{name: "pnpm", installCmd: []string{"pnpm", "install"}},
	{name: "yarn", installCmd: []string{"yarn", "install"}},
	{name: "npm", installCmd: []string{"npm", "install"}},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialiser un nouveau projet Sentient",
	Long: `Initialise un nouveau projet Sentient en clonant le template
protorians/sentient-cms puis en installant les dépendances.

Le gestionnaire de paquets est détecté automatiquement (bun, pnpm, yarn, npm)
et proposé à l'utilisateur.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(cmd, args)
	},
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := ""
	if len(args) > 0 {
		projectName = args[0]
	}

	// Step 1 — project name
	if projectName == "" {
		if !tui.IsInteractive() {
			projectName = "sentient-cms"
		} else {
			defaultName := defaultProjectName()
			name, err := tui.AskText("Nom du projet", defaultName)
			if err != nil {
				return err
			}
			projectName = strings.TrimSpace(name)
			if projectName == "" {
				projectName = defaultName
			}
		}
	}

	targetDir := projectName
	if info, err := os.Stat(targetDir); err == nil {
		if !info.IsDir() {
			return pkg.NewError("Projet", fmt.Sprintf("%s existe et n'est pas un dossier", targetDir), pkg.ExitError)
		}
		if !tui.IsInteractive() {
			return pkg.NewError("Projet", fmt.Sprintf("le dossier %q existe déjà", targetDir), pkg.ExitError)
		}
		overwrite, err := tui.Confirm(fmt.Sprintf("Le dossier %q existe déjà, l'écraser ?", targetDir), false)
		if err != nil {
			return err
		}
		if !overwrite {
			return pkg.NewErrorWithFix("Projet", fmt.Sprintf("le dossier %q existe déjà", targetDir),
				"Choisissez un autre nom de projet ou déplacez le dossier existant.", pkg.ExitError)
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return pkg.NewError("Projet", "suppression du dossier existant impossible", pkg.ExitError)
		}
	}

	// Step 2 — package manager detection (spec FR-001)
	available := make([]string, 0, len(packageManagers))
	for _, pm := range packageManagers {
		if pkg.HasCommand(pm.name) {
			available = append(available, pm.name)
		}
	}
	if len(available) == 0 {
		return pkg.NewErrorWithFix(
			"Gestionnaire de paquets",
			"aucun gestionnaire de paquets détecté (bun, pnpm, yarn, npm)",
			"Installez bun, pnpm, yarn ou npm puis réessayez.",
			pkg.ExitError,
		)
	}

	pmName := available[0] // bun is first; recommended default
	if len(available) > 1 {
		if !tui.IsInteractive() {
			pmName = available[0]
		} else {
			var items []string
			for i, name := range available {
				if i == 0 {
					items = append(items, name+" (recommandé)")
				} else {
					items = append(items, name)
				}
			}
			selected, err := tui.Select("Gestionnaire de paquets", items)
			if err != nil {
				return err
			}
			pmName = strings.TrimSuffix(selected, " (recommandé)")
		}
	}
	debugf("gestionnaire de paquets choisi : %s", pmName)

	var installCmd []string
	for _, pm := range packageManagers {
		if pm.name == pmName {
			installCmd = pm.installCmd
			break
		}
	}
	if installCmd == nil {
		return pkg.NewError("Gestionnaire de paquets", "gestionnaire inconnu : "+pmName, pkg.ExitError)
	}

	// Step 3 — clone (shallow)
	if _, err := tui.RunWithSpinner("Clonage de sentient-cms", func() (struct{}, error) {
		return struct{}{}, pkg.CloneShallow(templateRepo, targetDir)
	}); err != nil {
		return pkg.NewErrorWithFix("Réseau", err.Error(),
			"Vérifiez votre connexion et que 'git' est installé.", pkg.ExitNetwork)
	}

	// Step 4 — install dependencies
	if _, err := tui.RunWithSpinner("Installation des dépendances", func() (struct{}, error) {
		return struct{}{}, runInstall(targetDir, installCmd)
	}); err != nil {
		warn("L'installation des dépendances a échoué : " + err.Error())
	}

	// Step 5 — write .sentient-cli.toml
	cfg := config.Default()
	cfg.Project.Name = projectName
	cfg.Project.PackageManager = pmName
	if err := cfg.Save(filepath.Join(targetDir, config.ConfigFileName)); err != nil {
		debugf("écriture du fichier de configuration : %v", err)
	}

	// Step 6 — summary
	s := tui.NewStyles()
	fmt.Println()
	fmt.Println(s.Success.Render("✓ Projet initialisé avec succès"))
	fmt.Println(s.Muted.Render("  Gestionnaire : " + pmName))
	fmt.Println()
	fmt.Println(s.SubHeader.Render("Prochaines étapes :"))
	fmt.Printf("  %s\n", s.Info.Render("cd "+targetDir))
	fmt.Printf("  %s\n", s.Info.Render("sentients connect"))
	fmt.Printf("  %s\n", s.Info.Render("sentients create module"))
	return nil
}

func defaultProjectName() string {
	if cwd, err := os.Getwd(); err == nil {
		if base := filepath.Base(cwd); base != "" && base != "/" && base != "." {
			return base
		}
	}
	return "sentient-cms"
}

func runInstall(dir string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("commande d'installation invalide")
	}
	return pkg.StreamCommandIn(dir, args[0], args[1:]...)
}
