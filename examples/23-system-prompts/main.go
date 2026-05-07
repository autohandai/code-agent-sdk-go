// 23-system-prompts demonstrates system prompt configuration.
//
// Use AppendSystemPrompt for normal SDK integrations. Use SetSystemPrompt only
// when you intentionally own the complete agent contract.
//
// Usage:
//
//	go run ./examples/23-system-prompts
//	AUTOHAND_PROMPT_MODE=replace go run ./examples/23-system-prompts
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func createSDK() *autohand.SDK {
	cliPath := os.Getenv("AUTOHAND_CLI_PATH")
	cfg := &autohand.Config{
		CWD:   ".",
		Model: os.Getenv("AUTOHAND_MODEL"),
	}
	if cliPath != "" {
		cfg.CLIPath = cliPath
	}

	mode := os.Getenv("AUTOHAND_PROMPT_MODE")
	if mode == "replace" {
		cfg.SysPrompt = []string{
			"You are Autohand Code operating as a release-review agent.",
			"Inspect the repository carefully.",
			"Return concise findings with file references and verification steps.",
		}[0] + "\n" + []string{
			"You are Autohand Code operating as a release-review agent.",
			"Inspect the repository carefully.",
			"Return concise findings with file references and verification steps.",
		}[1] + "\n" + []string{
			"You are Autohand Code operating as a release-review agent.",
			"Inspect the repository carefully.",
			"Return concise findings with file references and verification steps.",
		}[2]
	} else {
		cfg.AppendSysPrompt = []string{
			"For this SDK repository, prefer standard Go commands.",
			"Call out permission-sensitive operations before recommending execution.",
			"Keep responses focused on Go SDK API design.",
		}[0] + "\n" + []string{
			"For this SDK repository, prefer standard Go commands.",
			"Call out permission-sensitive operations before recommending execution.",
			"Keep responses focused on Go SDK API design.",
		}[1] + "\n" + []string{
			"For this SDK repository, prefer standard Go commands.",
			"Call out permission-sensitive operations before recommending execution.",
			"Keep responses focused on Go SDK API design.",
		}[2]
	}

	return autohand.NewSDK(cfg)
}

func main() {
	ctx := context.Background()

	sdk := createSDK()

	if err := sdk.Start(ctx); err != nil {
		log.Fatalf("start SDK: %v", err)
	}

	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
		Message: "Review the public SDK surface for system prompt ergonomics.",
	})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.MessageUpdateEvent:
			fmt.Print(e.Delta)
		case autohand.PermissionRequestEvent:
			fmt.Printf("\n[permission] %s: %s\n", e.Tool, e.Description)
			if err := sdk.DenyPermission(ctx, e.RequestID, autohand.ScopeOnce); err != nil {
				log.Printf("deny: %v", err)
			}
		case autohand.AgentEndEvent:
			_ = sdk.Close()
			return
		}
	}

	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
}
