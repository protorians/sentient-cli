package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/module"
	"github.com/protorians/sentient-cli/internal/pkg"
)

func TestHumanSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tt := range tests {
		got := humanSize(tt.bytes)
		if got != tt.want {
			t.Errorf("humanSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestListModules(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, config.ExternalModulesDir)
	os.MkdirAll(extDir, 0o755)

	// Create two modules
	creator := &module.Creator{Root: root}
	if _, err := creator.Create("alpha-mod", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := creator.Create("beta-mod", ""); err != nil {
		t.Fatal(err)
	}

	// Create a dir without manifest (should be skipped)
	os.MkdirAll(filepath.Join(extDir, "not-a-module"), 0o755)

	names, err := listModules(root)
	if err != nil {
		t.Fatalf("listModules: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("attendu 2 modules, reçu %d: %v", len(names), names)
	}
	// Should be sorted
	if names[0] != "alpha-mod" || names[1] != "beta-mod" {
		t.Errorf("modules non triés: %v", names)
	}
}

func TestListModulesEmpty(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, config.ExternalModulesDir), 0o755)

	names, err := listModules(root)
	if err != nil {
		t.Fatalf("listModules: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("attendu 0 modules, reçu %d", len(names))
	}
}

func TestListModulesMissingDir(t *testing.T) {
	root := t.TempDir()
	_, err := listModules(root)
	if err == nil {
		t.Error("listModules doit échouer sans external_modules/")
	}
}

func TestResolveModuleWithArg(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, config.ExternalModulesDir), 0o755)
	creator := &module.Creator{Root: root}
	if _, err := creator.Create("my-mod", ""); err != nil {
		t.Fatal(err)
	}

	name, err := resolveModule(root, []string{"my-mod"})
	if err != nil {
		t.Fatalf("resolveModule: %v", err)
	}
	if name != "my-mod" {
		t.Errorf("name = %q, want my-mod", name)
	}
}

func TestResolveModuleWithArgNotFound(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, config.ExternalModulesDir), 0o755)

	_, err := resolveModule(root, []string{"nonexistent"})
	if err == nil {
		t.Error("resolveModule doit échouer pour un module absent")
	}
}

func TestResolveModuleSingle(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, config.ExternalModulesDir), 0o755)
	creator := &module.Creator{Root: root}
	if _, err := creator.Create("solo-mod", ""); err != nil {
		t.Fatal(err)
	}

	// With no args and single module, should auto-select
	name, err := resolveModule(root, nil)
	if err != nil {
		t.Fatalf("resolveModule: %v", err)
	}
	if name != "solo-mod" {
		t.Errorf("name = %q, want solo-mod", name)
	}
}

func TestResolveModuleEmpty(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, config.ExternalModulesDir), 0o755)

	_, err := resolveModule(root, nil)
	if err == nil {
		t.Error("resolveModule doit échouer sans modules")
	}
}

func TestRequireProjectRoot(t *testing.T) {
	root := t.TempDir()
	// Create external_modules to make it a project root
	os.MkdirAll(filepath.Join(root, config.ExternalModulesDir), 0o755)

	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	found, err := requireProjectRoot()
	if err != nil {
		t.Fatalf("requireProjectRoot: %v", err)
	}
	// Resolve symlinks (macOS: /var -> /private/var)
	wantRoot, _ := filepath.EvalSymlinks(root)
	foundRoot, _ := filepath.EvalSymlinks(found)
	if foundRoot != wantRoot {
		t.Errorf("root = %q, want %q", found, root)
	}
}

func TestRequireProjectRootNotFound(t *testing.T) {
	root := t.TempDir()
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	_, err := requireProjectRoot()
	if err == nil {
		t.Error("requireProjectRoot doit échouer hors d'un projet")
	}
}

func TestRootCmdStructure(t *testing.T) {
	if rootCmd.Use != "sentient" {
		t.Errorf("rootCmd.Use = %q, want sentient", rootCmd.Use)
	}
	if !rootCmd.SilenceUsage {
		t.Error("rootCmd doit silencer l'usage")
	}
	if !rootCmd.SilenceErrors {
		t.Error("rootCmd doit silencer les erreurs")
	}
}

