package autohand

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRequestBeforeStartReturnsLifecycleError(t *testing.T) {
	transport := NewTransport(&Config{})
	_, err := transport.Request(context.Background(), "autohand.getState", map[string]interface{}{})
	if !errors.Is(err, ErrTransportNotStarted) {
		t.Fatalf("Request error = %v, want ErrTransportNotStarted", err)
	}
}

func TestStdoutEOFFailsAndDrainsPendingRequest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake CLI")
	}
	cli := filepath.Join(t.TempDir(), "autohand")
	if err := os.WriteFile(cli, []byte("#!/bin/sh\nIFS= read -r line || exit 1\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := &Config{CLIPath: cli, Timeout: 5_000}
	transport := NewTransport(config)
	if err := transport.Start(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	_, err := transport.Request(context.Background(), "autohand.neverReplies", map[string]interface{}{})
	if !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Request error = %v, want ErrTransportClosed", err)
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("EOF resolution took %s; request waited toward its timeout", elapsed)
	}
	transport.mu.Lock()
	pending := len(transport.callbacks)
	transport.mu.Unlock()
	if pending != 0 {
		t.Fatalf("callbacks retained after EOF: %d", pending)
	}
	if err := transport.Stop(); err != nil {
		t.Fatal(err)
	}
}
