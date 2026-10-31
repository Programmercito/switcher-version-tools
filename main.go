package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/Programmercito/switcher-version-tools/internal/cli"
	"github.com/Programmercito/switcher-version-tools/internal/config"
	"github.com/Programmercito/switcher-version-tools/internal/doctor"
	"github.com/Programmercito/switcher-version-tools/internal/download"
	"github.com/Programmercito/switcher-version-tools/internal/env"
	"github.com/Programmercito/switcher-version-tools/internal/extract"
	"github.com/Programmercito/switcher-version-tools/internal/tool"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// version se sobreescribe en build time con:
// go build -ldflags "-X main.version=1.2.3" -o switchtool.exe .
var version = "dev"

func getVersion() string {
	if version != "dev" {
		return version
	}
	// Fallback para desarrollo: intentar obtener el tag de git.
	if out, err := exec.Command("git", "describe", "--tags", "--always").Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return version
}

func main() {
	currentVersion := getVersion()

	args := []string{os.Args[0]}
	for _, arg := range os.Args[1:] {
		if arg == "--no-color" {
			lipgloss.SetColorProfile(termenv.Ascii)
			continue
		}
		args = append(args, arg)
	}
	os.Args = args
	machineOutput := len(args) == 2 && args[1] == "--version" ||
		len(args) >= 3 && (args[1] == "list" || args[1] == "ls") && (args[2] == "--json" || args[2] == "--quiet")
	if !machineOutput {
		cli.PrintLogo()
	}

	if len(os.Args) < 2 {
		if !cli.IsInteractive() {
			cli.ShowHelp(currentVersion)
			os.Exit(1)
		}
		runInteractive(interactiveSwitch)
		return
	}

	ui := cli.NewUI()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	arg1 := os.Args[1]

	switch {
	case arg1 == "use" && len(os.Args) == 2:
		if !cli.IsInteractive() {
			cli.Errorf("'use' requiere una terminal interactiva.\n")
			os.Exit(1)
		}
		runInteractive(interactiveSwitch)
	case arg1 == "help" || arg1 == "-h" || arg1 == "--help":
		cli.ShowHelp(currentVersion)
	case arg1 == "--version" && len(os.Args) == 2:
		fmt.Println(currentVersion)
	case arg1 == "list" || arg1 == "ls":
		mode := ""
		if len(os.Args) == 3 && (os.Args[2] == "--json" || os.Args[2] == "--quiet") {
			mode = os.Args[2]
		} else if len(os.Args) != 2 {
			fmt.Fprintln(os.Stderr, "Error: uso: switchtool list [--json | --quiet]")
			os.Exit(1)
		}
		if err := listCurrent(mode); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case arg1 == "doctor" && len(os.Args) == 2:
		if err := runDoctor(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case arg1 == "project" && len(os.Args) == 3 && os.Args[2] == "use":
		if err := useProject(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		cli.ReloadNotice()
	case arg1 == "remove" && len(os.Args) == 2 && cli.IsInteractive():
		runInteractive(interactiveRemove)
	case arg1 == "remove":
		if len(os.Args) != 3 {
			cli.Errorf("'remove' requiere un alias.\n")
			cli.ShowHelp(currentVersion)
			os.Exit(1)
		}
		if err := removeAlias(os.Args[2]); err != nil {
			cli.Errorf("Error: %v\n", err)
			os.Exit(1)
		}
	case len(os.Args) == 2:
		if err := switchToAlias(arg1); err != nil {
			cli.Errorf("Error: %v\n", err)
			os.Exit(1)
		}
		cli.ReloadNotice()
	case len(os.Args) == 4:
		tipo := os.Args[1]
		alias := os.Args[2]
		source := os.Args[3]
		if !tool.IsValid(tipo) {
			cli.Errorf("Tipo '%s' no válido. Soportados: %s.\n", tipo, strings.Join(tool.Supported(), ", "))
			os.Exit(1)
		}
		if err := install(ctx, ui, tipo, alias, source); err != nil {
			cli.Errorf("Error: %v\n", err)
			os.Exit(1)
		}
	default:
		cli.Errorf("Número de parámetros incorrecto.\n\n")
		cli.ShowHelp(currentVersion)
		os.Exit(1)
	}
}

// runInteractive ejecuta un flujo interactivo y maneja cancelación y errores.
func runInteractive(flow func() error) {
	err := flow()
	switch {
	case err == nil:
	case errors.Is(err, cli.ErrCancelled):
		cli.Mutedf("Cancelado.\n")
	case errors.Is(err, cli.ErrNoInstalls):
		cli.Mutedf("No hay instalaciones registradas.\n")
		cli.Infof("Instala una con: switchtool <tipo> <alias> <url|path_zip>\n")
	default:
		cli.Errorf("Error: %v\n", err)
		os.Exit(1)
	}
}

func interactiveSwitch() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	alias, err := cli.PickInstalled(cfg, "usar")
	if err != nil {
		return err
	}
	if err := switchToAlias(alias); err != nil {
		return err
	}
	cli.ReloadNotice()
	return nil
}

func interactiveRemove() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	alias, err := cli.PickInstalled(cfg, "eliminar")
	if err != nil {
		return err
	}
	ok, err := cli.Confirm(fmt.Sprintf("¿Eliminar '%s' del registro y del disco?", alias))
	if err != nil {
		return err
	}
	if !ok {
		cli.Mutedf("Cancelado.\n")
		return nil
	}
	return removeAlias(alias)
}

