// 03-high-level-agent demonstrates the high-level Agent API.
//
// Usage:
//
//	go run ./examples/03-high-level-agent
package main

import (
	"context"
	"fmt"
	"log"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	agent, err := autohand.NewAgent(ctx, &autohand.Config{
		CWD:   ".",
		Model: "openrouter/auto",
	})
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}
	defer agent.Close()

	run, err := agent.Send(ctx, "Review this repository for release readiness", nil)
	if err != nil {
		log.Fatalf("send: %v", err)
	}

	events, err := run.Stream(ctx)
	if err != nil {
		log.Fatalf("stream: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.MessageUpdateEvent:
			fmt.Print(e.Delta)
		case autohand.PermissionRequestEvent:
			fmt.Printf("\n[permission] %s: %s\n", e.Tool, e.Description)
			if err := agent.DenyPermission(ctx, e.RequestID, autohand.ScopeOnce); err != nil {
				log.Printf("deny permission: %v", err)
			}
		case autohand.ToolStartEvent:
			fmt.Printf("\n[tool] %s\n", e.ToolName)
		}
	}

	result, err := run.Wait(ctx)
	if err != nil {
		log.Fatalf("wait: %v", err)
	}

	fmt.Printf("\n\nRun %s %s with %d events.\n", result.ID, result.Status, len(result.Events))
	fmt.Println("Final text:")
	fmt.Println(result.Text)
}
