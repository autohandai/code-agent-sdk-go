// 22-sdlc-release-readiness demonstrates release readiness review.
//
// The agent runs production gates and reports release risk.
//
// Usage:
//
//	go run ./examples/22-sdlc-release-readiness
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	cliPath := os.Getenv("AUTOHAND_CLI_PATH")
	cfg := &autohand.Config{
		CWD:   ".",
		Model: os.Getenv("AUTOHAND_MODEL"),
	}
	if cliPath != "" {
		cfg.CLIPath = cliPath
	}

	sdk := autohand.NewSDK(cfg)

	if err := sdk.Start(ctx); err != nil {
		log.Fatalf("start SDK: %v", err)
	}

	message := []string{
		"Run a release-readiness pass for this Go SDK.",
		"Use the repository standard commands: go vet, go test, go build.",
		"If a command fails, stop and explain the failure with file references.",
		"If all commands pass, summarize residual risks and production readiness.",
	}

	var toolResults []struct {
		Name    string
		Success bool
	}

	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
		Message: message[0] + "\n" + message[1] + "\n" + message[2] + "\n" + message[3],
	})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.MessageUpdateEvent:
			fmt.Print(e.Delta)
		case autohand.ToolStartEvent:
			fmt.Printf("\n[tool:start] %s\n", e.ToolName)
		case autohand.ToolUpdateEvent:
			fmt.Print(e.Output)
		case autohand.ToolEndEvent:
			toolResults = append(toolResults, struct {
				Name    string
				Success bool
			}{e.ToolName, e.Success})
			fmt.Printf("[tool:end] %s success=%v\n", e.ToolName, e.Success)
		case autohand.PermissionRequestEvent:
			fmt.Printf("\n[permission] %s: %s\n", e.Tool, e.Description)
		case autohand.ErrorEvent:
			fmt.Printf("\n[error] %s\n", e.Message)
		}
	}

	if len(toolResults) > 0 {
		fmt.Println("\n--- tool summary ---")
		for _, r := range toolResults {
			status := "pass"
			if !r.Success {
				status = "fail"
			}
			fmt.Printf("%s: %s\n", r.Name, status)
		}
	}

	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
}
