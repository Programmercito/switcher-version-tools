package download

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchLocalFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "source.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(tmp, "dest.txt")
	cleanup, err := Fetch(context.Background(), src, dst, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello" {
		t.Errorf("content = %q, want hello", content)
	}
}

func TestFetchRemoteFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("remote"))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	dst := filepath.Join(tmp, "dest.txt")
	cleanup, err := Fetch(context.Background(), srv.URL, dst, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "remote" {
		t.Errorf("content = %q, want remote", content)
	}
}

func TestFetchRemoteNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	dst := filepath.Join(tmp, "dest.txt")
	_, err := Fetch(context.Background(), srv.URL, dst, nil)
	if err == nil {
		t.Error("expected error for 404, got nil")
	}
}

func TestFetchReportsTotalFromIdentityHeader(t *testing.T) {
	body := []byte("0123456789")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept-Encoding"); got != "identity" {
			t.Errorf("Accept-Encoding = %q, want identity", got)
		}
		w.Header().Set("X-Identity-Content-Length", "10")
		w.Header().Set("Transfer-Encoding", "chunked")
		_, _ = w.Write(body[:5])
		w.(http.Flusher).Flush()
		_, _ = w.Write(body[5:])
	}))
	defer srv.Close()

	var lastTotal int64
	cleanup, err := Fetch(context.Background(), srv.URL, filepath.Join(t.TempDir(), "out"), func(_, total int64) { lastTotal = total })
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if lastTotal != 10 {
		t.Fatalf("total = %d, want 10", lastTotal)
	}
}
