package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Download representa una instalación registrada.
type Download struct {
	Alias string `json:"alias"`
	Type  string `json:"type"`
}

// Config es la estructura persistente del usuario.
// Se mantiene compatible con versiones anteriores:
// almacena en ~/switchjdk/downloads.json.
type Config struct {
	Downloads []Download        `json:"downloads"`
	Current   map[string]string `json:"current"`
}

// userHomeDir es reemplazable para tests.
var userHomeDir = os.UserHomeDir

// Dir retorna el directorio de configuración e instalaciones.
// Se conserva "switchjdk" para no romper instalaciones existentes.
func Dir() (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "switchjdk"), nil
}

// File retorna la ruta al archivo de configuración JSON.
func File() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "downloads.json"), nil
}

// Load lee la configuración. Si no existe, retorna una vacía.
func Load() (*Config, error) {
	path, err := File()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{Current: make(map[string]string)}, nil
		}
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	if cfg.Current == nil {
		cfg.Current = make(map[string]string)
	}
	return &cfg, nil
}

// Save persiste la configuración con indentación legible.
func Save(cfg *Config) error {
	path, err := File()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

// AddDownload agrega una instalación si el alias no existe.
func (c *Config) AddDownload(alias, toolType string) bool {
	for _, d := range c.Downloads {
		if d.Alias == alias {
			return false
		}
	}
	c.Downloads = append(c.Downloads, Download{Alias: alias, Type: toolType})
	return true
}

// FindDownload busca una instalación por alias.
func (c *Config) FindDownload(alias string) (Download, bool) {
	for _, d := range c.Downloads {
		if d.Alias == alias {
			return d, true
		}
	}
	return Download{}, false
}

// RemoveDownload elimina una instalación del registro y del "current".
func (c *Config) RemoveDownload(alias string) bool {
	for i, d := range c.Downloads {
		if d.Alias == alias {
			c.Downloads = append(c.Downloads[:i], c.Downloads[i+1:]...)
			delete(c.Current, d.Type)
			return true
		}
	}
	return false
}

// HomeFor retorna el directorio de instalación para un alias.
func HomeFor(alias string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, alias), nil
}
