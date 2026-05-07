package autohand

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// RPCClient is the JSON-RPC client for CLI communication.
type RPCClient struct {
	transport *Transport

	mu          sync.Mutex
	eventQueue  []Event
	eventWaiters []chan Event
}

// NewRPCClient creates a new RPC client.
func NewRPCClient(cfg *Config) *RPCClient {
	c := &RPCClient{
		transport: NewTransport(cfg),
	}
	c.setupNotifications()
	return c
}

// Start starts the transport.
func (c *RPCClient) Start(ctx context.Context, cfg *Config) error {
	return c.transport.Start(ctx, cfg)
}

// Stop stops the transport.
func (c *RPCClient) Stop() error {
	return c.transport.Stop()
}

// Prompt sends a prompt to the agent.
func (c *RPCClient) Prompt(ctx context.Context, params *PromptParams) error {
	_, err := c.transport.Request(ctx, "autohand.prompt", params)
	return err
}

// Abort aborts the current operation.
func (c *RPCClient) Abort(ctx context.Context) error {
	_, err := c.transport.Request(ctx, "autohand.abort", map[string]interface{}{})
	return err
}

// GetState returns the current agent state.
func (c *RPCClient) GetState(ctx context.Context) (*GetStateResult, error) {
	resp, err := c.transport.Request(ctx, "autohand.getState", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var result GetStateResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}
	return &result, nil
}

// GetMessages returns conversation messages.
func (c *RPCClient) GetMessages(ctx context.Context, limit int) (*GetMessagesResult, error) {
	params := map[string]interface{}{}
	if limit > 0 {
		params["limit"] = limit
	}
	resp, err := c.transport.Request(ctx, "autohand.getMessages", params)
	if err != nil {
		return nil, err
	}
	var result GetMessagesResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal messages: %w", err)
	}
	return &result, nil
}

// PermissionResponse responds to a permission request.
func (c *RPCClient) PermissionResponse(ctx context.Context, requestID string, decision PermissionDecision) error {
	params := map[string]interface{}{
		"requestId": requestID,
		"decision":  string(decision),
	}
	_, err := c.transport.Request(ctx, "autohand.permissionResponse", params)
	return err
}

// SetPermissionMode sets the permission mode.
func (c *RPCClient) SetPermissionMode(ctx context.Context, mode PermissionMode) error {
	params := map[string]interface{}{"mode": string(mode)}
	_, err := c.transport.Request(ctx, "autohand.permissionModeSet", params)
	return err
}

// SetPlanMode enables or disables plan mode.
func (c *RPCClient) SetPlanMode(ctx context.Context, enabled bool) error {
	params := map[string]interface{}{"enabled": enabled}
	_, err := c.transport.Request(ctx, "autohand.planModeSet", params)
	return err
}

// SetModel changes the model.
func (c *RPCClient) SetModel(ctx context.Context, model string) error {
	params := map[string]interface{}{"model": model}
	_, err := c.transport.Request(ctx, "autohand.modelSet", params)
	return err
}

// SetMaxThinkingTokens sets the max thinking tokens.
func (c *RPCClient) SetMaxThinkingTokens(ctx context.Context, tokens int) error {
	params := map[string]interface{}{"maxThinkingTokens": tokens}
	_, err := c.transport.Request(ctx, "autohand.maxThinkingTokensSet", params)
	return err
}

// ApplyFlagSettings applies flag settings at runtime.
func (c *RPCClient) ApplyFlagSettings(ctx context.Context, settings map[string]interface{}) error {
	params := map[string]interface{}{"settings": settings}
	_, err := c.transport.Request(ctx, "autohand.applyFlagSettings", params)
	return err
}

