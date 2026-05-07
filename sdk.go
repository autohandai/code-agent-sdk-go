package autohand

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// SDK is the main Autohand SDK for interacting with the CLI.
type SDK struct {
	cfg     *Config
	client  *RPCClient
	started bool
	mu      sync.Mutex
}

// NewSDK creates a new SDK instance.
func NewSDK(cfg *Config) *SDK {
	if cfg == nil {
		cfg = &Config{}
	}
	return &SDK{
		cfg:    cfg,
		client: NewRPCClient(cfg),
	}
}

// Start starts the SDK and initializes the CLI subprocess.
func (s *SDK) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	if err := s.client.Start(ctx, s.cfg); err != nil {
		return fmt.Errorf("start SDK: %w", err)
	}
	s.started = true

	if s.cfg.PermissionMode != "" && s.cfg.PermissionMode != PermissionInteractive {
		if err := s.client.SetPermissionMode(ctx, s.cfg.PermissionMode); err != nil {
			return fmt.Errorf("set permission mode: %w", err)
		}
	}

	if s.cfg.PlanMode {
		if err := s.client.SetPlanMode(ctx, true); err != nil {
			return fmt.Errorf("set plan mode: %w", err)
		}
	}

	return nil
}

// Stop stops the SDK and terminates the CLI subprocess.
func (s *SDK) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	if err := s.client.Stop(); err != nil {
		return err
	}
	s.started = false
	return nil
}

// Close is an alias for Stop.
func (s *SDK) Close() error {
	return s.Stop()
}

// Prompt sends a prompt to the agent (non-streaming).
func (s *SDK) Prompt(ctx context.Context, params *PromptParams) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.Prompt(ctx, params)
}

// StreamPrompt streams a prompt with real-time events.
// It races between the prompt request completion and incoming events so that
// events are yielded as soon as they arrive, even before the prompt call
// returns.
func (s *SDK) StreamPrompt(ctx context.Context, params *PromptParams) (<-chan Event, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}

	events := s.client.Events(ctx)
	out := make(chan Event, 256)

	go func() {
		defer close(out)

		promptSettled := false
		var promptErr error

		promptDone := make(chan struct{}, 1)
		go func() {
			if err := s.client.Prompt(ctx, params); err != nil {
				promptErr = err
			}
			promptSettled = true
			close(promptDone)
		}()

		for {
			if promptSettled && promptErr != nil {
				out <- ErrorEvent{
					Type:    "error",
					Code:    -1,
					Message: promptErr.Error(),
				}
				return
			}

			var event Event
			var ok bool

			if promptSettled {
				select {
				case event, ok = <-events:
					if !ok {
						return
					}
				case <-ctx.Done():
					return
				}
			} else {
				select {
				case event, ok = <-events:
					if !ok {
						return
					}
				case <-promptDone:
					continue
				case <-ctx.Done():
					return
				}
			}

			out <- event
			if e, ok := event.(AgentEndEvent); ok && e.Type == "agent_end" {
				return
			}
		}
	}()

	return out, nil
}

// Interrupt aborts the current operation.
func (s *SDK) Interrupt(ctx context.Context) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.Abort(ctx)
}

// Abort is an alias for Interrupt.
func (s *SDK) Abort(ctx context.Context) error {
	return s.Interrupt(ctx)
}

// SetPermissionMode changes the permission mode.
func (s *SDK) SetPermissionMode(ctx context.Context, mode PermissionMode) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	s.cfg.PermissionMode = mode
	return s.client.SetPermissionMode(ctx, mode)
}

// SetPlanMode enables or disables plan mode.
func (s *SDK) SetPlanMode(ctx context.Context, enabled bool) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	s.cfg.PlanMode = enabled
	return s.client.SetPlanMode(ctx, enabled)
}

// EnablePlanMode enables plan mode.
func (s *SDK) EnablePlanMode(ctx context.Context) error {
	return s.SetPlanMode(ctx, true)
}

// DisablePlanMode disables plan mode.
func (s *SDK) DisablePlanMode(ctx context.Context) error {
	return s.SetPlanMode(ctx, false)
}

// SetModel changes the model.
func (s *SDK) SetModel(ctx context.Context, model string) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	s.cfg.Model = model
	return s.client.SetModel(ctx, model)
}

// SetMaxThinkingTokens sets the max thinking tokens.
func (s *SDK) SetMaxThinkingTokens(ctx context.Context, tokens int) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.SetMaxThinkingTokens(ctx, tokens)
}

// ApplyFlagSettings applies flag settings at runtime.
func (s *SDK) ApplyFlagSettings(ctx context.Context, settings map[string]interface{}) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.ApplyFlagSettings(ctx, settings)
}