func install(ctx context.Context, ui *cli.UI, tipo, alias, source string) error {
	cli.Warnf("Iniciando instalación de %s (%s)\n", alias, tipo)
	cli.Infof("Origen: %s\n", source)

	t, ok := tool.Get(tipo)
	if !ok {
		return fmt.Errorf("tipo no soportado: %s", tipo)
	}

	targetDir, err := config.HomeFor(alias)
	if err != nil {
		return err
	}
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("el directorio %s ya existe; elimínalo primero o usá otro alias", targetDir)
	}

	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("switchtool-%s-%s", tipo, alias))
	cleanup, err := download.Fetch(ctx, source, tempFile, ui.PrintDownloadProgress)
	if err != nil {
		return err
	}
	defer cleanup()
	ui.PrintDownloadComplete()

	if err := extract.Extract(ctx, tempFile, targetDir, ui.PrintExtractProgress); err != nil {
		return err
	}
	ui.PrintExtractComplete()

	actualHome, err := tool.DetectRoot(t, targetDir)
	if err != nil {
		return err
	}
	cli.Successf("Directorio principal detectado: %s\n", actualHome)

	detectedVersion, err := tool.DetectVersion(ctx, t, actualHome)
	if err != nil {
		cli.Warnf("No se pudo detectar la versión real de %s: %v\n", tipo, err)
	} else {
		cli.Successf("Versión real detectada: %s\n", detectedVersion)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	previousAlias := cfg.Current[tipo]
	if previousAlias != "" && previousAlias != alias {
		if err := removeFromPathForAlias(t, previousAlias); err != nil {
			return err
		}
	}

	if cfg.AddDownload(alias, tipo, detectedVersion) {
		cli.Successf("Alias '%s' agregado a la configuración.\n", alias)
	} else {
		cli.Warnf("Alias '%s' ya existe en la configuración.\n", alias)
	}

	cfg.Current[tipo] = alias
	if err := config.Save(cfg); err != nil {
		return err
	}

	if err := applyToolEnv(t, actualHome); err != nil {
		return err
	}

	cli.ReloadNotice()
	return nil
}

