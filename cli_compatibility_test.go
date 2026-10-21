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

func TestAutoCommitUsesLongFlag(t *testing.T) {
	args := buildCLIArgs(&Config{AutoCommit: true})
	found := false
	for _, arg := range args {
		if arg == "-c" {
			t.Fatal("auto-commit emitted continue")
		}
		if arg == "--auto-commit" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing long auto-commit flag")
	}
	if err := validateCLIArguments(args); err != nil {
		t.Fatal(err)
	}
}
