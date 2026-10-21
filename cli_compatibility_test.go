package autohand

import "testing"

func TestCLICompatibilityRejectsOptionsWithoutInspectingArgumentValues(t *testing.T) {
	if err := validateCLIArguments([]string{"--mode", "rpc", "--auto-save-interval", "30"}); err == nil {
		t.Fatal("obsolete option accepted")
	}
	if err := validateCLIArguments([]string{"--mode", "rpc", "--sys-prompt", "--skills"}); err != nil {
		t.Fatalf("literal prompt value rejected: %v", err)
	}
}

func TestEphemeralModeIsExplicit(t *testing.T) {
	has := func(args []string) bool {
		for _, arg := range args {
			if arg == "--ephemeral" {
				return true
			}
		}
		return false
	}
	if has(buildCLIArgs(&Config{})) {
		t.Fatal("default unexpectedly ephemeral")
	}
	if !has(buildCLIArgs(&Config{Ephemeral: true})) {
		t.Fatal("ephemeral mode not forwarded")
	}
}
