// 08-memory-management demonstrates agent memory persistence across sessions.
//
// The Autohand CLI provides built-in memory tools (save_memory / recall_memory)
// that agents can use to persist and retrieve facts across sessions.
//
// Usage:
//
//	go run ./examples/08-memory-management
package main

import (
	"context"
	"fmt"
	"log"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func streamPromptWithLogging(ctx context.Context, sdk *autohand.SDK, prompt string) string {
	fmt.Printf("\n> %s\n\n", prompt)

	var fullResponse string
	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{Message: prompt})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.ToolStartEvent:
			fmt.Printf("[Tool: %s]\n", e.ToolName)
		case autohand.ToolUpdateEvent:
			fmt.Print(e.Output)
		case autohand.ToolEndEvent:
			fmt.Printf("\n[Tool completed: %s]\n", e.ToolName)
			if e.Output != "" {
				preview := e.Output
				if len(preview) > 500 {
					preview = preview[:500] + "..."
				}
				fmt.Println(preview)
			}
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
			if e.Content != "" {
				fullResponse = e.Content
			}
		}
	}

	fmt.Println()
	return fullResponse
}

func main() {
	ctx := context.Background()

	fmt.Println("=== Autohand SDK Memory Management Example ===\n")

	// Phase 1: Save a preference to memory
	saveSdk := autohand.NewSDK(&autohand.Config{})
	if err := saveSdk.Start(ctx); err != nil {
		log.Fatalf("start save SDK: %v", err)
	}
	fmt.Println("SDK started (save session)")

	savePrompt := "Save this to memory: \"The user prefers Go over JavaScript and likes functional programming patterns.\""
	streamPromptWithLogging(ctx, saveSdk, savePrompt)

	usageBefore, err := saveSdk.GetContextUsage(ctx)
	if err != nil {
		log.Printf("get context usage: %v", err)
	} else {
		fmt.Printf("Context usage before stop: %+v\n", usageBefore)
	}

	if err := saveSdk.Close(); err != nil {
		log.Fatalf("close save SDK: %v", err)
	}
	fmt.Println("SDK stopped (save session)")

	// Phase 2: Start a fresh session and recall the memory
	recallSdk := autohand.NewSDK(&autohand.Config{})
	if err := recallSdk.Start(ctx); err != nil {
		log.Fatalf("start recall SDK: %v", err)
	}
	fmt.Println("\nSDK started (recall session)")

	recallPrompt := "Recall what you know about my programming preferences from memory. What language do I prefer and what style do I like?"
	streamPromptWithLogging(ctx, recallSdk, recallPrompt)

	usageAfter, err := recallSdk.GetContextUsage(ctx)
	if err != nil {
		log.Printf("get context usage: %v", err)
	} else {
		fmt.Printf("Context usage after recall: %+v\n", usageAfter)
	}

	if err := recallSdk.Close(); err != nil {
		log.Fatalf("close recall SDK: %v", err)
	}
	fmt.Println("SDK stopped (recall session)")
}
