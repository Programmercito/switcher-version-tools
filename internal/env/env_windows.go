//go:build windows

package env

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Los cambios de entorno se hacen con PowerShell. Los valores viajan como
// variables de entorno del proceso hijo para no mezclar rutas con el script.
func init() {
	systemSet = psSet
	systemAddToPath = psAddToPath
	systemRemoveFromPath = psRemoveFromPath
	systemReloadPath = psReloadPath
}

func powershell() string {
	for _, shell := range []string{"powershell", "pwsh"} {
		if _, err := exec.LookPath(shell); err == nil {
			return shell
		}
	}
	return "powershell"
}

func psRun(script string, extraEnv ...string) (string, error) {
	cmd := exec.Command(powershell(), "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), extraEnv...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func psSet(name, value string) error {
	_, err := psRun(
		`[Environment]::SetEnvironmentVariable($env:SWTOOL_NAME, $env:SWTOOL_VALUE, 'User')`,
		"SWTOOL_NAME="+name, "SWTOOL_VALUE="+value,
	)
	if err != nil {
		return fmt.Errorf("escribir variable %s: %w", name, err)
	}
	return nil
}

func splitPath(val string) []string {
	var parts []string
	for _, p := range strings.Split(val, string(os.PathListSeparator)) {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// readPath lee el PATH de un ámbito: "User" o "Machine".
func readPath(scope string) ([]string, error) {
	val, err := psRun(`[Environment]::GetEnvironmentVariable('PATH', '` + scope + `')`)
	if err != nil {
		return nil, fmt.Errorf("leer PATH (%s): %w", scope, err)
	}
	return splitPath(val), nil
}

func writeUserPath(parts []string) error {
	_, err := psRun(
		`[Environment]::SetEnvironmentVariable('PATH', $env:SWTOOL_PATH, 'User')`,
		"SWTOOL_PATH="+strings.Join(parts, string(os.PathListSeparator)),
	)
	if err != nil {
		return fmt.Errorf("escribir PATH: %w", err)
	}
	return nil
}

func normalizeForComparison(p string) string {
	// Normalizamos separadores para comparar sin importar \ o /,
	// pero preservamos el original al guardar.
	return strings.ToLower(filepath.FromSlash(strings.TrimSpace(p)))
}

func psAddToPath(path string) error {
	if path == "" {
		return nil
	}
	path = filepath.FromSlash(strings.TrimSpace(path))
	target := normalizeForComparison(path)

	parts, err := readPath("User")
	if err != nil {
		return err
	}

	for _, p := range parts {
		if normalizeForComparison(p) == target {
			return nil
		}
	}

	parts = append(parts, path)
	return writeUserPath(parts)
}

func psRemoveFromPath(path string) error {
	if path == "" {
		return nil
	}
	path = filepath.FromSlash(strings.TrimSpace(path))
	target := normalizeForComparison(path)
	altTarget := strings.ReplaceAll(target, `\`, `/`)

	parts, err := readPath("User")
	if err != nil {
		return err
	}

	var filtered []string
	for _, p := range parts {
		n := normalizeForComparison(p)
		if n == target || n == altTarget {
			continue
		}
		filtered = append(filtered, p)
	}

	return writeUserPath(filtered)
}

func psReloadPath() error {
	userParts, err := readPath("User")
	if err != nil {
		return err
	}

	machineParts, err := readPath("Machine")
	if err != nil {
		// El PATH de máquina puede no ser legible; en ese caso usamos solo el del usuario.
		machineParts = nil
	}

	seen := make(map[string]struct{})
	var result []string

	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		key := normalizeForComparison(p)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		result = append(result, p)
	}

	for _, p := range machineParts {
		add(p)
	}
	for _, p := range userParts {
		add(p)
	}

	combined := strings.Join(result, string(os.PathListSeparator))
	return os.Setenv("PATH", combined)
}
