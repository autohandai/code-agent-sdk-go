// 01-hello-agent demonstrates basic SDK usage with a simple prompt.
//
// Usage:
//
//	go run ./examples/01-hello-agent
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

	fmt.Println("Sending prompt...")
	err := sdk.Prompt(ctx, &autohand.PromptParams{
		Message: "Tell me a good joke about code AI agents!",
	})
	if err != nil {
		log.Fatalf("prompt: %v", err)
	}
	fmt.Println("Prompt sent")

	state, err := sdk.GetState(ctx)
	if err != nil {
		log.Fatalf("get state: %v", err)
	}
	fmt.Printf("State: %+v\n", state)

	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
	fmt.Println("SDK stopped")
}
