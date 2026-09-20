package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/Programmercito/switcher-version-tools/internal/cli"
	"github.com/Programmercito/switcher-version-tools/internal/config"
	"github.com/Programmercito/switcher-version-tools/internal/download"
	"github.com/Programmercito/switcher-version-tools/internal/env"
	"github.com/Programmercito/switcher-version-tools/internal/extract"
	"github.com/Programmercito/switcher-version-tools/internal/tool"
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

	cli.PrintLogo()

	if len(os.Args) < 2 {
		cli.ShowHelp(currentVersion)
		os.Exit(1)
	}

	ui := cli.NewUI()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	arg1 := os.Args[1]

	switch {
	case arg1 == "help" || arg1 == "-h" || arg1 == "--help":
		cli.ShowHelp(currentVersion)
	case arg1 == "--version":
		fmt.Println(currentVersion)
	case arg1 == "list" || arg1 == "ls":
		if err := listCurrent(); err != nil {
			cli.Errorf("Error: %v\n", err)
			os.Exit(1)
		}
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

	if cfg.AddDownload(alias, tipo) {
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

func listCurrent() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Downloads) == 0 {
		cli.Mutedf("No hay instalaciones registradas.\n")
		return nil
	}

	cli.Infof("Instalaciones disponibles:\n")
	for _, d := range cfg.Downloads {
		marker := ""
		if current, ok := cfg.Current[d.Type]; ok && current == d.Alias {
			marker = " " + cli.SuccessStyle.Render("✓ actual")
		}
		fmt.Printf("  • %s (%s)%s\n", d.Alias, d.Type, marker)
	}
	return nil
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