func TestRootCmdSubcommands(t *testing.T) {
	expected := []string{
		"init", "create", "connect", "disconnect", "pack",
		"sign", "publish", "link", "unlink", "debug", "audit",
	}
	for _, name := range expected {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("sous-commande %q non enregistrée", name)
		}
	}
}

func TestCreateCmdArgs(t *testing.T) {
	if createCmd.Use != "create module [nom]" {
		t.Errorf("createCmd.Use = %q", createCmd.Use)
	}
}

func TestPackCmdArgs(t *testing.T) {
	if packCmd.Use != "pack [module]" {
		t.Errorf("packCmd.Use = %q", packCmd.Use)
	}
}

func TestAuditCmdArgs(t *testing.T) {
	if auditCmd.Use != "audit [module]" {
		t.Errorf("auditCmd.Use = %q", auditCmd.Use)
	}
}

func TestDebugCmdArgs(t *testing.T) {
	if debugCmd.Use != "debug [module]" {
		t.Errorf("debugCmd.Use = %q", debugCmd.Use)
	}
}

func TestSignCmdSubcommands(t *testing.T) {
	subNames := []string{"keygen", "verify"}
	for _, name := range subNames {
		found := false
		for _, cmd := range signCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("sous-commande %q non enregistrée dans sign", name)
		}
	}
}

func TestConnectCmdNoArgs(t *testing.T) {
	if connectCmd.Args == nil {
		t.Error("connectCmd.Args doit être configuré (cobra.NoArgs)")
	}
	if connectCmd.Use != "connect" {
		t.Errorf("connectCmd.Use = %q", connectCmd.Use)
	}
}

func TestDisconnectCmdExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "disconnect" {
			found = true
			break
		}
	}
	if !found {
		t.Error("disconnectCmd non enregistrée")
	}
}

func TestPublishCmdExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "publish" {
			found = true
			break
		}
	}
	if !found {
		t.Error("publishCmd non enregistrée")
	}
}

func TestLinkCmdExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "link" {
			found = true
			break
		}
	}
	if !found {
		t.Error("linkCmd non enregistrée")
	}
}

func TestUnlinkCmdExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "unlink" {
			found = true
			break
		}
	}
	if !found {
		t.Error("unlinkCmd non enregistrée")
	}
}

func TestInitCmdExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "init" {
			found = true
			break
		}
	}
	if !found {
		t.Error("initCmd non enregistrée")
	}
}

func TestExitCodes(t *testing.T) {
	// Verify all exit codes from spec section 11.1
	tests := []struct {
		err  error
		want int
	}{
		{pkg.NewError("test", "generic", pkg.ExitError), 1},
		{pkg.NewError("test", "auth", pkg.ExitAuth), 2},
		{pkg.NewError("test", "module", pkg.ExitModuleNotFound), 3},
		{pkg.NewError("test", "manifest", pkg.ExitManifest), 4},
		{pkg.NewError("test", "network", pkg.ExitNetwork), 5},
		{pkg.NewError("test", "perm", pkg.ExitPermission), 6},
		{pkg.NewError("test", "mfa", pkg.ExitMFA), 7},
		{pkg.NewError("test", "build", pkg.ExitBuild), 10},
		{pkg.NewError("test", "publish", pkg.ExitPublish), 11},
		{pkg.NewError("test", "sign", pkg.ExitSigning), 12},
	}
	for _, tt := range tests {
		got := pkg.ExitCodeFor(tt.err)
		if got != tt.want {
			t.Errorf("ExitCodeFor(%v) = %d, want %d", tt.err, got, tt.want)
		}
	}
}

func TestFormatError(t *testing.T) {
	err := pkg.NewErrorWithFix("Authentification", "Token expiré", "Exécutez 'sentient connect'.", pkg.ExitAuth)
	msg := pkg.FormatError(err)
	if msg == "" {
		t.Error("FormatError ne doit pas retourner une chaîne vide")
	}
	// Should contain the category and message
	if !containsStr(msg, "Token expiré") {
		t.Errorf("FormatError doit contenir le message, got: %s", msg)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
