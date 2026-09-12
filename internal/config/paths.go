package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Well-known directory and file names within a Sentient project.
const (
	ConfigFileName      = ".sentient-cli.toml"
	SentientConfigName  = "sentient.config.toml"
	ExternalModulesDir  = "external_modules"
	PublicAssetsDir     = "public/assets"
	SentientBuildsDir   = ".sentients/build"
	ManifestFileName    = "manifest.json"
	ModuleEntryFileName = "index.tsx"
)

// IsProjectRoot reports whether dir looks like a Sentient project root.
func IsProjectRoot(dir string) bool {
	if fileExists(filepath.Join(dir, SentientConfigName)) {
		return true
	}
	if fileExists(filepath.Join(dir, ConfigFileName)) {
		return true
	}
	if dirExists(filepath.Join(dir, ExternalModulesDir)) {
		return true
	}
	return false
}

// FindProjectRoot walks up from start (default: current directory) looking
// for a Sentient project root. Returns an error when none is found.
func FindProjectRoot(start string) (string, error) {
	dir := start
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("impossible de déterminer le répertoire courant : %w", err)
		}
	}
	for {
		if IsProjectRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("aucun projet Sentient trouvé (recherche de %q ou %q) — exécutez 'sentients init'",
				SentientConfigName, ExternalModulesDir)
		}
		dir = parent
	}
}

// Absolute returns path joined to root when root is non-empty.
func Absolute(root, path string) string {
	return filepath.Join(root, path)
}

// ConfigPath returns the CLI config file path for a project root.
func ConfigPath(root string) string {
	return filepath.Join(root, ConfigFileName)
}

// ModuleDir returns the directory of a module.
func ModuleDir(root, module string) string {
	return filepath.Join(root, ExternalModulesDir, module)
}

// ModuleAssetsDir returns the assets directory of a module.
func ModuleAssetsDir(root, module string) string {
	return filepath.Join(root, PublicAssetsDir, module)
}

// BuildDir returns the `.sentients/build/` directory for a project root.
func BuildDir(root string) string {
	return filepath.Join(root, SentientBuildsDir)
}

// ManifestPath returns the path of a module manifest.
func ManifestPath(root, module string) string {
	return filepath.Join(ModuleDir(root, module), ManifestFileName)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
