//go:build windows

package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	hwndBroadcast   = uintptr(0xFFFF)
	wmSettingChange = uintptr(0x001A)
	smtoAbortIfHung = uintptr(0x0002)
)

func init() {
	systemSet = registrySet
	systemAddToPath = registryAddToPath
	systemRemoveFromPath = registryRemoveFromPath
	systemReloadPath = registryReloadPath
}

// envKey abre la clave de entorno del usuario actual (HKCU\Environment).
func envKey(access uint32) (registry.Key, error) {
	return registry.OpenKey(registry.CURRENT_USER, `Environment`, access)
}

// notifyEnvChange avisa a Windows que las variables de entorno cambiaron,
// para que Explorer y otras apps recarguen sin necesidad de reiniciar.
func notifyEnvChange() error {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("SendMessageTimeoutW")

	ptr, err := syscall.UTF16PtrFromString("Environment")
	if err != nil {
		return err
	}

	ret, _, err := proc.Call(
		hwndBroadcast,
		wmSettingChange,
		0,
		uintptr(unsafe.Pointer(ptr)),
		smtoAbortIfHung,
		5000,
		0,
	)
	if ret == 0 {
		return fmt.Errorf("SendMessageTimeoutW falló: %w", err)
	}
	return nil
}

func registrySet(name, value string) error {
	k, err := envKey(registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("abrir registro para set: %w", err)
	}
	defer k.Close()

	if err := k.SetStringValue(name, value); err != nil {
		return fmt.Errorf("escribir variable %s: %w", name, err)
	}
	return notifyEnvChange()
}

func readUserPath() ([]string, error) {
	k, err := envKey(registry.QUERY_VALUE)
	if err != nil {
		return nil, fmt.Errorf("abrir registro para leer PATH: %w", err)
	}
	defer k.Close()

	val, _, err := k.GetStringValue("Path")
	if err != nil {
		if err == registry.ErrNotExist {
			return nil, nil
		}
		return nil, fmt.Errorf("leer PATH: %w", err)
	}

	var parts []string
	for _, p := range strings.Split(val, string(os.PathListSeparator)) {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return parts, nil
}

func writeUserPath(parts []string) error {
	k, err := envKey(registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("abrir registro para escribir PATH: %w", err)
	}
	defer k.Close()

	newPath := strings.Join(parts, string(os.PathListSeparator))
	if err := k.SetExpandStringValue("Path", newPath); err != nil {
		return fmt.Errorf("escribir PATH: %w", err)
	}
	return notifyEnvChange()
}

func normalizeForComparison(p string) string {
	// Normalizamos separadores para comparar sin importar \ o /,
	// pero preservamos el original al guardar.
	return strings.ToLower(filepath.FromSlash(strings.TrimSpace(p)))
}

func registryAddToPath(path string) error {
	if path == "" {
		return nil
	}
	path = filepath.FromSlash(strings.TrimSpace(path))
	target := normalizeForComparison(path)

	parts, err := readUserPath()
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

func registryRemoveFromPath(path string) error {
	if path == "" {
		return nil
	}
	path = filepath.FromSlash(strings.TrimSpace(path))
	target := normalizeForComparison(path)
	altTarget := strings.ReplaceAll(target, `\`, `/`)

	parts, err := readUserPath()
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

func registryReloadPath() error {
	userParts, err := readUserPath()
	if err != nil {
		return err
	}

	machineParts, err := readMachinePath()
	if err != nil {
		// El PATH de máquina puede no ser legible por permisos;
		// en ese caso usamos solo el del usuario.
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

func readMachinePath() ([]string, error) {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return nil, err
	}
	defer k.Close()

	val, _, err := k.GetStringValue("Path")
	if err != nil {
		if err == registry.ErrNotExist {
			return nil, nil
		}
		return nil, err
	}

	var parts []string
	for _, p := range strings.Split(val, string(os.PathListSeparator)) {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return parts, nil
}
