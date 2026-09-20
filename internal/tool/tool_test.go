package tool

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryContainsExpectedTools(t *testing.T) {
	expected := []string{"java", "maven", "gradle", "php", "go", "node"}
	for _, name := range expected {
		if _, ok := Get(name); !ok {
			t.Errorf("Get(%q) not found", name)
		}
	}
}

func TestIsValid(t *testing.T) {
	if !IsValid("java") {
		t.Error("java should be valid")
	}
	if IsValid("rust") {
		t.Error("rust should not be valid")
	}
}

func TestDetectRootSingleSubdirectory(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "jdk-17")
	if err := os.MkdirAll(filepath.Join(sub, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	got, err := DetectRoot(Registry["java"], tmp)
	if err != nil {
		t.Fatal(err)
	}
	if got != sub {
		t.Errorf("DetectRoot = %s, want %s", got, sub)
	}
}

func TestDetectRootPHP(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "php.exe"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	got, err := DetectRoot(Registry["php"], tmp)
	if err != nil {
		t.Fatal(err)
	}
	if got != tmp {
		t.Errorf("DetectRoot = %s, want %s", got, tmp)
	}
}

func TestDetectRootPHPInSubdir(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "php-8.2")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(sub, "php.exe"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	got, err := DetectRoot(Registry["php"], tmp)
	if err != nil {
		t.Fatal(err)
	}
	if got != sub {
		t.Errorf("DetectRoot = %s, want %s", got, sub)
	}
}

func TestBinPath(t *testing.T) {
	java := Registry["java"]
	want := filepath.Join("C:\\java", "bin")
	if got := java.BinPath("C:\\java"); got != want {
		t.Errorf("java BinPath = %s, want %s", got, want)
	}

	php := Registry["php"]
	if got := php.BinPath("C:\\php"); got != "C:\\php" {
		t.Errorf("php BinPath = %s, want C:\\php", got)
	}
}
