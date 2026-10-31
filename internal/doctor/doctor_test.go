package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Programmercito/switcher-version-tools/internal/config"
	"github.com/Programmercito/switcher-version-tools/internal/env"
	"github.com/Programmercito/switcher-version-tools/internal/tool"
)

func TestInspect(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	} else {
		t.Setenv("HOME", home)
	}
	dir := filepath.Join(home, "switchjdk", "jdk17")
	java, _ := tool.Get("java")
	executable := filepath.Join(dir, java.Executable)
	if err := os.MkdirAll(filepath.Dir(executable), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Downloads: []config.Download{{Alias: "jdk17", Type: "java", Version: "17.0.9"}},
		Current:   map[string]string{"java": "jdk17"},
	}
	snapshot := env.Snapshot{User: map[string]string{
		"JAVA_HOME": dir, "JDK_HOME": dir, "PATH": java.BinPath(dir),
	}, Process: map[string]string{"JAVA_HOME": dir, "JDK_HOME": dir}}
	for _, tt := range []struct {
		name, found, installedVersion, runningVersion, want string
		change                                              func(*env.Snapshot)
	}{
		{name: "healthy", found: executable, installedVersion: "17.0.9"},
		{name: "incorrect home", found: executable, installedVersion: "17.0.9", want: "JAVA_HOME", change: func(s *env.Snapshot) { s.User["JAVA_HOME"] = "wrong" }},
		{name: "stale terminal", found: executable, installedVersion: "17.0.9", want: "JAVA_HOME de esta terminal", change: func(s *env.Snapshot) { s.Process["JAVA_HOME"] = "old" }},
		{name: "missing user path", found: executable, installedVersion: "17.0.9", want: "PATH de usuario", change: func(s *env.Snapshot) { s.User["PATH"] = "" }},
		{name: "version changed", found: executable, installedVersion: "17.0.10", want: "versión instalada 17.0.10"},
		{name: "path conflict", found: filepath.Join(home, "other", "java.exe"), installedVersion: "17.0.9", runningVersion: "21.0.1", want: "versión en PATH 21.0.1"},
		{name: "missing command", installedVersion: "17.0.9", want: "no disponible en el PATH"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := env.Snapshot{User: make(map[string]string), Process: make(map[string]string)}
			for k, v := range snapshot.User {
				s.User[k] = v
			}
			for k, v := range snapshot.Process {
				s.Process[k] = v
			}
			if tt.change != nil {
				tt.change(&s)
			}
			look := func(string) (string, error) {
				if tt.found == "" {
					return "", errors.New("not found")
				}
				return tt.found, nil
			}
			detect := func(_ context.Context, _ tool.Tool, home string) (string, error) {
				if home == "" {
					return tt.runningVersion, nil
				}
				return tt.installedVersion, nil
			}
			issues := inspect(context.Background(), cfg, s, look, detect)
			if tt.want == "" && len(issues) != 0 || tt.want != "" && !strings.Contains(strings.Join(issues, "\n"), tt.want) {
				t.Fatalf("issues = %v, want %q", issues, tt.want)
			}
		})
	}
}

func TestInspectMissingInstallAndUnregisteredCurrent(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	} else {
		t.Setenv("HOME", home)
	}
	cfg := &config.Config{Downloads: []config.Download{{Alias: "missing", Type: "java"}}, Current: map[string]string{"node": "unknown"}}
	issues := inspect(context.Background(), cfg, env.Snapshot{}, func(string) (string, error) { return "", nil }, nil)
	text := strings.Join(issues, "\n")
	if !strings.Contains(text, "instalación inaccesible") || !strings.Contains(text, "no está registrado") {
		t.Fatalf("issues = %v", issues)
	}
}
