// 07-direct-skills demonstrates skills provided directly via SDK with file paths.
//
// The SDK auto-detects file paths and copies them to ~/.autohand/skills/.
//
// Usage:
//
//	go run ./examples/07-direct-skills
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

	// Mix of built-in skill names and file path references
	skills := []autohand.SkillRef{
		{Name: "typescript"},
		{Name: "testing"},
		// Uncomment if you have custom skill files:
		// {Name: "custom-api", Path: "./skills/my-custom/SKILL.md"},
	}

	sdk := autohand.NewSDK(&autohand.Config{
		CWD:    ".",
		Model:  "openrouter/auto",
		Skills: skills,
	})

	if err := sdk.Start(ctx); err != nil {
		log.Fatalf("start SDK: %v", err)
	}
	fmt.Println("SDK started")
	fmt.Printf("Skills loaded: %d\n", len(skills))

	prompt := "Review this codebase and suggest improvements."
	fmt.Printf("Sending prompt: %q\n\n", prompt)
	fmt.Println("(Skills are pre-loaded and available to the agent)")

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
