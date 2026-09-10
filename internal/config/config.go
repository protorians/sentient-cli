package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config mirrors the optional `.sentient-cli.toml` file at the project root.
type Config struct {
	Project ProjectConfig `toml:"project"`
	Publish PublishConfig `toml:"publish"`
	Debug   DebugConfig   `toml:"debug"`
}

// ProjectConfig configures the project-level settings.
type ProjectConfig struct {
	Name           string `toml:"name"`
	PackageManager string `toml:"package_manager"`
}

// PublishConfig configures publication defaults.
type PublishConfig struct {
	DefaultRegistry string `toml:"default_registry"`
	AutoAudit       bool   `toml:"auto_audit"`
}

// DebugConfig configures debug/log behaviour.
type DebugConfig struct {
	Verbose  bool   `toml:"verbose"`
	LogLevel string `toml:"log_level"`
}

// Default returns a Config populated with sensible defaults.
func Default() Config {
	return Config{
		Project: ProjectConfig{},
		Publish: PublishConfig{
			DefaultRegistry: "https://store.sentient.dev",
			AutoAudit:       true,
		},
		Debug: DebugConfig{
			Verbose:  false,
			LogLevel: "info",
		},
	}
}

// Load reads a config file at path. A missing file yields the defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	} else if err != nil {
		return cfg, fmt.Errorf("lecture du fichier de configuration %s impossible : %w", path, err)
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("décodage du fichier de configuration %s impossible : %w", path, err)
	}
	return cfg, nil
}

// Save writes the config as TOML at path.
func (c Config) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("création du fichier de configuration %s impossible : %w", path, err)
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(c); err != nil {
		return fmt.Errorf("écriture du fichier de configuration %s impossible : %w", path, err)
	}
	return nil
}