// GetSupportedModels returns available models.
func (c *RPCClient) GetSupportedModels(ctx context.Context) ([]ModelInfo, error) {
	resp, err := c.transport.Request(ctx, "autohand.getSupportedModels", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Models []ModelInfo `json:"models"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal models: %w", err)
	}
	return result.Models, nil
}

// GetContextUsage returns context window usage.
func (c *RPCClient) GetContextUsage(ctx context.Context) (*ContextUsage, error) {
	resp, err := c.transport.Request(ctx, "autohand.getContextUsage", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var result ContextUsage
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal context usage: %w", err)
	}
	return &result, nil
}

// GetAccountInfo returns account information.
func (c *RPCClient) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	resp, err := c.transport.Request(ctx, "autohand.getAccountInfo", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var result AccountInfo
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal account info: %w", err)
	}
	return &result, nil
}

// ToggleMCPServer enables or disables an MCP server.
func (c *RPCClient) ToggleMCPServer(ctx context.Context, serverName string, enabled bool) error {
	params := map[string]interface{}{
		"serverName": serverName,
		"enabled":    enabled,
	}
	_, err := c.transport.Request(ctx, "autohand.mcp.toggleServer", params)
	return err
}

// ReconnectMCPServer reconnects an MCP server.
func (c *RPCClient) ReconnectMCPServer(ctx context.Context, serverName string) error {
	params := map[string]interface{}{"serverName": serverName}
	_, err := c.transport.Request(ctx, "autohand.mcp.reconnectServer", params)
	return err
}

// SetMCPServers sets MCP server configurations.
func (c *RPCClient) SetMCPServers(ctx context.Context, servers map[string]MCPServerConfig) error {
	params := map[string]interface{}{"servers": servers}
	_, err := c.transport.Request(ctx, "autohand.mcp.setServers", params)
	return err
}

// GetHooks returns all hooks.
func (c *RPCClient) GetHooks(ctx context.Context) (*HooksSettings, error) {
	resp, err := c.transport.Request(ctx, "autohand.hooks.getHooks", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Settings HooksSettings `json:"settings"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal hooks: %w", err)
	}
	return &result.Settings, nil
}

// AddHook adds a hook.
func (c *RPCClient) AddHook(ctx context.Context, hook HookDefinition) error {
	params := map[string]interface{}{"hook": hook}
	_, err := c.transport.Request(ctx, "autohand.hooks.addHook", params)
	return err
}

// RemoveHook removes a hook.
func (c *RPCClient) RemoveHook(ctx context.Context, event HookEvent, index int) error {
	params := map[string]interface{}{
		"event": string(event),
		"index": index,
	}
	_, err := c.transport.Request(ctx, "autohand.hooks.removeHook", params)
	return err
}

// ToggleHook toggles a hook.
func (c *RPCClient) ToggleHook(ctx context.Context, event HookEvent, index int) error {
	params := map[string]interface{}{
		"event": string(event),
		"index": index,
	}
	_, err := c.transport.Request(ctx, "autohand.hooks.toggleHook", params)
	return err
}

// TestHook tests a hook.
func (c *RPCClient) TestHook(ctx context.Context, hook HookDefinition) error {
	params := map[string]interface{}{"hook": hook}
	_, err := c.transport.Request(ctx, "autohand.hooks.testHook", params)
	return err
}

// ReloadPlugins reloads CLI plugins.
func (c *RPCClient) ReloadPlugins(ctx context.Context) error {
	_, err := c.transport.Request(ctx, "autohand.reloadPlugins", map[string]interface{}{})
	return err
}

// SaveSession saves the current session to disk.
func (c *RPCClient) SaveSession(ctx context.Context) error {
	_, err := c.transport.Request(ctx, "autohand.saveSession", map[string]interface{}{})
	return err
}

