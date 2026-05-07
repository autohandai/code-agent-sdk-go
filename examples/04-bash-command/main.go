// 04-bash-command demonstrates an agent that runs shell commands.
//
// Usage:
//
//	go run ./examples/04-bash-command
package main

import (
	"context"
	"fmt"
	"log"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	sdk := autohand.NewSDK(&autohand.Config{})

	if err := sdk.Start(ctx); err != nil {
		log.Fatalf("start SDK: %v", err)
	}
	fmt.Println("SDK started")

	prompt := "What is the current directory listing and total file count?"
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
			if e.Output != "" {
				truncated := e.Output
				if len(truncated) > 1000 {
					truncated = truncated[:1000] + "..."
				}
				fmt.Println("Output:")
				fmt.Println(truncated)
			}
		case autohand.PermissionRequestEvent:
			fmt.Printf("[Permission request: %s]\n", e.Tool)
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
}
