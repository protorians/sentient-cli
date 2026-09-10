package module

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/protorians/sentient-cli/internal/config"
	"github.com/protorians/sentient-cli/internal/pkg"
)

// MaxArchiveSize is the maximum allowed archive size (store limit: 50 MB).
const MaxArchiveSize = 50 * 1024 * 1024

// Packer builds `.smp` archives (renamed ZIP) for a module.
type Packer struct {
	Root string
}

// PackResult describes a created archive.
type PackResult struct {
	Module  string
	Version string
	Path    string
	Size    int64
}

// Pack builds and moves the archive of `name` into `.sentients/build/`.
func (p *Packer) Pack(name string) (*PackResult, error) {
	v := &Validator{Root: p.Root}
	res, err := v.ValidateModule(name)
	if err != nil {
		return nil, err
	}
	if res.HasErrors() {
		return nil, fmt.Errorf("le module %q contient %d erreur(s) de validation, pack annulé", name, res.ErrorCount())
	}

	m, err := LoadManifest(config.ManifestPath(p.Root, name))
	if err != nil {
		return nil, err
	}

	buildDir := config.BuildDir(p.Root)
	if err := pkg.CreateDir(buildDir); err != nil {
		return nil, err
	}

	archivePath := filepath.Join(buildDir, fmt.Sprintf("%s-%s.smp", name, m.Version))

	moduleSrc := filepath.Join(p.Root, config.ExternalModulesDir, name)
	assetsSrc := config.ModuleAssetsDir(p.Root, name)

	if err := p.createArchive(archivePath, moduleSrc, assetsSrc); err != nil {
		return nil, err
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return nil, fmt.Errorf("stat de l'archive impossible : %w", err)
	}
	if info.Size() > MaxArchiveSize {
		os.Remove(archivePath)
		return nil, fmt.Errorf("l'archive dépasse la taille maximale de %d MB", MaxArchiveSize/(1024*1024))
	}

	return &PackResult{
		Module:  name,
		Version: m.Version,
		Path:    archivePath,
		Size:    info.Size(),
	}, nil
}

// createArchive zips `moduleSrc` (prefixed `external_modules/<name>/`) and,
// when present, `assetsSrc` (prefixed `public/assets/<name>/`) into `dest`.
func (p *Packer) createArchive(dest, moduleSrc, assetsSrc string) error {
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("création de l'archive %s impossible : %w", dest, err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	addToZip := func(src string) error {
		if !pkg.DirExists(src) {
			return nil
		}
		return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(p.Root, path)
			if err != nil {
				return err
			}
			w, err := zw.Create(filepath.ToSlash(rel))
			if err != nil {
				return fmt.Errorf("écriture dans l'archive impossible : %w", err)
			}
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, in); err != nil {
				in.Close()
				return fmt.Errorf("copie de %s dans l'archive impossible : %w", path, err)
			}
			return in.Close()
		})
	}

	if err := addToZip(moduleSrc); err != nil {
		return err
	}
	return addToZip(assetsSrc)
}
