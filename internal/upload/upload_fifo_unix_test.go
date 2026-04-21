//go:build unix

package upload

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestDescribe_RejectsFIFO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pipe")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	// Run Describe in a goroutine so an accidental blocking read on the
	// FIFO would show up as a test timeout rather than hanging the harness.
	done := make(chan error, 1)
	go func() {
		_, err := Describe(path, Options{})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error for FIFO")
		}
		if !strings.Contains(err.Error(), "not a regular file") {
			t.Errorf("error = %v, want mention of regular file", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Describe hung on a FIFO")
	}
}

func TestUpload_RejectsFIFO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pipe")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	mock := &railsMock{}
	client, _ := setupServers(t, mock, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("S3 must not be reached for a FIFO path")
	}))

	done := make(chan error, 1)
	go func() {
		_, err := Upload(context.Background(), client, path, Options{})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not a regular file") {
			t.Errorf("error = %v, want mention of regular file", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Upload hung on a FIFO — pre-Open stat check missing")
	}
	if mock.presignCalls.Load() != 0 {
		t.Errorf("presign called for FIFO")
	}
}
