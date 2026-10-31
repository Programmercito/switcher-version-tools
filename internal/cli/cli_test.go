package cli

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Programmercito/switcher-version-tools/internal/config"
	"github.com/Programmercito/switcher-version-tools/internal/tool"
)

func TestSupportedToolsHelpIncludesRegisteredToolsAndHomeVariables(t *testing.T) {
	output := supportedToolsHelp()

	for _, name := range tool.Supported() {
		registered, ok := tool.Get(name)
		if !ok {
			t.Fatalf("tool %q is not registered", name)
		}

		if !strings.Contains(output, registered.HomeVar) {
			t.Errorf("supported tools help does not include %s", registered.HomeVar)
		}
		for _, homeVar := range registered.ExtraHomeVars {
			if !strings.Contains(output, homeVar) {
				t.Errorf("supported tools help does not include %s", homeVar)
			}
		}
	}
}

func TestDownloadProgressIncludesSpeedAndETA(t *testing.T) {
	for _, tt := range []struct {
		name    string
		total   int64
		wantETA bool
	}{
		{"known size", 1024, true},
		{"unknown size", 0, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			previous := os.Stdout
			os.Stdout = w
			ui := NewUI()
			ui.downloadStart = time.Now().Add(-time.Second)
			ui.PrintDownloadProgress(512, tt.total)
			os.Stdout = previous
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			out, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.Close(); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(out), "/s") || strings.Contains(string(out), "restante") != tt.wantETA {
				t.Fatalf("progress = %q", out)
			}
		})
	}
}

func TestToolDisplayName(t *testing.T) {
	tests := map[string]string{
		"java":   "Java (JDK)",
		"node":   "Node.js",
		"php":    "PHP",
		"go":     "Go",
		"custom": "Custom",
	}
	for name, want := range tests {
		if got := toolDisplayName(name); got != want {
			t.Errorf("toolDisplayName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestDownloadsByTypeSortsToolsAndAliases(t *testing.T) {
	types, groups := downloadsByType([]config.Download{
		{Alias: "node20", Type: "node"},
		{Alias: "jdk21", Type: "java"},
		{Alias: "jdk17", Type: "java"},
	})

	if want := []string{"java", "node"}; !strings.EqualFold(strings.Join(types, ","), strings.Join(want, ",")) {
		t.Fatalf("types = %v, want %v", types, want)
	}
	if got := groups["java"][0].Alias; got != "jdk17" {
		t.Errorf("first java alias = %q, want jdk17", got)
	}
}
