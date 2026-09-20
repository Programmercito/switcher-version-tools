package extract

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZip(t *testing.T) {
	tmp := t.TempDir()
	zipFile := filepath.Join(tmp, "test.zip")
	createZip(t, zipFile, map[string]string{"app/bin/app.exe": "hello"})

	dst := filepath.Join(tmp, "out")
	if err := Extract(context.Background(), zipFile, dst, nil); err != nil {
		t.Fatal(err)
	}

	got := filepath.Join(dst, "app", "bin", "app.exe")
	if _, err := os.Stat(got); err != nil {
		t.Errorf("expected file %s: %v", got, err)
	}
}

func TestExtractZipSlipRejected(t *testing.T) {
	tmp := t.TempDir()
	zipFile := filepath.Join(tmp, "evil.zip")
	createZip(t, zipFile, map[string]string{"../evil.txt": "boom"})

	dst := filepath.Join(tmp, "out")
	if err := Extract(context.Background(), zipFile, dst, nil); err == nil {
		t.Error("expected error for zip-slip, got nil")
	}
}

func TestExtractTarGz(t *testing.T) {
	tmp := t.TempDir()
	tarFile := filepath.Join(tmp, "test.tar.gz")
	createTarGz(t, tarFile, []tarEntry{
		{Name: "app/", Typeflag: tar.TypeDir, Mode: 0755},
		{Name: "app/bin/", Typeflag: tar.TypeDir, Mode: 0755},
		{Name: "app/bin/app", Typeflag: tar.TypeReg, Content: "hello", Mode: 0755},
	})

	dst := filepath.Join(tmp, "out")
	if err := Extract(context.Background(), tarFile, dst, nil); err != nil {
		t.Fatal(err)
	}

	got := filepath.Join(dst, "app", "bin", "app")
	content, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("expected file %s: %v", got, err)
	}
	if string(content) != "hello" {
		t.Errorf("content = %q, want hello", string(content))
	}
}

func TestExtractTarGzSymlinkRejected(t *testing.T) {
	tmp := t.TempDir()
	tarFile := filepath.Join(tmp, "evil.tar.gz")
	createTarGz(t, tarFile, []tarEntry{
		{Name: "link", Typeflag: tar.TypeSymlink, LinkTarget: "/etc/passwd"},
	})

	dst := filepath.Join(tmp, "out")
	if err := Extract(context.Background(), tarFile, dst, nil); err == nil {
		t.Error("expected error for symlink, got nil")
	}
}

func createZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	w, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	zw := zip.NewWriter(w)
	for name, content := range files {
		f, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

type tarEntry struct {
	Name       string
	Typeflag   byte
	Content    string
	LinkTarget string
	Mode       int64
}

func createTarGz(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.Name,
			Mode:     e.Mode,
			Size:     int64(len(e.Content)),
			Typeflag: e.Typeflag,
			Linkname: e.LinkTarget,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if e.Typeflag == tar.TypeReg && e.Content != "" {
			if _, err := tw.Write([]byte(e.Content)); err != nil {
				t.Fatal(err)
			}
		}
	}
}
