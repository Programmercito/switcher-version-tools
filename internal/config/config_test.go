package config

import (
	"testing"
)

func patchHome(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	old := userHomeDir
	userHomeDir = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { userHomeDir = old })
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	patchHome(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil")
	}
	if len(cfg.Downloads) != 0 {
		t.Errorf("downloads = %v, want empty", cfg.Downloads)
	}
	if len(cfg.Current) != 0 {
		t.Errorf("current = %v, want empty", cfg.Current)
	}
}

func TestSaveAndLoad(t *testing.T) {
	patchHome(t)

	cfg := &Config{Current: map[string]string{}}
	if !cfg.AddDownload("jdk17", "java") {
		t.Fatal("first AddDownload should succeed")
	}
	if !cfg.AddDownload("php82", "php") {
		t.Fatal("second AddDownload should succeed")
	}
	cfg.Current["java"] = "jdk17"

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Downloads) != 2 {
		t.Errorf("downloads len = %d, want 2", len(loaded.Downloads))
	}
	if loaded.Current["java"] != "jdk17" {
		t.Errorf("current[java] = %s, want jdk17", loaded.Current["java"])
	}
}

func TestAddDownloadDuplicate(t *testing.T) {
	cfg := &Config{Current: map[string]string{}}
	if !cfg.AddDownload("a", "java") {
		t.Error("first AddDownload should return true")
	}
	if cfg.AddDownload("a", "java") {
		t.Error("duplicate AddDownload should return false")
	}
}

func TestFindAndRemoveDownload(t *testing.T) {
	cfg := &Config{
		Downloads: []Download{{Alias: "a", Type: "java"}, {Alias: "b", Type: "php"}},
		Current:   map[string]string{"java": "a", "php": "b"},
	}

	d, ok := cfg.FindDownload("b")
	if !ok || d.Type != "php" {
		t.Fatalf("FindDownload(b) = %+v, %v", d, ok)
	}

	if !cfg.RemoveDownload("b") {
		t.Error("RemoveDownload(b) should return true")
	}
	if _, ok := cfg.FindDownload("b"); ok {
		t.Error("b should be removed")
	}
	if _, ok := cfg.Current["php"]; ok {
		t.Error("current[php] should be removed")
	}
	if cfg.RemoveDownload("b") {
		t.Error("second RemoveDownload should return false")
	}
}
