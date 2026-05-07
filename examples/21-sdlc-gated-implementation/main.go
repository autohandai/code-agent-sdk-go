// 21-sdlc-gated-implementation demonstrates plan first, execute only after gate.
//
// By default this example stops after planning. Set AUTOHAND_EXECUTE_PLAN=1
// to disable plan mode and ask the agent to implement the approved plan.
//
// Usage:
//
//	AUTOHAND_EXECUTE_PLAN=1 go run ./examples/21-sdlc-gated-implementation
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
	case autohand.ToolUpdateEvent:
		fmt.Print(e.Output)
	case autohand.ToolEndEvent:
		fmt.Printf("[tool:end] %s success=%v\n", e.ToolName, e.Success)
	case autohand.PermissionRequestEvent:
		fmt.Printf("\n[permission] %s: %s\n", e.Tool, e.Description)
	case autohand.ErrorEvent:
		fmt.Printf("\n[error] %s\n", e.Message)
	}
}

func streamPrompt(ctx context.Context, sdk *autohand.SDK, message string) {
	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{Message: message})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}
	for event := range events {
		writeEvent(event)
	}
}

func main() {
	ctx := context.Background()

	executePlan := os.Getenv("AUTOHAND_EXECUTE_PLAN") == "1"

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

	planPrompt := []string{
		"Create an implementation plan for the requested SDK change.",
		"Use repository inspection only.",
		"Do not edit files in this planning pass.",
		"Return numbered steps with test coverage and rollback notes.",
	}

	fmt.Println("--- planning ---\n")
	streamPrompt(ctx, sdk, planPrompt[0]+"\n"+planPrompt[1]+"\n"+planPrompt[2]+"\n"+planPrompt[3])

	if !executePlan {
		fmt.Println("\n--- gate closed ---")
		fmt.Println("Set AUTOHAND_EXECUTE_PLAN=1 after reviewing the plan to run the implementation phase.")
		_ = sdk.Close()
		return
	}

	if err := sdk.DisablePlanMode(ctx); err != nil {
		log.Printf("disable plan mode: %v", err)
	}

	executePrompt := []string{
		"Implement the approved plan.",
		"Keep changes scoped.",
		"Run the relevant checks.",
		"Summarize changed files and verification results.",
	}

	fmt.Println("\n--- implementation ---\n")
	streamPrompt(ctx, sdk, executePrompt[0]+"\n"+executePrompt[1]+"\n"+executePrompt[2]+"\n"+executePrompt[3])

	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
}
