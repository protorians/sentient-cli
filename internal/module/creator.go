package module

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/pkg"
)

// Creator builds new modules with the standardised structure.
type Creator struct {
	Root string
}

// CreateResult summarises a module creation.
type CreateResult struct {
	Name     string
	Dir      string
	Token    string
	Manifest *Manifest
}

// Description is the optional human-readable description of the module.
type Description = string

// Indices are the empty sub-directories created inside each module.
var moduleSubDirs = []string{"components", "hooks", "services"}

// Create generates a module named `name` inside `external_modules/`.
// When `description` is empty, a description is not written to the manifest.
func (c *Creator) Create(name, description string) (*CreateResult, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	moduleDir := filepath.Join(c.Root, config.ExternalModulesDir, name)
	if pkg.DirExists(moduleDir) {
		return nil, fmt.Errorf("le module %q existe déjà dans %s", name, moduleDir)
	}
	if err := pkg.CreateDir(moduleDir); err != nil {
		return nil, err
	}

	manifest := NewManifest(name, description)
	manifestPath := filepath.Join(moduleDir, config.ManifestFileName)
	if err := manifest.Save(manifestPath); err != nil {
		return nil, err
	}

	if err := pkg.WriteString(filepath.Join(moduleDir, config.ModuleEntryFileName), indexTemplate(name)); err != nil {
		return nil, err
	}
	if err := pkg.WriteString(filepath.Join(moduleDir, "README.md"), readmeTemplate(name, description)); err != nil {
		return nil, err
	}
	for _, sub := range moduleSubDirs {
		if err := pkg.WriteString(filepath.Join(moduleDir, sub, ".gitkeep"), ""); err != nil {
			return nil, err
		}
	}

	return &CreateResult{
		Name:     name,
		Dir:      moduleDir,
		Token:    manifest.Token,
		Manifest: &manifest,
	}, nil
}

func indexTemplate(name string) string {
	return fmt.Sprintf(`import type { ModuleDeclarationInterface } from "@/modules";

const declaration: ModuleDeclarationInterface = {
  name: "%s",
  description: "",
  render: async () => {
    const mod = await import("./components");
    return mod.default;
  },
};

export default declaration;
`, displayName(name))
}

func readmeTemplate(name, description string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\n\n", displayName(name)))
	if description != "" {
		b.WriteString(description + "\n\n")
	}
	b.WriteString(fmt.Sprintf("Module Sentient `%s`.", name))
	b.WriteString("\n\n## Structure\n\n")
	b.WriteString("- `manifest.json` — métadonnées du module\n")
	b.WriteString("- `index.tsx` — point d'entrée du module\n")
	b.WriteString("- `components/` — composants React\n")
	b.WriteString("- `hooks/` — hooks React\n")
	b.WriteString("- `services/` — services métier\n")
	return b.String()
}
