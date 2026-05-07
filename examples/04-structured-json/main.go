// 04-structured-json demonstrates JSON-mode output with the high-level Agent API.
//
// Usage:
//
//	go run ./examples/04-structured-json
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

type ReleaseRisk struct {
	Summary string `json:"summary"`
	Risks   []Risk `json:"risks"`
}

type Risk struct {
	Title      string `json:"title"`
	Severity   string `json:"severity"`
	Mitigation string `json:"mitigation"`
}

func validateReleaseRisk(value interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result ReleaseRisk
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid shape: %w", err)
	}
	for _, r := range result.Risks {
		if r.Severity != "low" && r.Severity != "medium" && r.Severity != "high" {
			return nil, fmt.Errorf("invalid severity: %s", r.Severity)
		}
	}
	return data, nil
}

func main() {
	ctx := context.Background()

	agent, err := autohand.NewAgent(ctx, &autohand.Config{
		CWD:          ".",
		Model:        "openrouter/auto",
		Instructions: "Prefer concise, factual release-readiness analysis.",
	})
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}
	defer agent.Close()

	message := []string{
		"Assess this SDK repository for publish readiness. Do not execute commands.",
		"",
		"Return only valid JSON. Do not wrap the response in Markdown.",
		"The JSON value should satisfy: ReleaseRisk.",
		"Use this JSON schema or example shape:",
		`{`,
		`  "summary": "string",`,
		`  "risks": [`,
		`    {`,
		`      "title": "string",`,
		`      "severity": "low | medium | high",`,
		`      "mitigation": "string"`,
		`    }`,
		`  ]`,
		`}`,
		"If you cannot inspect the repository, still return a JSON object.",
	}

	run, err := agent.Send(ctx, message, nil)
	if err != nil {
		log.Fatalf("send: %v", err)
	}

	events, err := run.Stream(ctx)
	if err != nil {
		log.Fatalf("stream: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.AgentStartEvent:
			fmt.Printf("[agent] %s using %s\n", e.SessionID, e.Model)
		case autohand.MessageUpdateEvent:
			fmt.Print(e.Delta)
		case autohand.ToolStartEvent:
			fmt.Printf("\n[tool] %s\n", e.ToolName)
		case autohand.PermissionRequestEvent:
			fmt.Printf("\n[permission] %s: %s\n", e.Tool, e.Description)
			if err := agent.DenyPermission(ctx, e.RequestID, autohand.ScopeOnce); err != nil {
				log.Printf("deny permission: %v", err)
			}
		}
	}

	result, err := run.JSON(autohand.JsonParseOptions[json.RawMessage]{
		Validate: validateReleaseRisk,
	})
	if err != nil {
		if serr, ok := err.(*autohand.StructuredOutputError); ok {
			log.Fatalf("structured output error: %s\nRaw text:\n%s", serr.Message, serr.RawText)
		}
		log.Fatalf("JSON error: %v", err)
	}

	fmt.Println("\n\nParsed JSON:")
	var pretty map[string]interface{}
	json.Unmarshal(result, &pretty)
	prettyJSON, _ := json.MarshalIndent(pretty, "", "  ")
	fmt.Println(string(prettyJSON))
}
