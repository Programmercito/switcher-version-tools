package env

import (
	"os"
	"strings"
	"testing"
)

func disableSystemCalls(t *testing.T) {
	t.Helper()
	oldSet := systemSet
	oldAdd := systemAddToPath
	oldRemove := systemRemoveFromPath
	oldReload := systemReloadPath
	t.Cleanup(func() {
		systemSet = oldSet
		systemAddToPath = oldAdd
		systemRemoveFromPath = oldRemove
		systemReloadPath = oldReload
	})

	systemSet = func(name, value string) error { return nil }
	systemAddToPath = func(path string) error { return nil }
	systemRemoveFromPath = func(path string) error { return nil }
	systemReloadPath = func() error { return nil }
}

func restorePath(t *testing.T, original string) {
	t.Cleanup(func() { _ = os.Setenv("PATH", original) })
}

func TestAddToPath(t *testing.T) {
	disableSystemCalls(t)
	orig := os.Getenv("PATH")
	restorePath(t, orig)
	_ = os.Setenv("PATH", "")

	if err := AddToPath("C:\\foo"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(os.Getenv("PATH"), "C:\\foo") {
		t.Errorf("PATH = %q, should contain C:\\foo", os.Getenv("PATH"))
	}

	// Duplicado normalizado a backslash: no debe agregarse dos veces.
	if err := AddToPath("C:/foo"); err != nil {
		t.Fatal(err)
	}
	if strings.Count(os.Getenv("PATH"), "C:\\foo") != 1 {
		t.Errorf("PATH = %q, duplicate added", os.Getenv("PATH"))
	}
}

func TestRemoveFromPath(t *testing.T) {
	disableSystemCalls(t)
	orig := os.Getenv("PATH")
	restorePath(t, orig)
	sep := string(os.PathListSeparator)
	_ = os.Setenv("PATH", strings.Join([]string{"C:\\a", "C:\\b", "C:\\c"}, sep))

	if err := RemoveFromPath("C:/b"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(os.Getenv("PATH"), "C:\\b") {
		t.Errorf("PATH = %q, should not contain C:\\b", os.Getenv("PATH"))
	}
}

func TestSet(t *testing.T) {
	disableSystemCalls(t)
	defer os.Unsetenv("SWITCHTOOL_TEST_VAR")

	if err := Set("SWITCHTOOL_TEST_VAR", "value"); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("SWITCHTOOL_TEST_VAR") != "value" {
		t.Errorf("env = %q, want value", os.Getenv("SWITCHTOOL_TEST_VAR"))
	}
}
