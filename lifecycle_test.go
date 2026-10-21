package autohand

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func writeLifecycleCLI(t *testing.T, startupError bool) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake CLI")
	}
	startupReply := `{"ok":true}`
	if startupError {
		startupReply = `null,"error":{"code":-32000,"message":"readiness failure"}`
	}
	script := `#!/bin/sh
while IFS= read -r line; do
  id=$(printf '%s\n' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  case "$line" in
    *autohand.getState*) printf '{"jsonrpc":"2.0","id":%s,"result":` + startupReply + `}\n' "$id" ;;
    *) printf '{"jsonrpc":"2.0","id":%s,"result":{}}\n' "$id" ;;
  esac
done
`
	path := filepath.Join(t.TempDir(), "autohand")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestStartupFailureRollsBackAndRemainsRetryable(t *testing.T) {
	yes := true
	sdk := NewSDK(&Config{
		CLIPath:  writeLifecycleCLI(t, true),
		Features: &FeatureFlagSettings{SlashGoal: &yes},
		Timeout:  500,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sdk.Start(ctx); err == nil {
		t.Fatal("Start unexpectedly succeeded")
	}
	if sdk.IsStarted() || sdk.IsConnected() {
		t.Fatalf("failed startup left lifecycle active: started=%v connected=%v", sdk.IsStarted(), sdk.IsConnected())
	}
	if err := sdk.Start(ctx); err == nil {
		t.Fatal("retry unexpectedly skipped failed initialization")
	}
}

func TestStopClearsConnectedState(t *testing.T) {
	sdk := NewSDK(&Config{CLIPath: writeLifecycleCLI(t, false), Timeout: 500})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sdk.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if !sdk.IsConnected() {
		t.Fatal("SDK did not report a connected child")
	}
	if err := sdk.Stop(); err != nil {
		t.Fatal(err)
	}
	if sdk.IsConnected() {
		t.Fatal("SDK remained connected after Stop")
	}
}
