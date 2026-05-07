// Package autohand provides a Go SDK for interacting with the Autohand CLI
// through JSON-RPC 2.0 over stdio.
//
// It supports streaming events, permission management, model switching,
// and full lifecycle control of agent sessions.
//
// # Quick Start
//
//	import autohand "github.com/autohandai/code-agent-sdk-go"
//
//	func main() {
//	    sdk := autohand.NewSDK(&autohand.Config{
//	        CWD:   ".",
//	        Debug: true,
//	    })
//
//	    ctx := context.Background()
//	    if err := sdk.Start(ctx); err != nil {
//	        log.Fatal(err)
//	    }
//	    defer sdk.Close()
//
//	    events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{
//	        Message: "Hello, agent!",
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    for event := range events {
//	        switch e := event.(type) {
//	        case autohand.MessageUpdateEvent:
//	            fmt.Print(e.Delta)
//	        case autohand.AgentEndEvent:
//	            fmt.Println("\nDone!")
//	        }
//	    }
//	}
//
// # High-Level Agent API
//
// For simpler use cases, the Agent type manages the SDK lifecycle:
//
//	agent, err := autohand.NewAgent(ctx, &autohand.Config{
//	    CWD: ".",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer agent.Close()
//
//	result, err := agent.Run(ctx, "Explain this codebase", nil)
//
// # Event Types
//
// The SDK emits typed events during streaming. Use a type switch to handle them:
//
//	- AgentStartEvent
//	- AgentEndEvent
//	- TurnStartEvent
//	- TurnEndEvent
//	- MessageStartEvent
//	- MessageUpdateEvent
//	- MessageEndEvent
//	- ToolStartEvent
//	- ToolUpdateEvent
//	- ToolEndEvent
//	- FileModifiedEvent
//	- PermissionRequestEvent
//	- ErrorEvent
//
// See the examples/ directory for complete, runnable programs.
package autohand
