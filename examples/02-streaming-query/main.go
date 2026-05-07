// 02-streaming-query demonstrates real-time event streaming from agent execution.
//
// Usage:
//
//	go run ./examples/02-streaming-query
package main

import (
	"context"
	"fmt"
	"log"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func handleEvent(event autohand.Event) {
	switch e := event.(type) {
	case autohand.AgentStartEvent:
		fmt.Printf("\n[Agent started: %s]\n", e.SessionID)
	case autohand.TurnStartEvent:
		fmt.Printf("\n[Turn started: %s]\n", e.TurnID)
	case autohand.MessageUpdateEvent:
		fmt.Print(e.Delta)
	case autohand.MessageEndEvent:
		fmt.Println("\n[Message completed]")
	case autohand.ToolStartEvent:
		fmt.Printf("\n[Tool called: %s]\n", e.ToolName)
	case autohand.ToolUpdateEvent:
		fmt.Print(e.Output)
	case autohand.ToolEndEvent:
		fmt.Printf("\n[Tool completed: %s]\n", e.ToolName)
		if e.Output != "" {
			truncated := e.Output
			if len(truncated) > 500 {
				truncated = truncated[:500] + "..."
			}
			fmt.Printf("  Output: %s\n", truncated)
		}
	case autohand.PermissionRequestEvent:
		fmt.Printf("\n[Permission request: %s]\n", e.Tool)
		fmt.Printf("  Description: %s\n", e.Description)
	case autohand.TurnEndEvent:
		fmt.Println("\n[Turn ended]")
	case autohand.AgentEndEvent:
		fmt.Println("\n[Agent ended]")
	case autohand.ErrorEvent:
		fmt.Printf("\n[Error: %s]\n", e.Message)
	}
}

func main() {
	ctx := context.Background()

	sdk := autohand.NewSDK(&autohand.Config{})

	if err := sdk.Start(ctx); err != nil {
		log.Fatalf("start SDK: %v", err)
	}
	fmt.Println("SDK started")

	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
		Message: "Explain closures in one sentence",
	})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}

	for event := range events {
		handleEvent(event)
	}

	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
	fmt.Println("\nSDK stopped")
}
