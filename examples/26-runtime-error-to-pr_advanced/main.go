// 26-runtime-error-to-pr_advanced demonstrates a production incident to pull
// request workflow with GitHub credentials, reproduction context, validation
// commands, branch naming, and PR instructions.
//
// Usage:
//
//	AUTOHAND_TARGET_REPO=/path/to/app GITHUB_TOKEN=... go run ./examples/26-runtime-error-to-pr_advanced
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

type GitHubCredentials struct {
	TokenEnvName string `json:"token_env_name"`
	Remote       string `json:"remote"`
	BaseBranch   string `json:"base_branch"`
	Repository   string `json:"repository,omitempty"`
}

type IncidentPacket struct {
	ID                  string            `json:"id"`
	Severity            string            `json:"severity"`
	Service             string            `json:"service"`
	FirstSeen           string            `json:"first_seen"`
	Release             string            `json:"release"`
	ErrorSignature      string            `json:"error_signature"`
	UserImpact          string            `json:"user_impact"`
	StackTrace          string            `json:"stack_trace"`
	Logs                []string          `json:"logs"`
	Request             map[string]any    `json:"request"`
	SuspectedFiles      []string          `json:"suspected_files"`
	ReproductionCommand string            `json:"reproduction_command"`
	ValidationCommands  []string          `json:"validation_commands"`
	Runtime             map[string]string `json:"runtime"`
}

func githubCredentialsFromEnv() (GitHubCredentials, error) {
	tokenEnvName := ""
	if os.Getenv("GITHUB_TOKEN") != "" {
		tokenEnvName = "GITHUB_TOKEN"
	} else if os.Getenv("GH_TOKEN") != "" {
		tokenEnvName = "GH_TOKEN"
	}
	if tokenEnvName == "" {
		return GitHubCredentials{}, fmt.Errorf("set GITHUB_TOKEN or GH_TOKEN before running this example")
	}

	return GitHubCredentials{
		TokenEnvName: tokenEnvName,
		Remote:       getenvDefault("AUTOHAND_GITHUB_REMOTE", "origin"),
		BaseBranch:   getenvDefault("AUTOHAND_GITHUB_BASE_BRANCH", "main"),
		Repository:   os.Getenv("GITHUB_REPOSITORY"),
	}, nil
}

func captureIncidentPacket() IncidentPacket {
	return IncidentPacket{
		ID:             "INC-2026-05-12-0417",
		Severity:       "sev2",
		Service:        "checkout-api",
		FirstSeen:      "2026-05-12T09:14:22Z",
		Release:        "checkout-api@2026.05.12.3",
		ErrorSignature: "panic: checkout discount failed: nil customer while replaying coupon idempotency key",
		UserImpact:     "Checkout returns HTTP 500 for guest customers using coupon replay from mobile clients.",
		StackTrace: strings.Join([]string{
			"panic: checkout discount failed: nil customer while replaying coupon idempotency key",
			"    checkout/discounts.CalculateDiscount src/checkout/discounts.go:42",
			"    checkout/payments.BuildPaymentIntent src/checkout/payment_intent.go:118",
			"    checkout/session.CreateCheckoutSession src/checkout/session.go:88",
		}, "\n"),
		Logs: []string{
			"level=error trace=trk_94 request_id=req_7f2 route=POST /checkout status=500 duration_ms=184",
			"level=warn trace=trk_94 idempotency_key=checkout:cart_live_9834:attempt_2 cache_status=miss",
			"level=info trace=trk_94 feature_flags=discount-v2,coupon-replay",
		},
		Request: map[string]any{
			"method": "POST",
			"path":   "/checkout",
			"payload": map[string]any{
				"cartId":         "cart_live_9834",
				"subtotal":       129,
				"customer":       nil,
				"coupon":         map[string]string{"code": "SPRING25", "source": "mobile-v5"},
				"idempotencyKey": "checkout:cart_live_9834:attempt_2",
			},
			"headers": map[string]string{
				"x-client-version": "ios/5.18.0",
				"x-request-id":     "req_7f2",
			},
		},
		SuspectedFiles: []string{
			"src/checkout/discounts.go",
			"src/checkout/payment_intent.go",
			"src/checkout/session.go",
			"src/checkout/session_test.go",
		},
		ReproductionCommand: "go test ./src/checkout -run TestCreateCheckoutSessionGuestCouponReplay -count=1",
		ValidationCommands: []string{
			"go test ./src/checkout -run TestCreateCheckoutSessionGuestCouponReplay -count=1",
			"go test ./...",
			"go vet ./...",
		},
		Runtime: map[string]string{
			"go":         "1.22",
			"feature":    "discount-v2,coupon-replay",
			"deployment": "checkout-api@2026.05.12.3",
		},
	}
}

