package autohand

import (
	"bufio"
	"encoding/json"
	"io"
	"testing"
	"time"
)

type capturedRPCRequest struct {
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func newRPCTestClient(t *testing.T) (*RPCClient, <-chan capturedRPCRequest, func()) {
	t.Helper()
	reader, writer := io.Pipe()
	transport := &Transport{
		stdin:     writer,
		callbacks: make(map[int]chan transportResponse),
		notify:    make(map[string]func(json.RawMessage)),
		nextID:    1,
		timeout:   2 * time.Second,
	}
	client := &RPCClient{transport: transport}
	client.setupNotifications()
	requests := make(chan capturedRPCRequest, 16)

	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			var request capturedRPCRequest
			if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
				return
			}
			requests <- request
			result := map[string]interface{}{"success": true}
			var responseResult interface{} = result
			switch request.Method {
			case "autohand.reset":
				responseResult = map[string]interface{}{"sessionId": "session-new"}
			case "autohand.browserHandoff.create":
				responseResult = map[string]interface{}{
					"token": "handoff-token", "sessionId": "session-1", "workspaceRoot": "/workspace",
					"createdAt": "2026-07-20T01:00:00Z", "expiresAt": "2026-07-20T01:05:00Z", "url": "chrome-extension://ext/continue",
				}
			case "autohand.browserHandoff.attach":
				responseResult = map[string]interface{}{"success": true, "sessionId": "session-1", "workspaceRoot": "/workspace", "messageCount": 7}
			case "autohand.browserHandoff.attachLatest":
				responseResult = map[string]interface{}{"success": false}
			case "autohand.automode.start":
				responseResult = map[string]interface{}{"success": true, "sessionId": "auto-1"}
			case "autohand.automode.status":
				responseResult = map[string]interface{}{
					"active": true, "paused": false,
					"state": map[string]interface{}{
						"sessionId": "auto-1", "status": "running", "currentIteration": 4, "maxIterations": 12,
						"filesCreated": 2, "filesModified": 5, "branch": "autohand/auto-1",
						"lastCheckpoint": map[string]interface{}{"commit": "abc123", "message": "checkpoint", "timestamp": "2026-07-20T01:02:00Z"},
					},
				}
			case "autohand.automode.pause":
				responseResult = map[string]interface{}{"success": false, "error": "No auto-mode session is running"}
			case "autohand.automode.resume", "autohand.automode.cancel":
				responseResult = map[string]interface{}{"success": true}
			case "autohand.automode.getLog":
				responseResult = map[string]interface{}{
					"success": true,
					"iterations": []interface{}{map[string]interface{}{
						"iteration": 4, "timestamp": "2026-07-20T01:03:00Z", "actions": []string{"edit", "test"},
						"tokensUsed": 1234, "cost": 0.42,
						"checkpoint": map[string]interface{}{"commit": "def456", "message": "iteration 4"},
					}},
				}
			case "autohand.autoresearch.start":
				result["instruction"] = "Run the next experiment"
			case "autohand.autoresearch.status":
				result["active"] = true
				result["statusText"] = "Auto-research active"
				result["runsLogged"] = 2
			case "autohand.autoresearch.history":
				result["attempts"] = []interface{}{}
			case "autohand.autoresearch.rescore":
				result["decisions"] = []interface{}{}
			case "autohand.autoresearch.pareto":
				result["attemptIds"] = []interface{}{}
			case "autohand.autoresearch.pin":
				result["attemptId"] = "attempt-1"
				result["pinned"] = true
			case "autohand.autoresearch.prune":
				result["applied"] = false
				result["candidates"] = []interface{}{}
				result["bytesFreed"] = 0
				result["remainingBytes"] = 0
			case "autohand.goal.listTemplates":
				responseResult = []interface{}{}
			case "autohand.getSkillsRegistry":
				responseResult = map[string]interface{}{
					"success": true,
					"skills": []interface{}{map[string]interface{}{
						"id": "skill-1", "name": "review", "description": "Review code", "category": "quality",
					}},
					"categories": []interface{}{map[string]interface{}{"name": "quality", "count": 1}},
				}
			case "autohand.installSkill":
				responseResult = map[string]interface{}{"success": true, "skillName": "review", "path": ".autohand/skills/review"}
			case "autohand.mcp.listServers":
				responseResult = map[string]interface{}{"servers": []interface{}{map[string]interface{}{"name": "github", "status": "connected", "toolCount": 2}}}
			case "autohand.mcp.listTools":
				responseResult = map[string]interface{}{"tools": []interface{}{map[string]interface{}{"name": "issues", "description": "List issues", "serverName": "github"}}}
			case "autohand.mcp.getServerConfigs":
				responseResult = map[string]interface{}{"configs": []interface{}{map[string]interface{}{"name": "github", "transport": "stdio", "command": "gh-mcp", "args": []string{"serve"}, "autoConnect": true}}}
			}
			response, _ := json.Marshal(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result":  responseResult,
			})
			transport.handleLine(string(response))
		}
	}()

	return client, requests, func() {
		_ = writer.Close()
		_ = reader.Close()
	}
}
