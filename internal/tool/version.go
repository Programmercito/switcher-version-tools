package tool

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var versionPattern = regexp.MustCompile(`(?i)(?:^|[\s=:/("'` + "`" + `])(?:v|go)?(\d+\.\d+(?:\.\d+)?(?:[-+._][0-9a-z.-]+)?)`)

// DetectVersion ejecuta el comando de versión de la herramienta instalada y
// extrae una versión legible de su salida estándar o de error.
func DetectVersion(ctx context.Context, t Tool, home string) (string, error) {
	if len(t.VersionCommand) == 0 {
		return "", fmt.Errorf("la herramienta %s no tiene comando de versión configurado", t.Name)
	}

	args := append([]string(nil), t.VersionCommand[1:]...)
	command := t.VersionCommand[0]
	if !filepath.IsAbs(command) {
		command = filepath.Join(home, command)
	}

	var output bytes.Buffer
	commandArgs := append([]string(nil), args...)
	if strings.EqualFold(filepath.Ext(command), ".cmd") || strings.EqualFold(filepath.Ext(command), ".bat") {
		commandArgs = append([]string{"/c", command}, commandArgs...)
		command = "cmd"
	}
	cmd := exec.CommandContext(ctx, command, commandArgs...)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		text := strings.TrimSpace(output.String())
		if text == "" {
			return "", fmt.Errorf("no se pudo ejecutar %s: %w", command, err)
		}
		if version := ParseVersion(text); version != "" {
			return version, nil
		}
		return "", fmt.Errorf("el comando %s falló: %w", command, err)
	}

	version := ParseVersion(output.String())
	if version == "" {
		return "", fmt.Errorf("no se encontró una versión en la salida de %s", command)
	}
	return version, nil
}

// ParseVersion obtiene la primera versión semántica común en la salida de una
// herramienta, por ejemplo "17.0.9", "8.2.10" o "20.11.1".
func ParseVersion(output string) string {
	match := versionPattern.FindStringSubmatch(output)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}
