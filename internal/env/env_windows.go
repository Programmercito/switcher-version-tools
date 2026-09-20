//go:build windows

package env

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func init() {
	systemSet = windowsSystemSet
	systemAddToPath = windowsSystemAddToPath
	systemRemoveFromPath = windowsSystemRemoveFromPath
	systemReloadPath = windowsSystemReloadPath
}

func powershell() string {
	for _, shell := range []string{"powershell", "pwsh"} {
		if _, err := exec.LookPath(shell); err == nil {
			return shell
		}
	}
	return "powershell"
}

func runPS(script string) error {
	return exec.Command(powershell(), "-Command", script).Run()
}

func windowsSystemSet(name, value string) error {
	cmd := exec.Command("setx", name, value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("setx %s: %w", name, err)
	}
	return nil
}

func windowsSystemAddToPath(path string) error {
	ps := fmt.Sprintf(
		"$current = [Environment]::GetEnvironmentVariable('PATH', 'User'); "+
			"$parts = $current -split ';' | Where-Object { $_ -ne '' }; "+
			"if ($parts -notcontains '%s') { $parts += '%s' }; "+
			"[Environment]::SetEnvironmentVariable('PATH', ($parts -join ';'), 'User')",
		path, path,
	)
	if err := runPS(ps); err != nil {
		return fmt.Errorf("powershell add-path: %w", err)
	}
	return nil
}

func windowsSystemRemoveFromPath(path string) error {
	alt := strings.ReplaceAll(path, "\\", "/")
	ps := fmt.Sprintf(
		"$current = [Environment]::GetEnvironmentVariable('PATH', 'User'); "+
			"$parts = $current -split ';' | Where-Object { $_ -ne '%s' -and $_ -ne '%s' -and $_ -ne '' }; "+
			"[Environment]::SetEnvironmentVariable('PATH', ($parts -join ';'), 'User')",
		path, alt,
	)
	if err := runPS(ps); err != nil {
		return fmt.Errorf("powershell remove-path: %w", err)
	}
	return nil
}

func windowsSystemReloadPath() error {
	out, err := exec.Command(powershell(), "-Command",
		"$machine = [Environment]::GetEnvironmentVariable('PATH', 'Machine'); "+
			"$user = [Environment]::GetEnvironmentVariable('PATH', 'User'); "+
			"$machine + ';' + $user",
	).Output()
	if err != nil {
		return fmt.Errorf("powershell reload-path: %w", err)
	}
	_ = os.Setenv("PATH", strings.TrimSpace(string(out)))
	return nil
}
