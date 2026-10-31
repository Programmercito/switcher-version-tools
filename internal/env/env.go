package env

import (
	"os"
	"path/filepath"
	"strings"
)

// Funciones de sistema, reemplazables para tests.
var (
	systemSet            = func(name, value string) error { return nil }
	systemAddToPath      = func(path string) error { return nil }
	systemRemoveFromPath = func(path string) error { return nil }
	systemReloadPath     = func() error { return nil }
	systemSnapshot       = processSnapshot
)

// Snapshot describe las variables persistidas y el PATH efectivo del proceso.
type Snapshot struct {
	User        map[string]string `json:"user"`
	MachinePath string            `json:"machinePath"`
	ProcessPath string            `json:"-"`
	Process     map[string]string `json:"-"`
}

func processSnapshot(names []string) (Snapshot, error) {
	s := Snapshot{User: make(map[string]string), Process: make(map[string]string), ProcessPath: os.Getenv("PATH")}
	for _, name := range names {
		s.User[name] = os.Getenv(name)
		s.Process[name] = os.Getenv(name)
	}
	return s, nil
}

// ReadSnapshot lee el entorno persistido y el entorno de esta terminal.
func ReadSnapshot(names []string) (Snapshot, error) {
	return systemSnapshot(names)
}

// Set persiste una variable de entorno (si la plataforma lo permite)
// y la refleja en el proceso actual.
func Set(name, value string) error {
	if err := systemSet(name, value); err != nil {
		return err
	}
	return os.Setenv(name, value)
}

// AddToPath agrega una ruta al PATH sin duplicados.
func AddToPath(path string) error {
	if path == "" {
		return nil
	}
	normalized := normalizePath(path)
	if err := systemAddToPath(normalized); err != nil {
		return err
	}

	current := os.Getenv("PATH")
	sep := string(os.PathListSeparator)
	for _, p := range strings.Split(current, sep) {
		if normalizePath(p) == normalized {
			return nil
		}
	}
	if current == "" {
		return os.Setenv("PATH", normalized)
	}
	return os.Setenv("PATH", current+sep+normalized)
}

// RemoveFromPath elimina una ruta del PATH.
func RemoveFromPath(path string) error {
	if path == "" {
		return nil
	}
	normalized := normalizePath(path)
	if err := systemRemoveFromPath(normalized); err != nil {
		return err
	}

	current := os.Getenv("PATH")
	sep := string(os.PathListSeparator)
	var parts []string
	for _, p := range strings.Split(current, sep) {
		if p == "" {
			continue
		}
		if normalizePath(p) == normalized {
			continue
		}
		parts = append(parts, p)
	}
	return os.Setenv("PATH", strings.Join(parts, sep))
}

// ReloadPath recarga el PATH del sistema en el proceso actual.
func ReloadPath() error {
	return systemReloadPath()
}

func normalizePath(p string) string {
	return filepath.FromSlash(p)
}
