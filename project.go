package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/Programmercito/switcher-version-tools/internal/config"
	"github.com/Programmercito/switcher-version-tools/internal/tool"
)

type projectConfig struct {
	Tools map[string]string `json:"tools"`
}

func readProject(path string) (projectConfig, error) {
	var project projectConfig
	f, err := os.Open(path)
	if err != nil {
		return project, fmt.Errorf("abrir %s: %w", path, err)
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&project); err != nil {
		return project, fmt.Errorf("leer %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return project, fmt.Errorf("%s contiene datos adicionales", path)
		}
		return project, fmt.Errorf("leer %s: %w", path, err)
	}
	if len(project.Tools) == 0 {
		return project, fmt.Errorf("%s debe declarar al menos una herramienta en tools", path)
	}
	return project, nil
}

func projectAliases(project projectConfig, cfg *config.Config) ([]string, error) {
	types := make([]string, 0, len(project.Tools))
	for name := range project.Tools {
		types = append(types, name)
	}
	sort.Strings(types)
	aliases := make([]string, 0, len(types))
	for _, name := range types {
		alias := project.Tools[name]
		t, ok := tool.Get(name)
		if !ok {
			return nil, fmt.Errorf("herramienta no soportada en el proyecto: %s", name)
		}
		d, ok := cfg.FindDownload(alias)
		if !ok || d.Type != name {
			return nil, fmt.Errorf("alias %q no está instalado para %s", alias, name)
		}
		dir, err := config.HomeFor(alias)
		if err != nil {
			return nil, err
		}
		home, err := tool.DetectRoot(t, dir)
		if err != nil {
			return nil, fmt.Errorf("%s: instalación inaccesible: %w", alias, err)
		}
		executable := filepath.Join(home, t.Executable)
		info, err := os.Stat(executable)
		if err != nil {
			return nil, fmt.Errorf("%s: ejecutable inaccesible: %w", alias, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("%s: %s no es un ejecutable", alias, executable)
		}
		aliases = append(aliases, alias)
	}
	return aliases, nil
}

func useProject() error {
	project, err := readProject("switchtool.json")
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	aliases, err := projectAliases(project, cfg)
	if err != nil {
		return err
	}
	for _, alias := range aliases {
		if err := switchToAlias(alias); err != nil {
			return fmt.Errorf("no se pudo activar %s; las herramientas anteriores pueden haber cambiado: %w", alias, err)
		}
	}
	return nil
}
