package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
)

// jsxTagRE heuristically recognises JSX elements without a full TS/JSX parser:
// closing tags (`</div>`), custom (capitalised) components (`<Foo …/>`) and
// lowercase native elements carrying attributes (`<div className=…/>`). It
// deliberately ignores type arguments such as `Array<string>`.
var jsxTagRE = regexp.MustCompile(`(?:</[A-Z][A-Za-z0-9._-]*>|</[a-z][a-z0-9_-]*>|<[A-Z][A-Za-z0-9._-]*(?:\s[^<>]*?)?/?>|<[a-z][a-z0-9_-]*(?:\s+[a-zA-Z-]+=)[^<>]*?/?>)`)

// Auditor runs all conformance checks on a module.
type Auditor struct {
	Root string
}

// AuditResult aggregates audit findings for one or more modules.
type AuditResult struct {
	Modules []module.Result `json:"modules"`
}

// TotalErrors returns the total error count across all modules.
func (ar *AuditResult) TotalErrors() int {
	n := 0
	for _, m := range ar.Modules {
		n += m.ErrorCount()
	}
	return n
}

// TotalWarnings returns the total warning count across all modules.
func (ar *AuditResult) TotalWarnings() int {
	n := 0
	for _, m := range ar.Modules {
		n += m.WarningCount()
	}
	return n
}

// AuditModules audits one or all modules. When name is empty, all modules
// in external_modules/ are audited.
func (a *Auditor) AuditModules(name string) (*AuditResult, error) {
	if name != "" {
		return a.auditSingle(name)
	}
	return a.auditAll()
}

func (a *Auditor) auditSingle(name string) (*AuditResult, error) {
	moduleDir := filepath.Join(a.Root, config.ExternalModulesDir, name)
	if !pkg.DirExists(moduleDir) {
		return nil, fmt.Errorf("module %q introuvable dans %s", name, config.ExternalModulesDir)
	}
	res, err := a.auditModule(name)
	if err != nil {
		return nil, err
	}
	return &AuditResult{Modules: []module.Result{*res}}, nil
}

func (a *Auditor) auditAll() (*AuditResult, error) {
	dir := filepath.Join(a.Root, config.ExternalModulesDir)
	if !pkg.DirExists(dir) {
		return nil, fmt.Errorf("le dossier %q est introuvable", config.ExternalModulesDir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s impossible : %w", config.ExternalModulesDir, err)
	}

	result := &AuditResult{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !pkg.FileExists(filepath.Join(dir, e.Name(), config.ManifestFileName)) {
			continue
		}
		res, err := a.auditModule(e.Name())
		if err != nil {
			return nil, err
		}
		result.Modules = append(result.Modules, *res)
	}
	return result, nil
}

// auditModule runs the full audit pipeline on a single module.
func (a *Auditor) auditModule(name string) (*module.Result, error) {
	v := &module.Validator{Root: a.Root}
	res, err := v.ValidateModule(name)
	if err != nil {
		return nil, err
	}

	moduleDir := filepath.Join(a.Root, config.ExternalModulesDir, name)
	indexPath := filepath.Join(moduleDir, config.ModuleEntryFileName)

	a.auditArchitecture(moduleDir, indexPath, res)
	a.auditDependencies(name, res)
	a.auditAssets(name, res)

	return res, nil
}

// auditArchitecture checks Clean Architecture rules.
func (a *Auditor) auditArchitecture(moduleDir, indexPath string, res *module.Result) {
	// Check: components don't import services directly
	componentsDir := filepath.Join(moduleDir, "components")
	if pkg.DirExists(componentsDir) {
		_ = filepath.Walk(componentsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			content := string(data)
			if strings.Contains(content, "from \"../services") || strings.Contains(content, "from '../services") ||
				strings.Contains(content, "from \"./services") || strings.Contains(content, "from './services") {
				rel, _ := filepath.Rel(moduleDir, path)
				res.Findings = append(res.Findings, module.Finding{
					Category: "Clean Architecture",
					Rule:     "components→services",
					Severity: module.LevelError,
					Message:  fmt.Sprintf("%s importe directement des services", rel),
				})
			}
			return nil
		})
	}

	// Check: services don't contain JSX
	servicesDir := filepath.Join(moduleDir, "services")
	if pkg.DirExists(servicesDir) {
		_ = filepath.Walk(servicesDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if jsxTagRE.Match(data) {
				rel, _ := filepath.Rel(moduleDir, path)
				res.Findings = append(res.Findings, module.Finding{
					Category: "Clean Architecture",
					Rule:     "services→JSX",
					Severity: module.LevelError,
					Message:  fmt.Sprintf("%s contient du JSX dans un service", rel),
				})
			}
			return nil
		})
	}

	// Check: index.tsx has async render function
	if pkg.FileExists(indexPath) {
		data, err := os.ReadFile(indexPath)
		if err == nil {
			content := string(data)
			hasRender := strings.Contains(content, "render")
			hasAsync := strings.Contains(content, "async")
			addLevel(res, "index.tsx", "render", hasRender && hasAsync,
				"render est asynchrone", module.LevelError)
		}
	}
}

// auditDependencies checks that listed requirements and npm deps exist.
func (a *Auditor) auditDependencies(name string, res *module.Result) {
	manifest, err := module.LoadManifest(config.ManifestPath(a.Root, name))
	if err != nil {
		return
	}

	// Check requirements
	for req := range manifest.Requirements {
		reqDir := filepath.Join(a.Root, config.ExternalModulesDir, req)
		addLevel(res, "requirements", req, pkg.DirExists(reqDir),
			fmt.Sprintf("requirement %q existe", req), module.LevelError)
	}

	// Check that listed npm dependencies are installed (spec rule
	// « Toutes les dépendances npm sont installées »). The previous duplicate
	// check iterated a map — duplicate keys are impossible by construction, so
	// it could never trigger. Installed-ness is the real, verifiable rule.
	for dep := range manifest.Dependencies {
		depDir := filepath.Join(a.Root, "node_modules", filepath.FromSlash(dep))
		addLevel(res, "dependencies", dep, pkg.DirExists(depDir),
			fmt.Sprintf("dépendance %q installée", dep), module.LevelError)
	}
}

// auditAssets checks that assets referenced by the module exist.
func (a *Auditor) auditAssets(name string, res *module.Result) {
	assetsDir := config.ModuleAssetsDir(a.Root, name)
	if pkg.DirExists(assetsDir) {
		// Assets directory exists — check for files
		hasContent := false
		_ = filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				hasContent = true
				return filepath.SkipDir
			}
			return nil
		})
		addLevel(res, "assets", "fichiers", hasContent,
			"assets contiennent des fichiers", module.LevelWarning)
	}
}

func addLevel(res *module.Result, category, rule string, ok bool, okMsg string, failSev string) {
	sev := module.LevelOK
	msg := okMsg
	if !ok {
		sev = failSev
		msg = rule + " non conforme"
	}
	res.Findings = append(res.Findings, module.Finding{
		Category: category,
		Rule:     rule,
		Severity: sev,
		Message:  msg,
	})
}
