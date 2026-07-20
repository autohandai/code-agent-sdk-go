//go:build !windows

package autohand

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type currentCLIFixture struct {
	sdk     *SDK
	ctx     context.Context
	logPath string
}

func newCurrentCLIFixture(t *testing.T, result, notification string) *currentCLIFixture {
	t.Helper()
	dir := t.TempDir()
	cli := filepath.Join(dir, "autohand")
	logPath := filepath.Join(dir, "requests.log")
	script := `#!/bin/sh
while IFS= read -r line; do
  printf '%s\n' "$line" >> "$AUTOHAND_TEST_LOG"
  id=$(printf '%s\n' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  case "$line" in
    *autohand.getState*) response='{}' ;;
    *autohand.prompt*)
      if [ -n "$AUTOHAND_TEST_NOTIFICATION" ]; then
        printf '%s\n' "$AUTOHAND_TEST_NOTIFICATION"
      fi
      response='{"success":true}'
      ;;
    *) response="$AUTOHAND_TEST_RESULT" ;;
  esac
  printf '{"jsonrpc":"2.0","id":%s,"result":%s}\n' "$id" "$response"
done
`
	if err := os.WriteFile(cli, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	sdk := NewSDK(&Config{
		CLIPath: cli,
		Timeout: 2_000,
		Env: map[string]string{
			"AUTOHAND_TEST_LOG":          logPath,
			"AUTOHAND_TEST_RESULT":       result,
			"AUTOHAND_TEST_NOTIFICATION": notification,
		},
	})
	if err := sdk.Start(ctx); err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = sdk.Stop()
		cancel()
	})
	return &currentCLIFixture{sdk: sdk, ctx: ctx, logPath: logPath}
}

func (f *currentCLIFixture) assertRequest(t *testing.T, method string, params ...string) {
	t.Helper()
	contents, err := os.ReadFile(f.logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(contents)
	if !strings.Contains(log, `"method":"`+method+`"`) {
		t.Fatalf("request log does not contain %s: %s", method, log)
	}
	for _, param := range params {
		if !strings.Contains(log, param) {
			t.Fatalf("request log does not contain %s: %s", param, log)
		}
	}
}

func TestAcknowledgePermissionE2E(t *testing.T) {
	fixture := newCurrentCLIFixture(t, `{"success":true}`, "")
	result, err := fixture.sdk.AcknowledgePermission(fixture.ctx, "permission-1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
	fixture.assertRequest(t, "autohand.permissionAcknowledged", `"requestId":"permission-1"`)

	if _, err := fixture.sdk.AcknowledgePermission(fixture.ctx, "  "); err == nil {
		t.Fatal("expected blank request ID to fail before transport")
	}
}