// GetState returns the current agent state.
func (s *SDK) GetState(ctx context.Context) (*GetStateResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetState(ctx)
}

// GetMessages returns conversation messages.
func (s *SDK) GetMessages(ctx context.Context, limit int) (*GetMessagesResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetMessages(ctx, limit)
}

// SupportedModels returns available models.
func (s *SDK) SupportedModels(ctx context.Context) ([]ModelInfo, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetSupportedModels(ctx)
}

// GetContextUsage returns context window usage.
func (s *SDK) GetContextUsage(ctx context.Context) (*ContextUsage, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetContextUsage(ctx)
}

// AccountInfo returns account information.
func (s *SDK) AccountInfo(ctx context.Context) (*AccountInfo, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetAccountInfo(ctx)
}

// ToggleMCPServer enables or disables an MCP server.
func (s *SDK) ToggleMCPServer(ctx context.Context, serverName string, enabled bool) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.ToggleMCPServer(ctx, serverName, enabled)
}

// ReconnectMCPServer reconnects an MCP server.
func (s *SDK) ReconnectMCPServer(ctx context.Context, serverName string) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.ReconnectMCPServer(ctx, serverName)
}

// SetMCPServers sets MCP server configurations.
func (s *SDK) SetMCPServers(ctx context.Context, servers map[string]MCPServerConfig) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.SetMCPServers(ctx, servers)
}

// AllowPermission approves a permission request.
func (s *SDK) AllowPermission(ctx context.Context, requestID string, scope DecisionScope) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.PermissionResponse(ctx, requestID, allowDecision(scope))
}

// DenyPermission denies a permission request.
func (s *SDK) DenyPermission(ctx context.Context, requestID string, scope DecisionScope) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.PermissionResponse(ctx, requestID, denyDecision(scope))
}

// PermissionResponse responds to a permission request with a specific decision.
func (s *SDK) PermissionResponse(ctx context.Context, requestID string, decision PermissionDecision) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.PermissionResponse(ctx, requestID, decision)
}

// Events returns a channel of all SDK events.
func (s *SDK) Events(ctx context.Context) (<-chan Event, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.Events(ctx), nil
}

// GetHooks returns all hooks.
func (s *SDK) GetHooks(ctx context.Context) (*HooksSettings, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetHooks(ctx)
}

// AddHook adds a hook.
func (s *SDK) AddHook(ctx context.Context, hook HookDefinition) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.AddHook(ctx, hook)
}

// RemoveHook removes a hook.
func (s *SDK) RemoveHook(ctx context.Context, event HookEvent, index int) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.RemoveHook(ctx, event, index)
}

// ToggleHook toggles a hook.
func (s *SDK) ToggleHook(ctx context.Context, event HookEvent, index int) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.ToggleHook(ctx, event, index)
}

// TestHook tests a hook.
func (s *SDK) TestHook(ctx context.Context, hook HookDefinition) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.TestHook(ctx, hook)
}

// SetHooksSettings updates hooks settings.
func (s *SDK) SetHooksSettings(ctx context.Context, settings *HooksSettings) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	s.cfg.Hooks = settings
	_, err := s.client.Request(ctx, "autohand.hooks.setSettings", map[string]interface{}{
		"settings": settings,
	})
	return err
}

// SaveSession saves the current session to disk.
func (s *SDK) SaveSession(ctx context.Context) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.SaveSession(ctx)
}

// ResumeSession resumes a previous session.
func (s *SDK) ResumeSession(ctx context.Context, sessionID string) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	s.cfg.Resume = true
	s.cfg.SessionID = sessionID
	_, err := s.client.Request(ctx, "autohand.resumeSession", map[string]interface{}{
		"sessionId": sessionID,
	})
	return err
}

// GetStats returns session statistics.
func (s *SDK) GetStats(ctx context.Context) (*SessionStats, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetStats(ctx)
}

// GetSessionMetadata returns session metadata.
func (s *SDK) GetSessionMetadata(ctx context.Context) (*SessionMetadata, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetSessionMetadata(ctx)
}