func buildPrompt(incident IncidentPacket, github GitHubCredentials) (string, error) {
	incidentJSON, err := json.MarshalIndent(incident, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode incident: %w", err)
	}

	repoHint := "- Discover the GitHub repository from git remote output."
	if github.Repository != "" {
		repoHint = "- GitHub repository hint: " + github.Repository + "."
	}

	return strings.Join([]string{
		"You are a senior QA engineering agent responsible for converting production incidents into verified repair pull requests.",
		"",
		"GitHub credentials:",
		"- A GitHub token is available in the " + github.TokenEnvName + " environment variable. Do not print or commit the token.",
		"- Use git remote " + github.Remote + ".",
		"- Open the pull request against " + github.BaseBranch + ".",
		repoHint,
		"- Before pushing, run gh auth status or an equivalent non-secret auth check.",
		"",
		"Incident packet:",
		"```json",
		string(incidentJSON),
		"```",
		"",
		"Required workflow:",
		"1. Inspect the target repository and confirm the likely failing path.",
		"2. Reproduce the incident using the provided payload or nearest existing test harness.",
		"3. Fix the root cause, not just the thrown exception.",
		"4. Add a regression test covering guest checkout, coupon replay, and idempotency behavior.",
		"5. Run the focused test first, then the relevant validation commands.",
		"6. Create a branch named autohand/fix-checkout-incident-inc-2026-05-12-0417.",
		"7. Commit the fix with a clear message.",
		"8. Push the branch and open a pull request.",
		"9. In the PR body, include the incident id, error signature, files changed, tests run, and any residual risk.",
	}, "\n"), nil
}

func main() {
	ctx := context.Background()
	targetRepo := getenvDefault("AUTOHAND_TARGET_REPO", ".")

	github, err := githubCredentialsFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	prompt, err := buildPrompt(captureIncidentPacket(), github)
	if err != nil {
		log.Fatal(err)
	}

	agent, err := autohand.NewAgent(ctx, &autohand.Config{
		CWD:     targetRepo,
		CLIPath: os.Getenv("AUTOHAND_CLI_PATH"),
		Model:   os.Getenv("AUTOHAND_MODEL"),
	})
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}
	defer agent.Close()

	run, err := agent.Send(ctx, prompt, nil)
	if err != nil {
		log.Fatalf("send: %v", err)
	}

	events, err := run.Stream(ctx)
	if err != nil {
		log.Fatalf("stream: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.MessageUpdateEvent:
			fmt.Print(e.Delta)
		case autohand.ToolStartEvent:
			fmt.Printf("\n[tool] %s\n", e.ToolName)
		case autohand.PermissionRequestEvent:
			fmt.Printf("\n[permission] %s: %s\n", e.Tool, e.Description)
		case autohand.ErrorEvent:
			fmt.Printf("\n[error] %s\n", e.Message)
		}
	}

	result, err := run.Wait(ctx)
	if err != nil {
		log.Fatalf("wait: %v", err)
	}
	fmt.Printf("\n\nRun %s %s.\n", result.ID, result.Status)
}

func getenvDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
