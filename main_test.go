package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCommandOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the CLI")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "switchtool")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	home := filepath.Join(dir, "home")
	cfgDir := filepath.Join(home, "switchjdk")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "downloads.json"), []byte(`{"downloads":[{"alias":"jdk17","type":"java","version":"17.0.9"}],"current":{"java":"jdk17"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, string, error) {
		t.Helper()
		cmd := exec.Command(binary, args...)
		if runtime.GOOS == "windows" {
			cmd.Env = append(os.Environ(), "USERPROFILE="+home)
		} else {
			cmd.Env = append(os.Environ(), "HOME="+home)
		}
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		return stdout.String(), stderr.String(), err
	}
	t.Run("version is a single line", func(t *testing.T) {
		out, stderr, err := run("--version")
		if err != nil || stderr != "" || strings.Count(out, "\n") != 1 || strings.TrimSpace(out) == "" {
			t.Fatalf("out=%q stderr=%q err=%v", out, stderr, err)
		}
	})
	t.Run("json shape and paths", func(t *testing.T) {
		out, stderr, err := run("list", "--json")
		var entries []struct {
			Alias   string `json:"alias"`
			Type    string `json:"type"`
			Version string `json:"version"`
			Path    string `json:"path"`
			Current bool   `json:"current"`
		}
		if err != nil || stderr != "" {
			t.Fatalf("out=%q stderr=%q err=%v", out, stderr, err)
		}
		if err := json.Unmarshal([]byte(out), &entries); err != nil || len(entries) != 1 {
			t.Fatalf("out=%q error=%v", out, err)
		}
		entry := entries[0]
		if entry.Alias != "jdk17" || entry.Type != "java" || entry.Version != "17.0.9" || entry.Path != filepath.Join(cfgDir, "jdk17") || !entry.Current {
			t.Fatalf("entry=%+v", entry)
		}
	})
	t.Run("quiet emits aliases only", func(t *testing.T) {
		out, stderr, err := run("--no-color", "list", "--quiet")
		if err != nil || stderr != "" || out != "jdk17\n" {
			t.Fatalf("out=%q stderr=%q err=%v", out, stderr, err)
		}
	})
	t.Run("no color and list path", func(t *testing.T) {
		out, stderr, err := run("list", "--no-color")
		if err != nil || stderr != "" || strings.Contains(out, "\x1b[") || !strings.Contains(out, filepath.Join(cfgDir, "jdk17")) {
			t.Fatalf("out=%q stderr=%q err=%v", out, stderr, err)
		}
	})
	t.Run("incompatible list flags fail", func(t *testing.T) {
		out, stderr, err := run("list", "--json", "--quiet")
		if err == nil || out != "" || !strings.Contains(stderr, "uso:") {
			t.Fatalf("out=%q stderr=%q err=%v", out, stderr, err)
		}
	})
	t.Run("doctor reports absent installation", func(t *testing.T) {
		out, stderr, err := run("doctor", "--no-color")
		if err == nil || !strings.Contains(out, "instalación inaccesible") || !strings.Contains(stderr, "doctor encontró") {
			t.Fatalf("out=%q stderr=%q err=%v", out, stderr, err)
		}
	})
}
