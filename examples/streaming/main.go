// streaming demonstrates handling all event types from a prompt stream.
//
// Usage:
//
//	go run ./examples/streaming
package main

import (
	"context"
	"fmt"
	"log"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	sdk := autohand.NewSDK(&autohand.Config{
		CWD:   ".",
		Debug: true,
	})

	if err := sdk.Start(ctx); err != nil {
		log.Fatalf("start SDK: %v", err)
	}
	fmt.Println("SDK started")

	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
		Message: "Analyze the current directory structure",
	})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.AgentStartEvent:
			fmt.Printf("[Agent] Started: %s\n", e.SessionID)
		case autohand.TurnStartEvent:
			fmt.Printf("[Turn] Started: %s\n", e.TurnID)
		case autohand.MessageStartEvent:
			fmt.Printf("[Message] Started: %s\n", e.MessageID)
		case autohand.MessageUpdateEvent:
			fmt.Print(e.Delta)
		case autohand.MessageEndEvent:
			fmt.Println("\n[Message] Completed")
		case autohand.ToolStartEvent:
			fmt.Printf("[Tool] %s started\n", e.ToolName)
		case autohand.ToolEndEvent:
			status := "success"
			if !e.Success {
				status = "failed"
			}
			fmt.Printf("[Tool] %s completed: %s\n", e.ToolName, status)
			if e.Output != "" {
				truncated := e.Output
				if len(truncated) > 500 {
					truncated = truncated[:500] + "..."
				}
				fmt.Println("  Output:", truncated)
			}
		case autohand.AgentEndEvent:
			fmt.Printf("[Agent] Ended: %s\n", e.Reason)
		case autohand.ErrorEvent:
			fmt.Printf("[Error] %s\n", e.Message)
		}
	}

	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
	fmt.Println("SDK stopped")
}