func switchToAlias(alias string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	d, ok := cfg.FindDownload(alias)
	if !ok {
		return fmt.Errorf("alias '%s' no encontrado en la configuración", alias)
	}

	t, ok := tool.Get(d.Type)
	if !ok {
		return fmt.Errorf("tipo '%s' ya no está soportado", d.Type)
	}

	targetDir, err := config.HomeFor(alias)
	if err != nil {
		return err
	}
	actualHome, err := tool.DetectRoot(t, targetDir)
	if err != nil {
		return err
	}

	previousAlias := cfg.Current[d.Type]
	if previousAlias != "" && previousAlias != alias {
		if err := removeFromPathForAlias(t, previousAlias); err != nil {
			return err
		}
	}

	cfg.Current[d.Type] = alias
	if err := config.Save(cfg); err != nil {
		return err
	}

	cli.Warnf("Cambiando a alias '%s' (tipo: %s)...\n", alias, d.Type)
	if err := applyToolEnv(t, actualHome); err != nil {
		return err
	}
	cli.Successf("Listo: ahora se está usando '%s' para %s.\n", alias, d.Type)
	return nil
}

func removeAlias(alias string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	d, ok := cfg.FindDownload(alias)
	if !ok {
		return fmt.Errorf("alias '%s' no encontrado en la configuración", alias)
	}

	t, ok := tool.Get(d.Type)
	if !ok {
		return fmt.Errorf("tipo '%s' ya no está soportado", d.Type)
	}

	targetDir, err := config.HomeFor(alias)
	if err != nil {
		return err
	}

	if cfg.Current[d.Type] == alias {
		actualHome, err := tool.DetectRoot(t, targetDir)
		if err == nil {
			_ = env.RemoveFromPath(t.BinPath(actualHome))
		}
	}

	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("no se pudo eliminar %s: %w", targetDir, err)
	}

	cfg.RemoveDownload(alias)
	if err := config.Save(cfg); err != nil {
		return err
	}

	cli.Successf("Alias '%s' eliminado.\n", alias)
	return nil
}

func listCurrent(mode string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	dir, err := config.Dir()
	if err != nil {
		return err
	}
	if mode == "--json" {
		type entry struct {
			Alias   string `json:"alias"`
			Type    string `json:"type"`
			Version string `json:"version,omitempty"`
			Path    string `json:"path"`
			Current bool   `json:"current"`
		}
		entries := make([]entry, 0, len(cfg.Downloads))
		for _, d := range cfg.Downloads {
			entries = append(entries, entry{d.Alias, d.Type, d.Version, filepath.Join(dir, d.Alias), cfg.Current[d.Type] == d.Alias})
		}
		return json.NewEncoder(os.Stdout).Encode(entries)
	}
	if mode == "--quiet" {
		for _, d := range cfg.Downloads {
			fmt.Println(d.Alias)
		}
		return nil
	}
	if len(cfg.Downloads) == 0 {
		cli.Mutedf("No hay instalaciones registradas.\n")
		return nil
	}

	cli.PrintDownloads(cfg, dir)
	return nil
}

func runDoctor(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.Downloads) == 0 && len(cfg.Current) == 0 {
		cli.Infof("No hay instalaciones registradas para diagnosticar.\n")
		return nil
	}
	issues, err := doctor.Inspect(ctx, cfg)
	if err != nil {
		return err
	}
	if len(issues) == 0 {
		cli.Successf("No se encontraron problemas.\n")
		return nil
	}
	for _, issue := range issues {
		cli.Warnf("%s\n", issue)
	}
	return fmt.Errorf("doctor encontró %d problema(s)", len(issues))
}

func applyToolEnv(t tool.Tool, actualHome string) error {
	if err := env.Set(t.HomeVar, actualHome); err != nil {
		return err
	}
	for _, name := range t.ExtraHomeVars {
		if err := env.Set(name, actualHome); err != nil {
			return err
		}
	}
	if err := env.AddToPath(t.BinPath(actualHome)); err != nil {
		return err
	}
	return env.ReloadPath()
}

func removeFromPathForAlias(t tool.Tool, alias string) error {
	targetDir, err := config.HomeFor(alias)
	if err != nil {
		return err
	}
	actualHome, err := tool.DetectRoot(t, targetDir)
	if err != nil {
		return err
	}
	return env.RemoveFromPath(t.BinPath(actualHome))
}
