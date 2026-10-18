package cli

import (
	"strings"
	"testing"

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
