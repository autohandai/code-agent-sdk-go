// 13-permissions demonstrates different permission modes.
//
// Usage:
//
//	go run ./examples/13-permissions
package main

import (
	"context"
	"fmt"
	"log"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Permission Modes Demo ===\n")

	modes := []autohand.PermissionMode{
		autohand.PermissionInteractive,
		autohand.PermissionRestricted,
		autohand.PermissionUnrestricted,
	}

	for _, mode := range modes {
		fmt.Printf("\n--- Testing %s mode ---\n", mode)

		sdk := autohand.NewSDK(&autohand.Config{})

		if err := sdk.Start(ctx); err != nil {
			log.Fatalf("start SDK: %v", err)
		}
		if err := sdk.SetPermissionMode(ctx, mode); err != nil {
			log.Printf("set permission mode: %v", err)
		}
		fmt.Printf("SDK started with permission mode: %s\n", mode)

		prompt := "List the files in the current directory"
		fmt.Printf("\nSending prompt: %q\n", prompt)

		events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{Message: prompt})
		if err != nil {
			log.Printf("stream prompt: %v", err)
			_ = sdk.Close()
			continue
		}

		for event := range events {
			switch e := event.(type) {
			case autohand.ToolStartEvent:
				fmt.Printf("\n[Tool called: %s]\n", e.ToolName)
			case autohand.ToolEndEvent:
				fmt.Printf("\n[Tool completed: %s]\n", e.ToolName)
				if e.Output != "" {
					truncated := e.Output
					if len(truncated) > 500 {
						truncated = truncated[:500] + "..."
					}
					fmt.Println("  Output:", truncated)
				}
			case autohand.PermissionRequestEvent:
				fmt.Printf("\n[Permission request: %s]\n", e.Tool)
				fmt.Printf("  Description: %s\n", e.Description)
				fmt.Printf("  Request ID: %s\n", e.RequestID)
				if mode == autohand.PermissionInteractive {
					fmt.Println("  Auto-approving for demo...")
					if err := sdk.AllowPermission(ctx, e.RequestID, autohand.ScopeOnce); err != nil {
						log.Printf("allow: %v", err)
					}
				}
			case autohand.MessageUpdateEvent:
				fmt.Print(e.Delta)
			case autohand.MessageEndEvent:
				fmt.Println("\nPrompt completed")
			}
		}

		if err := sdk.Close(); err != nil {
			log.Fatalf("close SDK: %v", err)
		}
		fmt.Println("SDK stopped")
	}

	fmt.Println("\n=== Demo Complete ===")
}
