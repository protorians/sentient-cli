package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
)

// createTestModule sets up a valid module under a project root.
func createTestModule(t *testing.T, root, name string) {
	t.Helper()
	creator := &module.Creator{Root: root}
	if _, err := creator.Create(name, "Test module"); err != nil {
		t.Fatalf("Creator.Create(%q): %v", name, err)
	}
}

func setupAuditProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	// Create the external_modules dir (Creator does this, but be explicit)
	os.MkdirAll(filepath.Join(root, config.ExternalModulesDir), 0o755)
	return root
}

func TestAuditCleanModule(t *testing.T) {
	root := setupAuditProject(t)
	createTestModule(t, root, "blog-manager")

	auditor := &Auditor{Root: root}
	result, err := auditor.AuditModules("blog-manager")
	if err != nil {
		t.Fatalf("AuditModules: %v", err)
	}
	if len(result.Modules) != 1 {
		t.Fatalf("attendu 1 module, reçu %d", len(result.Modules))
	}
	if result.Modules[0].HasErrors() {
		t.Errorf("module propre ne doit pas avoir d'erreurs: %v", result.Modules[0].Findings)
	}
	if result.TotalErrors() != 0 {
		t.Errorf("TotalErrors = %d, want 0", result.TotalErrors())
	}
}

func TestAuditMissingModule(t *testing.T) {
	root := setupAuditProject(t)
	auditor := &Auditor{Root: root}
	_, err := auditor.AuditModules("nonexistent")
	if err == nil {
		t.Error("AuditModules d'un module absent doit échouer")
	}
}

func TestAuditAllModules(t *testing.T) {
	root := setupAuditProject(t)
	createTestModule(t, root, "mod-alpha")
	createTestModule(t, root, "mod-beta")

	auditor := &Auditor{Root: root}
	result, err := auditor.AuditModules("")
	if err != nil {
		t.Fatalf("AuditModules(all): %v", err)
	}
	if len(result.Modules) != 2 {
		t.Errorf("attendu 2 modules, reçu %d", len(result.Modules))
	}
}

func TestAuditAllSkipsDirsWithoutManifest(t *testing.T) {
	root := setupAuditProject(t)
	createTestModule(t, root, "real-module")
	// Create a dir without manifest.json
	os.MkdirAll(filepath.Join(root, config.ExternalModulesDir, "empty-dir"), 0o755)

	auditor := &Auditor{Root: root}
	result, err := auditor.AuditModules("")
	if err != nil {
		t.Fatalf("AuditModules: %v", err)
	}
	if len(result.Modules) != 1 {
		t.Errorf("attendu 1 module (skip empty dirs), reçu %d", len(result.Modules))
	}
}

func TestAuditAllMissingDir(t *testing.T) {
	root := t.TempDir() // no external_modules
	auditor := &Auditor{Root: root}
	_, err := auditor.AuditModules("")
	if err == nil {
		t.Error("AuditModules sans external_modules/ doit échouer")
	}
}

func TestAuditArchitectureComponentsImportServices(t *testing.T) {
	root := setupAuditProject(t)
	createTestModule(t, root, "bad-module")

	// Write a component that imports from services
	compDir := filepath.Join(root, config.ExternalModulesDir, "bad-module", "components")
	compFile := filepath.Join(compDir, "Widget.tsx")
	os.WriteFile(compFile, []byte(`import { fetchData } from "../services/api";`), 0o644)

	auditor := &Auditor{Root: root}
	result, err := auditor.AuditModules("bad-module")
	if err != nil {
		t.Fatalf("AuditModules: %v", err)
	}

	found := false
	for _, f := range result.Modules[0].Findings {
		if f.Category == "Clean Architecture" && f.Rule == "components→services" {
			found = true
			break
		}
	}
	if !found {
		t.Error("audit doit détecter l'import components→services")
	}
}

func TestAuditArchitectureServicesJSX(t *testing.T) {
	root := setupAuditProject(t)
	createTestModule(t, root, "jsx-service")

	// Write a service file containing JSX
	svcDir := filepath.Join(root, config.ExternalModulesDir, "jsx-service", "services")
	svcFile := filepath.Join(svcDir, "api.ts")
	os.WriteFile(svcFile, []byte(`// @jsx react
const el = <div>hello</div>;
// @tsx
`), 0o644)

	auditor := &Auditor{Root: root}
	result, err := auditor.AuditModules("jsx-service")
	if err != nil {
		t.Fatalf("AuditModules: %v", err)
	}

	found := false
	for _, f := range result.Modules[0].Findings {
		if f.Category == "Clean Architecture" && f.Rule == "services→JSX" {
			found = true
			break
		}
	}
	if !found {
		t.Error("audit doit détecter le JSX dans les services")
	}
}

