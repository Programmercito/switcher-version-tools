package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Programmercito/switcher-version-tools/internal/config"
	"github.com/Programmercito/switcher-version-tools/internal/env"
	"github.com/Programmercito/switcher-version-tools/internal/tool"
)

// Inspect comprueba instalaciones, entorno persistido y comandos de esta terminal.
func Inspect(ctx context.Context, cfg *config.Config) ([]string, error) {
	names := []string{"PATH"}
	for _, name := range tool.Supported() {
		t, _ := tool.Get(name)
		names = append(names, t.HomeVar)
		names = append(names, t.ExtraHomeVars...)
	}
	snapshot, err := env.ReadSnapshot(names)
	if err != nil {
		return nil, err
	}
	return inspect(ctx, cfg, snapshot, exec.LookPath, tool.DetectVersion), nil
}

func samePath(a, b string) bool {
	return a != "" && b != "" && strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func containsPath(paths, wanted string) bool {
	for _, p := range filepath.SplitList(paths) {
		if samePath(strings.TrimSpace(p), wanted) {
			return true
		}
	}
	return false
}

func inspect(ctx context.Context, cfg *config.Config, snapshot env.Snapshot, lookPath func(string) (string, error), detectVersion func(context.Context, tool.Tool, string) (string, error)) []string {
	var issues []string
	for _, d := range cfg.Downloads {
		t, ok := tool.Get(d.Type)
		if !ok {
			issues = append(issues, fmt.Sprintf("%s: tipo %q no soportado", d.Alias, d.Type))
			continue
		}
		dir, err := config.HomeFor(d.Alias)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: no se pudo resolver la ruta: %v", d.Alias, err))
			continue
		}
		home, err := tool.DetectRoot(t, dir)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: instalación inaccesible: %v", d.Alias, err))
			continue
		}
		executable := filepath.Join(home, t.Executable)
		info, err := os.Stat(executable)
		if err != nil || info.IsDir() {
			issues = append(issues, fmt.Sprintf("%s: ejecutable no encontrado o inaccesible: %s", d.Alias, executable))
			continue
		}
		versionCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		version, versionErr := detectVersion(versionCtx, t, home)
		cancel()
		if versionErr != nil {
			issues = append(issues, fmt.Sprintf("%s: no se pudo comprobar la versión: %v", d.Alias, versionErr))
		} else if d.Version != "" && d.Version != version {
			issues = append(issues, fmt.Sprintf("%s: versión registrada %s, versión instalada %s", d.Alias, d.Version, version))
		}
		if cfg.Current[d.Type] != d.Alias {
			continue
		}
		for _, name := range append([]string{t.HomeVar}, t.ExtraHomeVars...) {
			if !samePath(snapshot.User[name], home) {
				issues = append(issues, fmt.Sprintf("%s: %s de usuario es %q; se esperaba %q", d.Alias, name, snapshot.User[name], home))
			}
			if !samePath(snapshot.Process[name], home) {
				issues = append(issues, fmt.Sprintf("%s: %s de esta terminal es %q; se esperaba %q (actualizá la terminal)", d.Alias, name, snapshot.Process[name], home))
			}
		}
		if !containsPath(snapshot.User["PATH"], t.BinPath(home)) {
			issues = append(issues, fmt.Sprintf("%s: %s no está en el PATH de usuario", d.Alias, t.BinPath(home)))
		}
		found, err := lookPath(filepath.Base(t.Executable))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: ejecutable no disponible en el PATH de esta terminal: %v", d.Alias, err))
		} else if !samePath(found, executable) {
			issues = append(issues, fmt.Sprintf("%s: conflicto en PATH de esta terminal: se ejecutaría %s en vez de %s", d.Alias, found, executable))
			other := t
			other.VersionCommand = append([]string(nil), t.VersionCommand...)
			other.VersionCommand[0] = found
			versionCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			actual, err := detectVersion(versionCtx, other, "")
			cancel()
			if err != nil {
				issues = append(issues, fmt.Sprintf("%s: no se pudo comprobar la versión del ejecutable en PATH: %v", d.Alias, err))
			} else if d.Version != "" && actual != d.Version {
				issues = append(issues, fmt.Sprintf("%s: versión registrada %s, versión en PATH %s", d.Alias, d.Version, actual))
			}
		}
	}
	for toolType, alias := range cfg.Current {
		d, ok := cfg.FindDownload(alias)
		if !ok || d.Type != toolType {
			issues = append(issues, fmt.Sprintf("%s: alias activo %q no está registrado para esta herramienta", toolType, alias))
		}
	}
	sort.Strings(issues)
	return issues
}
