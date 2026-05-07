// 06-prompt-skills demonstrates skills referenced via "/skill <name>" in prompts.
//
// Usage:
//
//	go run ./examples/06-prompt-skills
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
		fmt.Printf("[Agent started: %s]\n", e.SessionID)
		fmt.Printf("  Model: %s\n", e.Model)
	case autohand.ToolStartEvent:
		fmt.Printf("\n[Tool called: %s]\n", e.ToolName)
	case autohand.ToolUpdateEvent:
		fmt.Print(e.Output)
	case autohand.ToolEndEvent:
		fmt.Printf("[Tool completed: %s]\n", e.ToolName)
	case autohand.PermissionRequestEvent:
		fmt.Printf("\n[Permission request: %s]\n", e.Tool)
		fmt.Printf("  Description: %s\n", e.Description)
	case autohand.MessageUpdateEvent:
		fmt.Print(e.Delta)
	case autohand.MessageEndEvent:
		if e.Content != "" {
			fmt.Println("\n[Message completed]")
		}
	case autohand.AgentEndEvent:
		fmt.Println("\n[Agent ended]")
	case autohand.ErrorEvent:
		fmt.Printf("\n[Error: %s]\n", e.Message)
	}
}

func main() {
	ctx := context.Background()

	sdk := autohand.NewSDK(&autohand.Config{
		CWD:   ".",
		Model: "openrouter/auto",
		Skills: []autohand.SkillRef{
			{Name: "typescript"},
			{Name: "testing"},
			{Name: "react"},
			{Name: "nodejs"},
		},
	})

	if err := sdk.Start(ctx); err != nil {
		log.Fatalf("start SDK: %v", err)
	}
	fmt.Println("SDK started")
	fmt.Println("Skills loaded: typescript, testing, react, nodejs")

	prompt := "Review this Go code using /skill typescript best practices and suggest improvements."
	fmt.Printf("Sending prompt: %q\n\n", prompt)
	fmt.Println("The agent can reference pre-loaded skills via /skill syntax")

	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{Message: prompt})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}

	for event := range events {
		handleEvent(event)
	}

	fmt.Println("\nSDK stopped")
	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
}
