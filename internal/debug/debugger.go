package debug

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
)

// DebugResult holds the outcome of a debug run for a single module.
type DebugResult struct {
	Module string
	Status string
	Errors int
	Logs   []string
}

// Debugger runs debug builds for modules.
type Debugger struct {
	Root string
}

// DebugModule runs a debug build of a single module.
func (d *Debugger) DebugModule(name string) (*DebugResult, error) {
	moduleDir := filepath.Join(d.Root, config.ExternalModulesDir, name)
	if !pkg.DirExists(moduleDir) {
		return nil, fmt.Errorf("module %q introuvable dans %s", name, config.ExternalModulesDir)
	}

	result := &DebugResult{Module: name}

	// Validate first
	v := &module.Validator{Root: d.Root}
	res, err := v.ValidateModule(name)
	if err != nil {
		return nil, err
	}

	if res.HasErrors() {
		result.Status = "ERREUR"
		result.Errors = res.ErrorCount()
		for _, f := range res.Findings {
			if f.Severity == module.LevelError {
				result.Logs = append(result.Logs, fmt.Sprintf("[%s] %s", f.Rule, f.Message))
			}
		}
		return result, nil
	}

	// Try to run the build command if available
	pm := d.detectPackageManager()
	if pm == "" {
		result.Status = "AVERTISSEMENT"
		result.Logs = append(result.Logs, "aucun gestionnaire de paquets détecté, validation seule effectuée")
		return result, nil
	}

	// Look for a debug or build script in the project
	buildCmd := d.findBuildCommand(pm)
	if buildCmd != nil {
		output, err := d.runCommand(moduleDir, buildCmd)
		if err != nil {
			result.Status = "ERREUR"
			result.Errors++
			result.Logs = append(result.Logs, fmt.Sprintf("erreur de compilation: %s", err.Error()))
			if output != "" {
				result.Logs = append(result.Logs, output)
			}
		} else {
			result.Status = "OK"
			if output != "" {
				result.Logs = append(result.Logs, output)
			}
		}
	} else {
		// No build command, just validate
		result.Status = "OK"
		result.Logs = append(result.Logs, "validation du manifest et de la structure OK")
	}

	return result, nil
}

// DebugAll runs debug on all modules in external_modules/.
func (d *Debugger) DebugAll() ([]*DebugResult, error) {
	dir := filepath.Join(d.Root, config.ExternalModulesDir)
	if !pkg.DirExists(dir) {
		return nil, fmt.Errorf("le dossier %q est introuvable", config.ExternalModulesDir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s impossible : %w", config.ExternalModulesDir, err)
	}

	var results []*DebugResult
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !pkg.FileExists(filepath.Join(dir, e.Name(), config.ManifestFileName)) {
			continue
		}
		r, err := d.DebugModule(e.Name())
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func (d *Debugger) detectPackageManager() string {
	for _, pm := range []string{"bun", "pnpm", "yarn", "npm"} {
		if _, err := exec.LookPath(pm); err == nil {
			return pm
		}
	}
	return ""
}

func (d *Debugger) findBuildCommand(pm string) []string {
	// Check for common build/debug scripts
	scripts := []string{"debug", "dev", "build"}
	for _, script := range scripts {
		packageJson := filepath.Join(d.Root, "package.json")
		if pkg.FileExists(packageJson) {
			data, err := os.ReadFile(packageJson)
			if err == nil {
				content := string(data)
				if strings.Contains(content, fmt.Sprintf(`"%s"`, script)) {
					return []string{pm, "run", script}
				}
			}
		}
	}
	return nil
}

func (d *Debugger) runCommand(dir string, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("commande vide")
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(output)), err
	}
	return strings.TrimSpace(string(output)), nil
}

// FormatDebugLogs returns formatted log lines with timestamps.
func FormatDebugLogs(moduleName string, logs []string) []string {
	result := make([]string, 0, len(logs))
	for _, log := range logs {
		t := time.Now().Format("15:04:05")
		result = append(result, fmt.Sprintf("[%s] %s: %s", t, moduleName, log))
	}
	return result
}
