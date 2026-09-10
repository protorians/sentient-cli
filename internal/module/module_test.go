package module

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupProject creates a project root with one module via the Creator.
func setupProject(t *testing.T) (root string, creator *Creator) {
	t.Helper()
	root = t.TempDir()
	creator = &Creator{Root: root}
	return root, creator
}

func TestCreateModuleStructure(t *testing.T) {
	root, creator := setupProject(t)
	res, err := creator.Create("blog-manager", "Gestion de blog")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Token == "" {
		t.Error("Token vide")
	}

	expected := []string{
		"manifest.json",
		"index.tsx",
		"README.md",
		"components/.gitkeep",
		"hooks/.gitkeep",
		"services/.gitkeep",
	}
	for _, rel := range expected {
		p := filepath.Join(root, "external_modules", "blog-manager", filepath.FromSlash(rel))
		if _, err := os.Stat(p); err != nil {
			t.Errorf("fichier attendu manquant : %s (%v)", p, err)
		}
	}
}

func TestCreateModuleDuplicate(t *testing.T) {
	_ = t.TempDir()
	_, creator := setupProject(t)
	if _, err := creator.Create("blog", ""); err != nil {
		t.Fatalf("premier Create: %v", err)
	}
	if _, err := creator.Create("blog", ""); err == nil {
		t.Error("second Create doit échouer sur un module existant")
	}
}

func TestArchiveStructure(t *testing.T) {
	root, creator := setupProject(t)
	if _, err := creator.Create("blog-manager", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// assets dir
	assetPath := filepath.Join(root, "public", "assets", "blog-manager", "logo.svg")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	packer := &Packer{Root: root}
	res, err := packer.Pack("blog-manager")
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if res.Size == 0 {
		t.Error("archive vide")
	}

	zr, err := zip.OpenReader(res.Path)
	if err != nil {
		t.Fatalf("ouverture de l'archive: %v", err)
	}
	defer zr.Close()

	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
		if strings.HasSuffix(f.Name, "logo.svg") {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("lecture logo.svg: %v", err)
			}
			data, _ := io.ReadAll(rc)
			rc.Close()
			if string(data) != "<svg/>" {
				t.Errorf("contenu asset incorrect: %q", data)
			}
		}
	}

	joined := strings.Join(names, "\n")
	for _, want := range []string{
		"external_modules/blog-manager/manifest.json",
		"external_modules/blog-manager/index.tsx",
		"public/assets/blog-manager/logo.svg",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("l'archive doit contenir %q (contenu: %s)", want, joined)
		}
	}
}

func TestPackRejectsInvalidModule(t *testing.T) {
	root := t.TempDir()
	packer := &Packer{Root: root}
	if _, err := packer.Pack("absent"); err == nil {
		t.Error("Pack d'un module inexistant doit échouer")
	}
}

func TestValidateModuleOnCleanModule(t *testing.T) {
	root, creator := setupProject(t)
	if _, err := creator.Create("blog-manager", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	v := &Validator{Root: root}
	res, err := v.ValidateModule("blog-manager")
	if err != nil {
		t.Fatalf("ValidateModule: %v", err)
	}
	if res.HasErrors() {
		t.Errorf("module propre doit être valide, erreurs: %v", res.Findings)
	}
}

func TestValidateModuleMissingToken(t *testing.T) {
	root, creator := setupProject(t)
	if _, err := creator.Create("blog-manager", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Invalider le token
	m, err := LoadManifest(filepath.Join(root, "external_modules", "blog-manager", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	m.Token = "pas-un-uuid"
	if err := m.Save(filepath.Join(root, "external_modules", "blog-manager", "manifest.json")); err != nil {
		t.Fatal(err)
	}

	v := &Validator{Root: root}
	res, err := v.ValidateModule("blog-manager")
	if err != nil {
		t.Fatalf("ValidateModule: %v", err)
	}
	if !res.HasErrors() {
		t.Error("token invalide doit produire une erreur")
	}
}
