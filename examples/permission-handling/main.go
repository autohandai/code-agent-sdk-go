// permission-handling demonstrates responding to permission requests.
//
// Usage:
//
//	go run ./examples/permission-handling
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
		Debug: false,
	})

	if err := sdk.Start(ctx); err != nil {
		log.Fatalf("start SDK: %v", err)
	}
	fmt.Println("SDK started")

	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
		Message: "Create a new file called test.txt with some content",
	})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.PermissionRequestEvent:
			fmt.Printf("[Permission Request] %s\n", e.Description)
			fmt.Printf("Tool: %s\n", e.Tool)
			// Auto-allow for this example. Production hosts should route this to a UI.
			if err := sdk.AllowPermission(ctx, e.RequestID, autohand.ScopeOnce); err != nil {
				log.Printf("allow: %v", err)
			}
		case autohand.MessageUpdateEvent:
			fmt.Print(e.Delta)
		case autohand.AgentEndEvent:
			fmt.Println("\nAgent ended")
			_ = sdk.Close()
			return
		}
	}

	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
	fmt.Println("SDK stopped")
}
