package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Programmercito/switcher-version-tools/internal/config"
	"github.com/Programmercito/switcher-version-tools/internal/tool"
)

func TestReadProject(t *testing.T) {
	for _, tt := range []struct {
		name, contents string
		valid          bool
	}{
		{"valid", `{"tools":{"java":"jdk17"}}`, true},
		{"empty tools", `{"tools":{}}`, false},
		{"unknown field", `{"tools":{"java":"jdk17"},"typo":1}`, false},
		{"trailing data", `{"tools":{"java":"jdk17"}} {}`, false},
		{"malformed", `{`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "switchtool.json")
			if err := os.WriteFile(path, []byte(tt.contents), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := readProject(path)
			if (err == nil) != tt.valid {
				t.Fatalf("error = %v, valid = %v", err, tt.valid)
			}
		})
	}
}

func TestProjectAliasesValidatesBeforeActivation(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	} else {
		t.Setenv("HOME", home)
	}
	java, _ := tool.Get("java")
	exe := filepath.Join(home, "switchjdk", "jdk17", java.Executable)
	if err := os.MkdirAll(filepath.Dir(exe), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Downloads: []config.Download{{Alias: "jdk17", Type: "java"}}}
	aliases, err := projectAliases(projectConfig{Tools: map[string]string{"java": "jdk17"}}, cfg)
	if err != nil || len(aliases) != 1 || aliases[0] != "jdk17" {
		t.Fatalf("aliases=%v err=%v", aliases, err)
	}
	for _, p := range []projectConfig{
		{Tools: map[string]string{"java": "missing"}},
		{Tools: map[string]string{"node": "jdk17"}},
		{Tools: map[string]string{"unknown": "jdk17"}},
	} {
		if _, err := projectAliases(p, cfg); err == nil {
			t.Errorf("expected error for %v", p.Tools)
		}
	}
	if err := os.Remove(exe); err != nil {
		t.Fatal(err)
	}
	_, err = projectAliases(projectConfig{Tools: map[string]string{"java": "jdk17"}}, cfg)
	if err == nil || !strings.Contains(err.Error(), "ejecutable") {
		t.Fatalf("missing executable error = %v", err)
	}
}
