package debug

import (
	"encoding/json"
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

	// Look for a debug or build script, preferring the module's own
	// package.json over the project root one.
	build := d.findBuildCommand(pm, moduleDir)
	if build != nil {
		output, err := d.runCommand(build.dir, build.cmd)
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
		// No build script: the validation above is the only thing executed —
		// this must not be reported as a successful build (previously a false
		// "OK" hid the absence of any real compilation).
		result.Status = "AVERTISSEMENT"
		result.Logs = append(result.Logs,
			"aucun script de build (debug/dev/build) trouvé — validation seule effectuée")
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

// buildCommand is a build script to run and the directory it belongs to.
type buildCommand struct {
	cmd []string
	dir string
}

func (d *Debugger) detectPackageManager() string {
	for _, pm := range []string{"bun", "pnpm", "yarn", "npm"} {
		if _, err := exec.LookPath(pm); err == nil {
			return pm
		}
	}
	return ""
}

// findBuildCommand searches for a debug/dev/build script, first in the module's
// own package.json (if any), then at the project root. The scripts map is
// parsed as JSON so a substring collision (e.g. `"build:prod"`) does not
// trigger a match.
func (d *Debugger) findBuildCommand(pm, moduleDir string) *buildCommand {
	scripts := []string{"debug", "dev", "build"}
	candidates := []string{
		filepath.Join(moduleDir, "package.json"),
		filepath.Join(d.Root, "package.json"),
	}
	for _, candidate := range candidates {
		dir := filepath.Dir(candidate)
		for _, script := range scripts {
			if _, ok := packageScript(candidate, script); ok {
				return &buildCommand{cmd: []string{pm, "run", script}, dir: dir}
			}
		}
	}
	return nil
}

// packageScript returns the value of a package.json script if present.
func packageScript(path, name string) (string, bool) {
	if !pkg.FileExists(path) {
		return "", false
	}
	var nodePackage struct {
		Scripts map[string]string `json:"scripts"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	if err := json.Unmarshal(data, &nodePackage); err != nil {
		return "", false
	}
	v, ok := nodePackage.Scripts[name]
	return v, ok
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
