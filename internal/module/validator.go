package module

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/pkg"
)

// Severity levels for validation findings.
const (
	LevelError   = "ERROR"
	LevelWarning = "WARNING"
	LevelOK      = "OK"
)

// Finding is a single validation result.
type Finding struct {
	Category string `json:"category"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// Validator validates a module against the Sentient rules.
type Validator struct {
	Root string
}

// Result aggregates validation findings for one module.
type Result struct {
	Module   string    `json:"module"`
	Findings []Finding `json:"findings"`
}

// HasErrors reports whether any ERROR finding exists.
func (r *Result) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == LevelError {
			return true
		}
	}
	return false
}

// ErrorCount returns the number of ERROR findings.
func (r *Result) ErrorCount() int {
	return count(r.Findings, LevelError)
}

// WarningCount returns the number of WARNING findings.
func (r *Result) WarningCount() int {
	return count(r.Findings, LevelWarning)
}

func count(findings []Finding, level string) int {
	n := 0
	for _, f := range findings {
		if f.Severity == level {
			n++
		}
	}
	return n
}

// ValidateModule checks the core requirements of a single module.
// Used by `pack` and shared by the audit pipeline.
func (v *Validator) ValidateModule(name string) (*Result, error) {
	moduleDir := filepath.Join(v.Root, config.ExternalModulesDir, name)
	if !pkg.DirExists(moduleDir) {
		return nil, fmt.Errorf("module %q introuvable dans %s", name, moduleDir)
	}

	res := &Result{Module: name}

	manifestPath := filepath.Join(moduleDir, config.ManifestFileName)
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		res.Findings = append(res.Findings, Finding{
			Category: "manifest.json", Rule: "lecture",
			Severity: LevelError, Message: err.Error(),
		})
		return res, nil
	}

	// id
	add(res, "manifest.json", "id", manifest.ID != "", "id présent")
	// name
	add(res, "manifest.json", "name", manifest.Name != "", "name présent")
	// version (semver)
	add(res, "manifest.json", "version", isSemver(manifest.Version), "version SemVer valide")
	// token (UUID)
	add(res, "manifest.json", "token", pkg.IsUUID(manifest.Token), "token UUID valide")
	// entry exists
	entryPath := filepath.Join(moduleDir, manifest.Entry)
	add(res, "manifest.json", "entry", pkg.FileExists(entryPath), "fichier d'entrée présent")
	// domain format warning
	expectedDomain := "mod.sentients." + name
	addLevel(res, "manifest.json", "domain", manifest.Domain == expectedDomain,
		"domain au format mod.sentients.<name>", LevelWarning)
	// entry default export present in index.tsx
	indexPath := filepath.Join(moduleDir, config.ModuleEntryFileName)
	addLevel(res, "index.tsx", "export", pkg.FileExists(indexPath) && containsDefaultExport(indexPath),
		"fichier index.tsx avec export par défaut", LevelError)

	return res, nil
}

func add(res *Result, category, rule string, ok bool, okMsg string) {
	sev := LevelOK
	msg := okMsg
	if !ok {
		sev = LevelError
		msg = rule + " invalide"
	}
	res.Findings = append(res.Findings, Finding{Category: category, Rule: rule, Severity: sev, Message: msg})
}

func addLevel(res *Result, category, rule string, ok bool, okMsg string, warningSev string) {
	sev := LevelOK
	msg := okMsg
	if !ok {
		sev = warningSev
		msg = rule + " non conforme"
	}
	res.Findings = append(res.Findings, Finding{Category: category, Rule: rule, Severity: sev, Message: msg})
}

var semverRE = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

func isSemver(v string) bool {
	return semverRE.MatchString(v)
}

var defaultExportRE = regexp.MustCompile(`export\s+default\s+declaration`)

func containsDefaultExport(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return defaultExportRE.Match(data)
}
