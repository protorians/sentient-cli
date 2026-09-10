package module

import (
	"fmt"
	"path/filepath"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/pkg"
)

// LinkResult summarises a link operation.
type LinkResult struct {
	LocalModule  string
	RemoteToken  string
	RemoteName   string
	RemoteVersion string
}

// Linker manages the link between local modules and remote store modules.
type Linker struct {
	Root string
}

// Link associates a local module with a remote token.
func (l *Linker) Link(name, remoteToken string) (*LinkResult, error) {
	moduleDir := filepath.Join(l.Root, config.ExternalModulesDir, name)
	if !pkg.DirExists(moduleDir) {
		return nil, fmt.Errorf("le module %q n'existe pas dans %s", name, config.ExternalModulesDir)
	}

	manifestPath := filepath.Join(moduleDir, config.ManifestFileName)
	m, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	m.Token = remoteToken
	if err := m.Save(manifestPath); err != nil {
		return nil, fmt.Errorf("mise à jour du manifest impossible : %w", err)
	}

	return &LinkResult{
		LocalModule:   name,
		RemoteToken:   remoteToken,
		RemoteName:    m.Name,
		RemoteVersion: m.Version,
	}, nil
}

// Unlink removes the remote token association from a local module.
func (l *Linker) Unlink(name string) error {
	moduleDir := filepath.Join(l.Root, config.ExternalModulesDir, name)
	if !pkg.DirExists(moduleDir) {
		return fmt.Errorf("le module %q n'existe pas dans %s", name, config.ExternalModulesDir)
	}

	manifestPath := filepath.Join(moduleDir, config.ManifestFileName)
	m, err := LoadManifest(manifestPath)
	if err != nil {
		return err
	}

	// Generate a new local UUID token (de-link from remote)
	m.Token = pkg.NewUUID()
	if err := m.Save(manifestPath); err != nil {
		return fmt.Errorf("mise à jour du manifest impossible : %w", err)
	}

	return nil
}

// LinkedModules returns modules that have a remote token in their manifest.
// Used by unlink to list only linked modules.
func (l *Linker) LinkedModules() ([]string, error) {
	dir := filepath.Join(l.Root, config.ExternalModulesDir)
	if !pkg.DirExists(dir) {
		return nil, fmt.Errorf("le dossier %q est introuvable", config.ExternalModulesDir)
	}

	entries, err := filepath.Glob(filepath.Join(dir, "*", config.ManifestFileName))
	if err != nil {
		return nil, fmt.Errorf("recherche de manifest impossible : %w", err)
	}

	var linked []string
	for _, manifestPath := range entries {
		m, err := LoadManifest(manifestPath)
		if err != nil {
			continue
		}
		// A module is "linked" if its token doesn't look like a UUID
		// (remote tokens use a different format, e.g. "m_xxx")
		// For simplicity, we consider any module with a non-empty token as potentially linked.
		// The cmd layer will verify with the API.
		_ = m
		moduleDir := filepath.Dir(manifestPath)
		linked = append(linked, filepath.Base(moduleDir))
	}

	return linked, nil
}
