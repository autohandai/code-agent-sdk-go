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

	mu               sync.Mutex
	subscribers      map[uint64]eventSubscription
	nextSubscriberID uint64
}

type eventSubscription struct {
	events chan Event
	done   chan struct{}
}

// NewRPCClient creates a new RPC client.
func NewRPCClient(cfg *Config) *RPCClient {
	c := &RPCClient{
		transport:   NewTransport(cfg),
		subscribers: make(map[uint64]eventSubscription),
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
	err := c.transport.Stop()
	c.closeSubscribers()
	return err
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

// Reset replaces the active conversation and returns the new session ID.
func (c *RPCClient) Reset(ctx context.Context) (*ResetResult, error) {
	return rpcRequest[ResetResult](ctx, c, "autohand.reset", map[string]interface{}{})
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

// CreateBrowserHandoff creates a browser continuation token for the active session.
func (c *RPCClient) CreateBrowserHandoff(ctx context.Context, params *BrowserHandoffCreateParams) (*BrowserHandoffCreateResult, error) {
	if params == nil {
		params = &BrowserHandoffCreateParams{}
	}
	return rpcRequest[BrowserHandoffCreateResult](ctx, c, "autohand.browserHandoff.create", params)
}

// AttachBrowserHandoff restores the session referenced by a handoff token.
func (c *RPCClient) AttachBrowserHandoff(ctx context.Context, params *BrowserHandoffAttachParams) (*BrowserHandoffAttachResult, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	return rpcRequest[BrowserHandoffAttachResult](ctx, c, "autohand.browserHandoff.attach", params)
}

// AttachLatestBrowserHandoff restores the newest unexpired browser handoff.
func (c *RPCClient) AttachLatestBrowserHandoff(ctx context.Context) (*BrowserHandoffAttachResult, error) {
	return rpcRequest[BrowserHandoffAttachResult](ctx, c, "autohand.browserHandoff.attachLatest", map[string]interface{}{})
}

// StartAutomode starts an auto-mode task and returns when the CLI accepts it.
func (c *RPCClient) StartAutomode(ctx context.Context, params *AutomodeStartParams) (*AutomodeStartResult, error) {
	if params == nil {
		params = &AutomodeStartParams{}
	}
	return rpcRequest[AutomodeStartResult](ctx, c, "autohand.automode.start", params)
}

// GetAutomodeStatus returns live flags and optional persisted session state.
func (c *RPCClient) GetAutomodeStatus(ctx context.Context) (*AutomodeStatusResult, error) {
	return rpcRequest[AutomodeStatusResult](ctx, c, "autohand.automode.status", map[string]interface{}{})
}

// PauseAutomode pauses the active auto-mode session.
func (c *RPCClient) PauseAutomode(ctx context.Context) (*AutomodePauseResult, error) {
	return rpcRequest[AutomodePauseResult](ctx, c, "autohand.automode.pause", map[string]interface{}{})
}

// ResumeAutomode resumes a paused auto-mode session.
func (c *RPCClient) ResumeAutomode(ctx context.Context) (*AutomodeResumeResult, error) {
	return rpcRequest[AutomodeResumeResult](ctx, c, "autohand.automode.resume", map[string]interface{}{})
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

// GetSkillsRegistry returns the community skill registry.
func (c *RPCClient) GetSkillsRegistry(ctx context.Context, params *GetSkillsRegistryParams) (*GetSkillsRegistryResult, error) {
	if params == nil {
		params = &GetSkillsRegistryParams{}
	}
	return rpcRequest[GetSkillsRegistryResult](ctx, c, "autohand.getSkillsRegistry", params)
}

// InstallSkill installs a registry skill into user or project scope.
func (c *RPCClient) InstallSkill(ctx context.Context, params *InstallSkillParams) (*InstallSkillResult, error) {
	if params == nil || params.SkillName == "" {
		return nil, fmt.Errorf("install skill: skill name is required")
	}
	if params.Scope != SkillInstallScopeUser && params.Scope != SkillInstallScopeProject {
		return nil, fmt.Errorf("install skill: scope must be %q or %q", SkillInstallScopeUser, SkillInstallScopeProject)
	}
	return rpcRequest[InstallSkillResult](ctx, c, "autohand.installSkill", params)
}

// ListMCPServers returns all known MCP servers and their status.
func (c *RPCClient) ListMCPServers(ctx context.Context) (*MCPListServersResult, error) {
	return rpcRequest[MCPListServersResult](ctx, c, "autohand.mcp.listServers", map[string]interface{}{})
}

// ListMCPTools returns all available MCP tools, optionally filtered by server.
func (c *RPCClient) ListMCPTools(ctx context.Context, params *MCPListToolsParams) (*MCPListToolsResult, error) {
	if params == nil {
		params = &MCPListToolsParams{}
	}
	return rpcRequest[MCPListToolsResult](ctx, c, "autohand.mcp.listTools", params)
}

// GetMCPServerConfigs returns the configured MCP server definitions.
func (c *RPCClient) GetMCPServerConfigs(ctx context.Context) (*MCPGetServerConfigsResult, error) {
	return rpcRequest[MCPGetServerConfigsResult](ctx, c, "autohand.mcp.getServerConfigs", map[string]interface{}{})
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

func (c *RPCClient) GetGoal(ctx context.Context) (*GoalSnapshot, error) {
	return goalRequest[GoalSnapshot](ctx, c, "autohand.goal.get", map[string]interface{}{})
}
func (c *RPCClient) CreateGoal(ctx context.Context, params *GoalCreateParams) (*GoalMutationResult, error) {
	return goalRequest[GoalMutationResult](ctx, c, "autohand.goal.create", params)
}
func (c *RPCClient) UpdateGoal(ctx context.Context, params *GoalUpdateParams) (*GoalMutationResult, error) {
	return goalRequest[GoalMutationResult](ctx, c, "autohand.goal.update", params)
}
func (c *RPCClient) QueueGoal(ctx context.Context, params *GoalCreateParams) (*GoalMutationResult, error) {
	return goalRequest[GoalMutationResult](ctx, c, "autohand.goal.queue", params)
}
func (c *RPCClient) StartQueuedGoal(ctx context.Context) (*GoalMutationResult, error) {
	return goalRequest[GoalMutationResult](ctx, c, "autohand.goal.startQueued", map[string]interface{}{})
}
func (c *RPCClient) ListGoalTemplates(ctx context.Context) ([]GoalTemplateMetadata, error) {
	return goalRequestValue[[]GoalTemplateMetadata](ctx, c, "autohand.goal.listTemplates", map[string]interface{}{})
}
func (c *RPCClient) ClearGoal(ctx context.Context) (*GoalMutationResult, error) {
	return goalRequest[GoalMutationResult](ctx, c, "autohand.goal.clear", map[string]interface{}{})
}

func goalRequest[T any](ctx context.Context, client *RPCClient, method string, params interface{}) (*T, error) {
	result, err := goalRequestValue[T](ctx, client, method, params)
	return &result, err
}
func goalRequestValue[T any](ctx context.Context, client *RPCClient, method string, params interface{}) (T, error) {
	var result T
	response, err := client.transport.Request(ctx, method, params)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(response, &result); err != nil {
		return result, fmt.Errorf("unmarshal %s result: %w", method, err)
	}
	return result, nil
}

// StartAutoresearch initializes or resumes a persisted autoresearch loop.
func (c *RPCClient) StartAutoresearch(ctx context.Context, params *AutoresearchStartParams) (*AutoresearchStartResult, error) {
	return autoresearchRequest[AutoresearchStartResult](ctx, c, "autohand.autoresearch.start", params)
}

// GetAutoresearchStatus returns current persisted autoresearch state.
func (c *RPCClient) GetAutoresearchStatus(ctx context.Context) (*AutoresearchStatusResult, error) {
	return autoresearchRequest[AutoresearchStatusResult](ctx, c, "autohand.autoresearch.status", map[string]interface{}{})
}

// StopAutoresearch pauses autoresearch without deleting persisted state.
func (c *RPCClient) StopAutoresearch(ctx context.Context) (*AutoresearchStopResult, error) {
	return autoresearchRequest[AutoresearchStopResult](ctx, c, "autohand.autoresearch.stop", map[string]interface{}{})
}

// GetAutoresearchHistory lists persisted attempts.
func (c *RPCClient) GetAutoresearchHistory(ctx context.Context) (*AutoresearchHistoryResult, error) {
	return autoresearchRequest[AutoresearchHistoryResult](ctx, c, "autohand.autoresearch.history", map[string]interface{}{})
}

// ReplayAutoresearch re-evaluates a candidate in an isolated worktree.
func (c *RPCClient) ReplayAutoresearch(ctx context.Context, params *AutoresearchReplayParams) (*AutoresearchReplayResult, error) {
	return autoresearchRequest[AutoresearchReplayResult](ctx, c, "autohand.autoresearch.replay", params)
}

// RescoreAutoresearch reapplies current decision policy to persisted measurements.
func (c *RPCClient) RescoreAutoresearch(ctx context.Context, params *AutoresearchRescoreParams) (*AutoresearchRescoreResult, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return autoresearchRequest[AutoresearchRescoreResult](ctx, c, "autohand.autoresearch.rescore", params)
}

// CompareAutoresearch compares persisted evidence for two attempts.
func (c *RPCClient) CompareAutoresearch(ctx context.Context, params *AutoresearchCompareParams) (*AutoresearchCompareResult, error) {
	return autoresearchRequest[AutoresearchCompareResult](ctx, c, "autohand.autoresearch.compare", params)
}

// GetAutoresearchPareto returns the current constraint-passing Pareto frontier.
func (c *RPCClient) GetAutoresearchPareto(ctx context.Context) (*AutoresearchParetoResult, error) {
	return autoresearchRequest[AutoresearchParetoResult](ctx, c, "autohand.autoresearch.pareto", map[string]interface{}{})
}

// PinAutoresearch pins or unpins a candidate's replay artifacts.
func (c *RPCClient) PinAutoresearch(ctx context.Context, params *AutoresearchPinParams) (*AutoresearchPinResult, error) {
	return autoresearchRequest[AutoresearchPinResult](ctx, c, "autohand.autoresearch.pin", params)
}

// PruneAutoresearch previews or applies artifact retention.
func (c *RPCClient) PruneAutoresearch(ctx context.Context, params *AutoresearchPruneParams) (*AutoresearchPruneResult, error) {
	if params == nil {
		params = &AutoresearchPruneParams{}
	}
	return autoresearchRequest[AutoresearchPruneResult](ctx, c, "autohand.autoresearch.prune", params)
}

func autoresearchRequest[T any](ctx context.Context, client *RPCClient, method string, params interface{}) (*T, error) {
	response, err := client.transport.Request(ctx, method, params)
	if err != nil {
		return nil, err
	}
	var result T
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("unmarshal %s result: %w", method, err)
	}
	return &result, nil
}

func rpcRequest[T any](ctx context.Context, client *RPCClient, method string, params interface{}) (*T, error) {
	response, err := client.transport.Request(ctx, method, params)
	if err != nil {
		return nil, err
	}
	var result T
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("unmarshal %s result: %w", method, err)
	}
	return &result, nil
}

// Request sends a custom RPC request.
func (c *RPCClient) Request(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	return c.transport.Request(ctx, method, params)
}

// Events returns a channel of SDK events.
func (c *RPCClient) Events(ctx context.Context) <-chan Event {
	ch := make(chan Event, 256)
	done := make(chan struct{})
	c.mu.Lock()
	if c.subscribers == nil {
		c.subscribers = make(map[uint64]eventSubscription)
	}
	id := c.nextSubscriberID
	c.nextSubscriberID++
	c.subscribers[id] = eventSubscription{events: ch, done: done}
	c.mu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
			c.mu.Lock()
			if subscriber, ok := c.subscribers[id]; ok {
				delete(c.subscribers, id)
				close(subscriber.events)
				close(subscriber.done)
			}
			c.mu.Unlock()
		case <-done:
		}
	}()
	return ch
}

func (c *RPCClient) queueEvent(event Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, subscriber := range c.subscribers {
		select {
		case subscriber.events <- event:
		default:
			// A slow subscriber must not block the JSON-RPC reader or other subscribers.
		}
	}
}

func (c *RPCClient) closeSubscribers() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, subscriber := range c.subscribers {
		delete(c.subscribers, id)
		close(subscriber.events)
		close(subscriber.done)
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

	queueAutoresearchLifecycle := func(phase AutoresearchPhase) func(json.RawMessage) {
		return func(params json.RawMessage) {
			var event AutoresearchLifecycleEvent
			if err := json.Unmarshal(params, &event); err != nil {
				return
			}
			event.Type = "autoresearch"
			event.Phase = phase
			c.queueEvent(event)
		}
	}
	c.transport.OnNotification("autohand.autoresearch.start", queueAutoresearchLifecycle(AutoresearchPhaseStart))
	c.transport.OnNotification("autohand.autoresearch.status", queueAutoresearchLifecycle(AutoresearchPhaseStatus))
	c.transport.OnNotification("autohand.autoresearch.pause", queueAutoresearchLifecycle(AutoresearchPhasePause))
	c.transport.OnNotification("autohand.autoresearch.event", func(params json.RawMessage) {
		var event AutoresearchOperationEvent
		if err := json.Unmarshal(params, &event); err != nil {
			return
		}
		event.Type = "autoresearch"
		c.queueEvent(event)
	})
}
