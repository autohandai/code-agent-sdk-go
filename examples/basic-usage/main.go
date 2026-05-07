// basic-usage demonstrates fundamental SDK usage.
//
// Usage:
//
//	go run ./examples/basic-usage
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

	fmt.Println("Sending prompt: \"Hello, Autohand!\"\n")
	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{Message: "Hello, Autohand!"})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.MessageUpdateEvent:
			fmt.Print(e.Delta)
		case autohand.MessageEndEvent:
			fmt.Println()
		}
	}

	fmt.Println("\nPrompt completed")

	state, err := sdk.GetState(ctx)
	if err != nil {
		log.Fatalf("get state: %v", err)
	}
	fmt.Printf("Current state: %+v\n", state)

	messages, err := sdk.GetMessages(ctx, 0)
	if err != nil {
		log.Fatalf("get messages: %v", err)
	}
	fmt.Printf("Messages: %d\n", len(messages.Messages))

	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
	fmt.Println("SDK stopped")
}