// SupportedAgents returns available subagents.
func (s *SDK) SupportedAgents(ctx context.Context) ([]AgentInfo, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	resp, err := s.client.Request(ctx, "autohand.getSupportedAgents", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Agents []AgentInfo `json:"agents"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal agents: %w", err)
	}
	return result.Agents, nil
}

// MCPServerStatus returns MCP server statuses.
func (s *SDK) MCPServerStatus(ctx context.Context) (map[string]interface{}, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	resp, err := s.client.Request(ctx, "autohand.mcp.serverStatus", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal mcp status: %w", err)
	}
	return result, nil
}

// RewindFiles rewinds file modifications to a checkpoint.
func (s *SDK) RewindFiles(ctx context.Context, checkpointID string) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.RewindFiles(ctx, checkpointID)
}

// SeedReadState seeds read state for deterministic runs.
func (s *SDK) SeedReadState(ctx context.Context, state map[string]interface{}) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.SeedReadState(ctx, state)
}

// StreamInput sends streaming input to the agent.
func (s *SDK) StreamInput(ctx context.Context, input string) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.StreamInput(ctx, input)
}

// SetSystemPrompt sets the system prompt.
func (s *SDK) SetSystemPrompt(ctx context.Context, prompt string) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	s.cfg.SysPrompt = prompt
	return s.client.SetSystemPrompt(ctx, prompt)
}

// AppendSystemPrompt appends to the system prompt.
func (s *SDK) AppendSystemPrompt(ctx context.Context, prompt string) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	s.cfg.AppendSysPrompt += prompt
	return s.client.AppendSystemPrompt(ctx, prompt)
}

// LoadAgentsMd loads an AGENTS.md file from a path or URL.
func (s *SDK) LoadAgentsMd(ctx context.Context, source string) (string, error) {
	content, err := LoadAgentsMd(source)
	if err != nil {
		return "", err
	}
	return content, nil
}

// CreateDefaultAgentsMd creates a default AGENTS.md template.
func (s *SDK) CreateDefaultAgentsMd(projectName string) string {
	return CreateDefaultAgentsMd(projectName)
}

// SetAgentsMdAsPrompt sets AGENTS.md content as a prompt parameter.
func (s *SDK) SetAgentsMdAsPrompt(ctx context.Context, params *PromptParams, content string) *PromptParams {
	if params == nil {
		params = &PromptParams{}
	}
	params.AgentsMd = content
	return params
}

// ReloadPlugins reloads CLI plugins.
func (s *SDK) ReloadPlugins(ctx context.Context) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.ReloadPlugins(ctx)
}

// IsStarted returns whether the SDK is started.
func (s *SDK) IsStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

// IsConnected returns whether the CLI process is running.
func (s *SDK) IsConnected() bool {
	return s.client.IsConnected()
}

// Config returns a copy of the current configuration.
func (s *SDK) Config() *Config {
	cfg := *s.cfg
	return &cfg
}

// UpdateConfig merges the provided configuration into the current config.
func (s *SDK) UpdateConfig(cfg *Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg.CWD != "" {
		s.cfg.CWD = cfg.CWD
	}
	if cfg.CLIPath != "" {
		s.cfg.CLIPath = cfg.CLIPath
	}
	if cfg.Debug {
		s.cfg.Debug = cfg.Debug
	}
	if cfg.Timeout > 0 {
		s.cfg.Timeout = cfg.Timeout
	}
	if cfg.Model != "" {
		s.cfg.Model = cfg.Model
	}
	if cfg.FallbackModel != "" {
		s.cfg.FallbackModel = cfg.FallbackModel
	}
	if cfg.MaxTurns > 0 {
		s.cfg.MaxTurns = cfg.MaxTurns
	}
	if cfg.Provider != "" {
		s.cfg.Provider = cfg.Provider
	}
	if cfg.APIKey != "" {
		s.cfg.APIKey = cfg.APIKey
	}
	if cfg.BaseURL != "" {
		s.cfg.BaseURL = cfg.BaseURL
	}
	if cfg.PermissionMode != "" {
		s.cfg.PermissionMode = cfg.PermissionMode
	}
	if cfg.Yolo != "" {
		s.cfg.Yolo = cfg.Yolo
	}
	if cfg.SysPrompt != "" {
		s.cfg.SysPrompt = cfg.SysPrompt
	}
	if cfg.AppendSysPrompt != "" {
		s.cfg.AppendSysPrompt = cfg.AppendSysPrompt
	}
	if len(cfg.Skills) > 0 {
		s.cfg.Skills = cfg.Skills
	}
	if len(cfg.SkillRefs) > 0 {
		s.cfg.SkillRefs = cfg.SkillRefs
	}
	if cfg.MaxTokens > 0 {
		s.cfg.MaxTokens = cfg.MaxTokens
	}
	if cfg.AgentsMd != nil {
		s.cfg.AgentsMd = cfg.AgentsMd
	}
}

func (s *SDK) ensureStarted(ctx context.Context) error {
	if !s.started {
		return s.Start(ctx)
	}
	return nil
}