// GetSupportedCommands returns available slash commands.
func (c *RPCClient) GetSupportedCommands(ctx context.Context) ([]string, error) {
	resp, err := c.transport.Request(ctx, "autohand.getSupportedCommands", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Commands []string `json:"commands"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal commands: %w", err)
	}
	return result.Commands, nil
}

// GetSessionMetadata returns session metadata.
func (c *RPCClient) GetSessionMetadata(ctx context.Context) (*SessionMetadata, error) {
	resp, err := c.transport.Request(ctx, "autohand.getSessionMetadata", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var result SessionMetadata
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal session metadata: %w", err)
	}
	return &result, nil
}

// GetStats returns session statistics.
func (c *RPCClient) GetStats(ctx context.Context) (*SessionStats, error) {
	resp, err := c.transport.Request(ctx, "autohand.getStats", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var result SessionStats
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal stats: %w", err)
	}
	return &result, nil
}

// RewindFiles rewinds file modifications to a checkpoint.
func (c *RPCClient) RewindFiles(ctx context.Context, checkpointID string) error {
	params := map[string]interface{}{"checkpointId": checkpointID}
	_, err := c.transport.Request(ctx, "autohand.rewindFiles", params)
	return err
}

// SeedReadState seeds read state for deterministic runs.
func (c *RPCClient) SeedReadState(ctx context.Context, state map[string]interface{}) error {
	_, err := c.transport.Request(ctx, "autohand.seedReadState", state)
	return err
}

// StreamInput sends streaming input to the agent.
func (c *RPCClient) StreamInput(ctx context.Context, input string) error {
	params := map[string]interface{}{"input": input}
	_, err := c.transport.Request(ctx, "autohand.streamInput", params)
	return err
}

// SetSystemPrompt sets the system prompt.
func (c *RPCClient) SetSystemPrompt(ctx context.Context, prompt string) error {
	params := map[string]interface{}{"prompt": prompt}
	_, err := c.transport.Request(ctx, "autohand.setSystemPrompt", params)
	return err
}

// AppendSystemPrompt appends to the system prompt.
func (c *RPCClient) AppendSystemPrompt(ctx context.Context, prompt string) error {
	params := map[string]interface{}{"prompt": prompt}
	_, err := c.transport.Request(ctx, "autohand.appendSystemPrompt", params)
	return err
}

// Request sends a custom RPC request.
func (c *RPCClient) Request(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	return c.transport.Request(ctx, method, params)
}

// Events returns a channel of SDK events.
func (c *RPCClient) Events(ctx context.Context) <-chan Event {
	ch := make(chan Event, 256)
	go func() {
		defer close(ch)
		for {
			event, err := c.nextEvent(ctx)
			if err != nil {
				return
			}
			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

func (c *RPCClient) nextEvent(ctx context.Context) (Event, error) {
	c.mu.Lock()
	if len(c.eventQueue) > 0 {
		event := c.eventQueue[0]
		c.eventQueue = c.eventQueue[1:]
		c.mu.Unlock()
		return event, nil
	}

	waiter := make(chan Event, 1)
	c.eventWaiters = append(c.eventWaiters, waiter)
	c.mu.Unlock()

	select {
	case event := <-waiter:
		return event, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *RPCClient) queueEvent(event Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.eventWaiters) > 0 {
		waiter := c.eventWaiters[0]
		c.eventWaiters = c.eventWaiters[1:]
		waiter <- event
	} else {
		c.eventQueue = append(c.eventQueue, event)
	}
}

// IsConnected returns whether the transport is running.
func (c *RPCClient) IsConnected() bool {
	return c.transport.IsRunning()
}

func (c *RPCClient) setupNotifications() {
	c.transport.OnNotification("autohand.agentStart", func(params json.RawMessage) {
		var e AgentStartEvent
		json.Unmarshal(params, &e)
		e.Type = "agent_start"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.agentEnd", func(params json.RawMessage) {
		var e AgentEndEvent
		json.Unmarshal(params, &e)
		e.Type = "agent_end"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.turnStart", func(params json.RawMessage) {
		var e TurnStartEvent
		json.Unmarshal(params, &e)
		e.Type = "turn_start"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.turnEnd", func(params json.RawMessage) {
		var e TurnEndEvent
		json.Unmarshal(params, &e)
		e.Type = "turn_end"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.messageStart", func(params json.RawMessage) {
		var e MessageStartEvent
		json.Unmarshal(params, &e)
		e.Type = "message_start"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.messageUpdate", func(params json.RawMessage) {
		var e MessageUpdateEvent
		json.Unmarshal(params, &e)
		e.Type = "message_update"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.messageEnd", func(params json.RawMessage) {
		var e MessageEndEvent
		json.Unmarshal(params, &e)
		e.Type = "message_end"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.toolStart", func(params json.RawMessage) {
		var e ToolStartEvent
		json.Unmarshal(params, &e)
		e.Type = "tool_start"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.toolUpdate", func(params json.RawMessage) {
		var e ToolUpdateEvent
		json.Unmarshal(params, &e)
		e.Type = "tool_update"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.toolEnd", func(params json.RawMessage) {
		var e ToolEndEvent
		json.Unmarshal(params, &e)
		e.Type = "tool_end"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.hook.fileModified", func(params json.RawMessage) {
		var e FileModifiedEvent
		json.Unmarshal(params, &e)
		e.Type = "file_modified"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.permissionRequest", func(params json.RawMessage) {
		var e PermissionRequestEvent
		json.Unmarshal(params, &e)
		e.Type = "permission_request"
		c.queueEvent(e)
	})

	c.transport.OnNotification("autohand.error", func(params json.RawMessage) {
		var e ErrorEvent
		json.Unmarshal(params, &e)
		e.Type = "error"
		c.queueEvent(e)
	})
}
