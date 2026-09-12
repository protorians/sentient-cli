package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/pkg"
	"github.com/protorians/sentient-cli/internal/tui"
)

// requireProjectRoot locates the current Sentient project root or returns a
// dedicated error (exit code 3 in French per spec).
func requireProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", pkg.NewError("Projet", "impossible de déterminer le répertoire courant", pkg.ExitError)
	}
	root, err := config.FindProjectRoot(cwd)
	if err != nil {
		return "", pkg.NewErrorWithFix(
			"Projet",
			err.Error(),
			"Exécutez 'sentients init' pour initialiser un projet Sentient.",
			pkg.ExitModuleNotFound,
		)
	}
	debugf("racine du projet : %s", root)
	return root, nil
}

// listModules returns the module names present in `external_modules/`.
func listModules(root string) ([]string, error) {
	dir := filepath.Join(root, config.ExternalModulesDir)
	if !pkg.DirExists(dir) {
		return nil, pkg.NewError(
			"Projet",
			fmt.Sprintf("le dossier %q est introuvable", config.ExternalModulesDir),
			pkg.ExitModuleNotFound,
		)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, pkg.NewError("Projet", "lecture de external_modules impossible", pkg.ExitError)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if pkg.FileExists(filepath.Join(dir, e.Name(), config.ManifestFileName)) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// resolveModule returns the module name to operate on: the positional arg if
// provided, otherwise a single-module selection or an interactive menu.
func resolveModule(root string, args []string) (string, error) {
	if len(args) > 0 {
		name := args[0]
		if !pkg.DirExists(filepath.Join(root, config.ExternalModulesDir, name)) {
			return "", pkg.NewErrorWithFix(
				"Module",
				fmt.Sprintf("le module %q n'existe pas dans %s", name, config.ExternalModulesDir),
				"Vérifiez le nom du module ou exécutez 'sentients create module'.",
				pkg.ExitModuleNotFound,
			)
		}
		return name, nil
	}

	modules, err := listModules(root)
	if err != nil {
		return "", err
	}
	if len(modules) == 0 {
		return "", pkg.NewError(
			"Module",
			"aucun module trouvé dans external_modules",
			pkg.ExitModuleNotFound,
		)
	}
	if len(modules) == 1 {
		return modules[0], nil
	}

	if !tui.IsInteractive() {
		return "", pkg.NewError(
			"Sélection",
			"plusieurs modules disponibles, fournissez le nom du module en argument",
			pkg.ExitError,
		)
	}

	return selectModule(modules)
}

func selectModule(modules []string) (string, error) {
	selected, err := tui.Select("Sélectionner le module", modules)
	if err != nil {
		return "", err
	}
	return selected, nil
}
