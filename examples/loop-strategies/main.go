// loop-strategies demonstrates different execution modes for the agent.
//
// Note: Loop strategies are configured on the CLI side. The SDK passes
// configuration to the CLI which handles the execution strategy.
//
// Usage:
//
//	go run ./examples/loop-strategies
package main

import (
	"context"
	"fmt"
	"log"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Loop Strategies Demo ===")
	fmt.Println("Note: Loop strategies are configured on the CLI side.")
	fmt.Println("The SDK passes configuration to the CLI.\n")

	sdk := autohand.NewSDK(&autohand.Config{})

	if err := sdk.Start(ctx); err != nil {
		log.Fatalf("start SDK: %v", err)
	}
	fmt.Println("SDK started")

	prompt := "List all Go files in the current directory and read each one. Summarize the codebase."
	fmt.Printf("Sending prompt: %q\n\n", prompt)

	var fullResponse string
	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{Message: prompt})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.ToolStartEvent:
			fmt.Printf("[Tool called: %s]\n", e.ToolName)
		case autohand.ToolEndEvent:
			fmt.Printf("[Tool completed: %s]\n", e.ToolName)
		case autohand.PermissionRequestEvent:
			fmt.Printf("[Permission request: %s]\n", e.Tool)
			fmt.Printf("  Description: %s\n", e.Description)
			if err := sdk.AllowPermission(ctx, e.RequestID, autohand.ScopeOnce); err != nil {
				log.Printf("allow: %v", err)
			}
		case autohand.MessageUpdateEvent:
			fmt.Print(e.Delta)
			fullResponse += e.Delta
		case autohand.MessageEndEvent:
			fullResponse = e.Content
		}
	}

	fmt.Println("\n=== Agent Response ===")
	fmt.Println(fullResponse)

	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
	fmt.Println("\nSDK stopped")

	fmt.Println("\n=== To use different loop strategies ===")
	fmt.Println("Configure the CLI with appropriate flags or config:")
	fmt.Println("  - ReAct (default): Standard reasoning loop")
	fmt.Println("  - Plan-and-Execute: Plan first, then execute")
	fmt.Println("  - Parallel: Execute tools in parallel")
	fmt.Println("  - Reflexion: Self-reflective execution")
}
