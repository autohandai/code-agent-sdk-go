// 20-sdlc-discovery-plan demonstrates SDLC workflow: discovery and planning.
//
// The agent runs in plan mode so it can inspect the project and produce
// an implementation plan without performing write operations.
//
// Usage:
//
//	go run ./examples/20-sdlc-discovery-plan
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func writeEvent(event autohand.Event) {
	switch e := event.(type) {
	case autohand.MessageUpdateEvent:
		fmt.Print(e.Delta)
	case autohand.ToolStartEvent:
		fmt.Printf("\n[tool:start] %s\n", e.ToolName)
	case autohand.ToolEndEvent:
		fmt.Printf("[tool:end] %s success=%v\n", e.ToolName, e.Success)
	case autohand.PermissionRequestEvent:
		fmt.Printf("\n[permission] %s: %s\n", e.Tool, e.Description)
	case autohand.ErrorEvent:
		fmt.Printf("\n[error] %s\n", e.Message)
	}
}

func main() {
	ctx := context.Background()

	cliPath := os.Getenv("AUTOHAND_CLI_PATH")
	cfg := &autohand.Config{
		CWD:   ".",
		Model: os.Getenv("AUTOHAND_MODEL"),
	}
	if cliPath != "" {
		cfg.CLIPath = cliPath
	}

	sdk := autohand.NewSDK(cfg)

	if err := sdk.Start(ctx); err != nil {
		log.Fatalf("start SDK: %v", err)
	}

	message := []string{
		"We are in discovery for a production Go SDK change.",
		"Inspect the repository and produce an SDLC plan only.",
		"Do not edit files.",
		"Include scope, risks, test strategy, rollout steps, and explicit non-goals.",
	}

	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
		Message: message[0] + "\n" + message[1] + "\n" + message[2] + "\n" + message[3],
	})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}

	for event := range events {
		writeEvent(event)
	}

	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
}
