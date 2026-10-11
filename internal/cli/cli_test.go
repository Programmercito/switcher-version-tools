package cli

import (
	"strings"
	"testing"

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