func TestAuditRequirementsMissing(t *testing.T) {
	root := setupAuditProject(t)
	createTestModule(t, root, "dependent")

	// Add a requirement that doesn't exist
	manifestPath := filepath.Join(root, config.ExternalModulesDir, "dependent", "manifest.json")
	manifest, err := module.LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest.Requirements = map[string]any{"nonexistent-module": true}
	manifest.Save(manifestPath)

	auditor := &Auditor{Root: root}
	result, err := auditor.AuditModules("dependent")
	if err != nil {
		t.Fatalf("AuditModules: %v", err)
	}

	found := false
	for _, f := range result.Modules[0].Findings {
		if f.Category == "requirements" && f.Severity == module.LevelError {
			found = true
			break
		}
	}
	if !found {
		t.Error("audit doit détecter une requirement manquante")
	}
}

func TestAuditRequirementsPresent(t *testing.T) {
	root := setupAuditProject(t)
	createTestModule(t, root, "dep-a")
	createTestModule(t, root, "dep-b")

	// dep-b requires dep-a
	manifestPath := filepath.Join(root, config.ExternalModulesDir, "dep-b", "manifest.json")
	manifest, err := module.LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest.Requirements = map[string]any{"dep-a": true}
	manifest.Save(manifestPath)

	auditor := &Auditor{Root: root}
	result, err := auditor.AuditModules("dep-b")
	if err != nil {
		t.Fatalf("AuditModules: %v", err)
	}

	for _, f := range result.Modules[0].Findings {
		if f.Category == "requirements" && f.Rule == "dep-a" {
			if f.Severity != module.LevelOK {
				t.Errorf("requirement existante doit être OK, reçu %s", f.Severity)
			}
			return
		}
	}
	t.Error("finding pour dep-a requirement attendu")
}

func TestAuditAssetsEmpty(t *testing.T) {
	root := setupAuditProject(t)
	createTestModule(t, root, "with-assets")

	// Create empty assets directory
	assetsDir := config.ModuleAssetsDir(root, "with-assets")
	os.MkdirAll(assetsDir, 0o755)

	auditor := &Auditor{Root: root}
	result, err := auditor.AuditModules("with-assets")
	if err != nil {
		t.Fatalf("AuditModules: %v", err)
	}

	found := false
	for _, f := range result.Modules[0].Findings {
		if f.Category == "assets" {
			found = true
			if f.Severity != module.LevelWarning {
				t.Errorf("assets vides doit être WARNING, reçu %s", f.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("finding pour assets attendu")
	}
}

func TestAuditAssetsWithContent(t *testing.T) {
	root := setupAuditProject(t)
	createTestModule(t, root, "with-content")

	assetsDir := config.ModuleAssetsDir(root, "with-content")
	os.MkdirAll(assetsDir, 0o755)
	pkg.WriteFile(filepath.Join(assetsDir, "icon.png"), []byte("fake-png"))

	auditor := &Auditor{Root: root}
	result, err := auditor.AuditModules("with-content")
	if err != nil {
		t.Fatalf("AuditModules: %v", err)
	}

	for _, f := range result.Modules[0].Findings {
		if f.Category == "assets" {
			if f.Severity != module.LevelOK {
				t.Errorf("assets avec contenu doit être OK, reçu %s", f.Severity)
			}
			return
		}
	}
}

func TestAuditResultTotals(t *testing.T) {
	root := setupAuditProject(t)
	createTestModule(t, root, "mod-a")
	createTestModule(t, root, "mod-b")

	auditor := &Auditor{Root: root}
	result, err := auditor.AuditModules("")
	if err != nil {
		t.Fatalf("AuditModules: %v", err)
	}
	// Two clean modules: no errors, no warnings
	if result.TotalErrors() != 0 {
		t.Errorf("TotalErrors = %d, want 0", result.TotalErrors())
	}
	if result.TotalWarnings() != 0 {
		t.Errorf("TotalWarnings = %d, want 0", result.TotalWarnings())
	}
}

func TestAuditIndexAsyncRender(t *testing.T) {
	root := setupAuditProject(t)
	createTestModule(t, root, "no-async")

	// Overwrite index.tsx to remove async
	indexPath := filepath.Join(root, config.ExternalModulesDir, "no-async", "index.tsx")
	os.WriteFile(indexPath, []byte(`export default { name: "test" };`), 0o644)

	auditor := &Auditor{Root: root}
	result, err := auditor.AuditModules("no-async")
	if err != nil {
		t.Fatalf("AuditModules: %v", err)
	}

	found := false
	for _, f := range result.Modules[0].Findings {
		if f.Category == "index.tsx" && f.Rule == "render" {
			found = true
			if f.Severity != module.LevelError {
				t.Errorf("render non async doit être ERROR, reçu %s", f.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("finding pour render async attendu")
	}
}
