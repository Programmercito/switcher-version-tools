package download

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
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
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	dst := filepath.Join(tmp, "dest.txt")
	_, err := Fetch(context.Background(), srv.URL, dst, nil)
	if err == nil {
		t.Error("expected error for 404, got nil")
	}
	if requests.Load() != 1 {
		t.Errorf("requests = %d, want 1", requests.Load())
	}
}

func TestFetchRetriesTransientStatus(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("complete"))
	}))
	defer srv.Close()
	dst := filepath.Join(t.TempDir(), "out")
	cleanup, err := Fetch(context.Background(), srv.URL, dst, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "complete" || requests.Load() != 2 {
		t.Fatalf("content=%q requests=%d error=%v", got, requests.Load(), err)
	}
}

func TestRetryableStatuses(t *testing.T) {
	for _, tt := range []struct {
		status int
		want   bool
	}{
		{http.StatusTooManyRequests, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusInternalServerError, true},
		{http.StatusNotFound, false},
		{http.StatusForbidden, false},
	} {
		if got := retryable(statusError(tt.status)); got != tt.want {
			t.Errorf("status %d: retryable = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestFetchCancelDuringRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cancel()
		http.Error(w, "temporary", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	dst := filepath.Join(t.TempDir(), "out")
	_, err := Fetch(ctx, srv.URL, dst, nil)
	if err == nil {
		t.Fatal("expected cancellation")
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("partial download remains: %v", err)
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
