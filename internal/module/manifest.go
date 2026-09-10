package module

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/protorians/sentient-cli/internal/pkg"
)

// Manifest is the metadata file of a Sentient module (`manifest.json`).
type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	ID            string            `json:"id"`
	Domain        string            `json:"domain"`
	Key           string            `json:"key"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Version       string            `json:"version"`
	Icon          string            `json:"icon"`
	Type          string            `json:"type"`
	Entry         string            `json:"entry"`
	URI           string            `json:"uri"`
	Token         string            `json:"token"`
	Publisher     Publisher         `json:"publisher"`
	Platforms     Platforms         `json:"platforms"`
	ManagerCompat Compatibility     `json:"managerCompatibility"`
	APICompat     Compatibility     `json:"apiCompatibility"`
	Permissions   []string          `json:"permissions"`
	APIScopes     []string          `json:"apiScopes"`
	Capabilities  Capabilities      `json:"capabilities"`
	IsEnabled     bool              `json:"isEnabled"`
	IsDefault     bool              `json:"isDefault"`
	Requirements  map[string]any    `json:"requirements"`
	Dependencies  map[string]string `json:"dependencies"`
	Widgets       []json.RawMessage `json:"widgets"`
	Routines      []json.RawMessage `json:"routines"`
	Menu          Menu              `json:"menu"`
}

// Publisher describes the developer publishing the module.
type Publisher struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Platforms declares which platforms the module supports.
type Platforms struct {
	Web     Platform `json:"web"`
	Desktop Platform `json:"desktop"`
	Mobile  Platform `json:"mobile"`
}

// Platform describes support for a single platform.
type Platform struct {
	Supported bool     `json:"supported"`
	Modes     []string `json:"modes,omitempty"`
}

// Compatibility expresses a semver window.
type Compatibility struct {
	Min string `json:"min"`
	Max string `json:"max"`
}

// Capabilities declares module capabilities.
type Capabilities struct {
	NeedsNetwork              bool `json:"needsNetwork"`
	SupportsOffline           bool `json:"supportsOffline"`
	RequiresOrganization      bool `json:"requiresOrganization"`
	RequiresAuthenticatedUser bool `json:"requiresAuthenticatedUser"`
}

// Menu holds menu entries declared by the module.
type Menu struct {
	Items []json.RawMessage `json:"items"`
}

// NewManifest builds a fresh manifest for a module.
func NewManifest(name, description string) Manifest {
	upperKey := upperSnake(name)
	return Manifest{
		SchemaVersion: 1,
		ID:            name,
		Domain:        "mod.sentients." + name,
		Key:           upperKey,
		Name:          displayName(name),
		Description:   description,
		Version:       "0.1.0",
		Icon:          "PuzzleIcon",
		Type:          "EXTERNAL",
		Entry:         "index.tsx",
		URI:           "/" + name,
		Token:         pkg.NewUUID(),
		Publisher:     Publisher{ID: "", Name: ""},
		Platforms: Platforms{
			Web:     Platform{Supported: true, Modes: []string{"web"}},
			Desktop: Platform{Supported: false},
			Mobile:  Platform{Supported: false},
		},
		ManagerCompat: Compatibility{Min: "0.0.0", Max: "*.x"},
		APICompat:     Compatibility{Min: "0.0.0", Max: "*.x"},
		Permissions:   []string{},
		APIScopes:     []string{},
		Capabilities: Capabilities{
			NeedsNetwork:              true,
			SupportsOffline:           false,
			RequiresOrganization:      false,
			RequiresAuthenticatedUser: true,
		},
		IsEnabled:    true,
		IsDefault:    false,
		Requirements: map[string]any{},
		Dependencies: map[string]string{"@sentients/sdk": "workspace:*"},
		Widgets:      []json.RawMessage{},
		Routines:     []json.RawMessage{},
		Menu:         Menu{Items: []json.RawMessage{}},
	}
}

// LoadManifest reads and decodes a manifest.json file.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s impossible : %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("décodage JSON de %s impossible : %w", path, err)
	}
	return &m, nil
}

// Save writes the manifest to path with pretty-printed JSON.
func (m *Manifest) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("sérialisation du manifest impossible : %w", err)
	}
	data = append(data, '\n')
	return pkg.WriteFile(path, data)
}

var kebabNameRE = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ValidateName checks a module name: kebab-case, 3–64 chars.
func ValidateName(name string) error {
	if len(name) < 3 || len(name) > 64 {
		return fmt.Errorf("le nom du module doit contenir entre 3 et 64 caractères")
	}
	if !kebabNameRE.MatchString(name) {
		return fmt.Errorf("le nom du module doit être en kebab-case (minuscules, chiffres et tirets uniquement)")
	}
	return nil
}

func upperSnake(name string) string {
	out := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '-' {
			out = append(out, '_')
		} else {
			if c >= 'a' && c <= 'z' {
				c = c - 'a' + 'A'
			}
			out = append(out, c)
		}
	}
	return string(out)
}

// displayName converts a kebab-case name to a Title Case display name.
func displayName(name string) string {
	out := make([]byte, 0, len(name))
	start := true
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '-' {
			out = append(out, ' ')
			start = true
			continue
		}
		if start {
			if c >= 'a' && c <= 'z' {
				c = c - 'a' + 'A'
			}
			start = false
		}
		out = append(out, c)
	}
	return string(out)
}
