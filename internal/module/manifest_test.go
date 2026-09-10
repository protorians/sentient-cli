package module

import (
	"testing"

	"github.com/protorians/sentient-cli/internal/pkg"
)

func TestValidateName(t *testing.T) {
	valid := []string{"blog", "blog-manager", "a1b2", "my-module-name"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []string{"", "ab", "Blog", "blog manager", "blog_", "-blog", "blog-", "éé"}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", name)
		}
	}
}

func TestNewManifest(t *testing.T) {
	m := NewManifest("blog-manager", "Gestion de blog")

	if m.ID != "blog-manager" {
		t.Errorf("ID = %q, want blog-manager", m.ID)
	}
	if m.Key != "BLOG_MANAGER" {
		t.Errorf("Key = %q, want BLOG_MANAGER", m.Key)
	}
	if m.Domain != "mod.sentients.blog-manager" {
		t.Errorf("Domain = %q, want mod.sentients.blog-manager", m.Domain)
	}
	if m.URI != "/blog-manager" {
		t.Errorf("URI = %q, want /blog-manager", m.URI)
	}
	if m.Version != "0.1.0" {
		t.Errorf("Version = %q, want 0.1.0", m.Version)
	}
	if m.Token == "" {
		t.Error("Token est vide, un UUID doit être généré")
	}
	if !pkg.IsUUID(m.Token) {
		t.Errorf("Token %q n'est pas un UUID valide", m.Token)
	}
	if m.Entry != "index.tsx" {
		t.Errorf("Entry = %q, want index.tsx", m.Entry)
	}
	if m.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", m.SchemaVersion)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/manifest.json"

	m := NewManifest("billing", "")
	if err := m.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if loaded.ID != m.ID || loaded.Key != m.Key || loaded.Token != m.Token {
		t.Errorf("round-trip mismatch: loaded=%+v want=%+v", loaded, m)
	}
}

func TestUpperSnake(t *testing.T) {
	cases := map[string]string{
		"blog":         "BLOG",
		"blog-manager": "BLOG_MANAGER",
		"a1b2":         "A1B2",
	}
	for in, want := range cases {
		if got := upperSnake(in); got != want {
			t.Errorf("upperSnake(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSemver(t *testing.T) {
	ok := []string{"0.1.0", "1.2.3", "1.2.3-beta.1", "v2.0.0", "2.0.0+meta"}
	bad := []string{"", "1", "1.2", "abc", "1.2.x"}

	for _, v := range ok {
		if !isSemver(v) {
			t.Errorf("isSemver(%q) = false, want true", v)
		}
	}
	for _, v := range bad {
		if isSemver(v) {
			t.Errorf("isSemver(%q) = true, want false", v)
		}
	}
}
