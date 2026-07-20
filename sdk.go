package autohand

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	if cfg.Provider == "" && os.Getenv("AUTOHAND_AI_API_KEY") != "" {
		cfg.Provider = ProviderAutohandAI
	}
	if cfg.Provider == ProviderAutohandAI {
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("AUTOHAND_AI_API_KEY")
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = os.Getenv("AUTOHAND_AI_BASE_URL")
		}
		if cfg.AutohandAIPlan == "" {
			cfg.AutohandAIPlan = os.Getenv("AUTOHAND_AI_PLAN")
		}
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
	rollback := func(cause error) error {
		if stopErr := s.client.Stop(); stopErr != nil {
			return fmt.Errorf("%w (rollback failed: %v)", cause, stopErr)
		}
		return cause
	}
	if _, err := s.client.GetState(ctx); err != nil {
		return rollback(fmt.Errorf("wait for CLI readiness: %w", err))
	}
	if s.cfg.Features != nil {
		if err := s.client.ApplyFlagSettings(ctx, map[string]interface{}{"features": s.cfg.Features}); err != nil {
			return rollback(fmt.Errorf("apply startup feature settings: %w", err))
		}
	}

	if s.cfg.PermissionMode != "" && s.cfg.PermissionMode != PermissionInteractive {
		if err := s.client.SetPermissionMode(ctx, s.cfg.PermissionMode); err != nil {
			return rollback(fmt.Errorf("set permission mode: %w", err))
		}
	}

	if s.cfg.PlanMode {
		if err := s.client.SetPlanMode(ctx, true); err != nil {
			return rollback(fmt.Errorf("set plan mode: %w", err))
		}
	}

	s.started = true
	return nil
}

// StreamCommand streams a validated slash command.
func (s *SDK) StreamCommand(ctx context.Context, command string, args []string, opts *PromptParams) (<-chan Event, error) {
	formatted, err := FormatSlashCommand(command, args...)
	if err != nil {
		return nil, err
	}
	if opts == nil {
		opts = &PromptParams{}
	}
	opts.Message = formatted
	return s.StreamPrompt(ctx, opts)
}

// SupportedCommands returns normalized commands discovered from the live CLI.
func (s *SDK) SupportedCommands(ctx context.Context) ([]string, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	commands, err := s.client.GetSupportedCommands(ctx)
	if err != nil {
		return nil, err
	}
	for i, command := range commands {
		command = strings.TrimSpace(command)
		if !strings.HasPrefix(command, "/") {
			command = "/" + command
		}
		commands[i] = command
	}
	return commands, nil
}

func (s *SDK) SupportsCommand(ctx context.Context, command string) (bool, error) {
	command = "/" + strings.TrimPrefix(strings.TrimSpace(command), "/")
	commands, err := s.SupportedCommands(ctx)
	if err != nil {
		return false, err
	}
	for _, candidate := range commands {
		if candidate == command {
			return true, nil
		}
	}
	return false, nil
}

func (s *SDK) GetGoal(ctx context.Context) (*GoalSnapshot, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetGoal(ctx)
}
func (s *SDK) CreateGoal(ctx context.Context, p *GoalCreateParams) (*GoalMutationResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.CreateGoal(ctx, p)
}
func (s *SDK) UpdateGoal(ctx context.Context, p *GoalUpdateParams) (*GoalMutationResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.UpdateGoal(ctx, p)
}
func (s *SDK) QueueGoal(ctx context.Context, p *GoalCreateParams) (*GoalMutationResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.QueueGoal(ctx, p)
}
func (s *SDK) StartQueuedGoal(ctx context.Context) (*GoalMutationResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.StartQueuedGoal(ctx)
}
func (s *SDK) ListGoalTemplates(ctx context.Context) ([]GoalTemplateMetadata, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.ListGoalTemplates(ctx)
}
func (s *SDK) ClearGoal(ctx context.Context) (*GoalMutationResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.ClearGoal(ctx)
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

	streamCtx, cancelStream := context.WithCancel(ctx)
	events := s.client.Events(streamCtx)
	out := make(chan Event, 256)

	go func() {
		defer close(out)
		defer cancelStream()

		promptDone := make(chan error, 1)
		go func() {
			promptDone <- s.client.Prompt(streamCtx, params)
		}()
		promptSettled := false

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}
				select {
				case out <- event:
				case <-streamCtx.Done():
					return
				}
				if e, ok := event.(AgentEndEvent); ok && e.Type == "agent_end" {
					return
				}
			case err := <-promptDone:
				if promptSettled {
					continue
				}
				promptSettled = true
				if err != nil {
					select {
					case out <- ErrorEvent{Type: "error", Code: -1, Message: err.Error()}:
					case <-streamCtx.Done():
					}
					return
				}
			case <-streamCtx.Done():
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

// Reset replaces the active conversation and returns the new session ID.
func (s *SDK) Reset(ctx context.Context) (*ResetResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.Reset(ctx)
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

// CreateBrowserHandoff creates a browser continuation token for the active session.
func (s *SDK) CreateBrowserHandoff(ctx context.Context, params *BrowserHandoffCreateParams) (*BrowserHandoffCreateResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.CreateBrowserHandoff(ctx, params)
}

// AttachBrowserHandoff restores the session referenced by a handoff token.
func (s *SDK) AttachBrowserHandoff(ctx context.Context, params *BrowserHandoffAttachParams) (*BrowserHandoffAttachResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.AttachBrowserHandoff(ctx, params)
}

// AttachLatestBrowserHandoff restores the newest unexpired browser handoff.
func (s *SDK) AttachLatestBrowserHandoff(ctx context.Context) (*BrowserHandoffAttachResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.AttachLatestBrowserHandoff(ctx)
}

// StartAutomode starts an auto-mode task and returns when the CLI accepts it.
func (s *SDK) StartAutomode(ctx context.Context, params *AutomodeStartParams) (*AutomodeStartResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.StartAutomode(ctx, params)
}

// GetAutomodeStatus returns live flags and optional persisted session state.
func (s *SDK) GetAutomodeStatus(ctx context.Context) (*AutomodeStatusResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetAutomodeStatus(ctx)
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

// StartAutoresearch initializes or resumes a persisted autoresearch loop.
func (s *SDK) StartAutoresearch(ctx context.Context, params *AutoresearchStartParams) (*AutoresearchStartResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.StartAutoresearch(ctx, params)
}

// GetAutoresearchStatus returns current persisted autoresearch state.
func (s *SDK) GetAutoresearchStatus(ctx context.Context) (*AutoresearchStatusResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetAutoresearchStatus(ctx)
}

// StopAutoresearch pauses autoresearch without deleting persisted state.
func (s *SDK) StopAutoresearch(ctx context.Context) (*AutoresearchStopResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.StopAutoresearch(ctx)
}

// GetAutoresearchHistory lists persisted attempts.
func (s *SDK) GetAutoresearchHistory(ctx context.Context) (*AutoresearchHistoryResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetAutoresearchHistory(ctx)
}

// ReplayAutoresearch re-evaluates a candidate in an isolated worktree.
func (s *SDK) ReplayAutoresearch(ctx context.Context, params *AutoresearchReplayParams) (*AutoresearchReplayResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.ReplayAutoresearch(ctx, params)
}

// RescoreAutoresearch reapplies current policy to persisted measurements.
func (s *SDK) RescoreAutoresearch(ctx context.Context, params *AutoresearchRescoreParams) (*AutoresearchRescoreResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.RescoreAutoresearch(ctx, params)
}

// CompareAutoresearch compares persisted evidence for two attempts.
func (s *SDK) CompareAutoresearch(ctx context.Context, params *AutoresearchCompareParams) (*AutoresearchCompareResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.CompareAutoresearch(ctx, params)
}

// GetAutoresearchPareto returns the current constraint-passing Pareto frontier.
func (s *SDK) GetAutoresearchPareto(ctx context.Context) (*AutoresearchParetoResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetAutoresearchPareto(ctx)
}

// PinAutoresearch pins or unpins a candidate's replay artifacts.
func (s *SDK) PinAutoresearch(ctx context.Context, params *AutoresearchPinParams) (*AutoresearchPinResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.PinAutoresearch(ctx, params)
}

// PruneAutoresearch previews or applies artifact retention.
func (s *SDK) PruneAutoresearch(ctx context.Context, params *AutoresearchPruneParams) (*AutoresearchPruneResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.PruneAutoresearch(ctx, params)
}

// SetMCPServers sets MCP server configurations.
func (s *SDK) SetMCPServers(ctx context.Context, servers map[string]MCPServerConfig) error {
	if err := s.ensureStarted(ctx); err != nil {
		return err
	}
	return s.client.SetMCPServers(ctx, servers)
}

// GetSkillsRegistry returns the community skill registry.
func (s *SDK) GetSkillsRegistry(ctx context.Context, params *GetSkillsRegistryParams) (*GetSkillsRegistryResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetSkillsRegistry(ctx, params)
}

// InstallSkill installs a registry skill into user or project scope.
func (s *SDK) InstallSkill(ctx context.Context, params *InstallSkillParams) (*InstallSkillResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.InstallSkill(ctx, params)
}

// ListMCPServers returns all known MCP servers and their status.
func (s *SDK) ListMCPServers(ctx context.Context) (*MCPListServersResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.ListMCPServers(ctx)
}

// ListMCPTools returns available MCP tools, optionally filtered by server.
func (s *SDK) ListMCPTools(ctx context.Context, params *MCPListToolsParams) (*MCPListToolsResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.ListMCPTools(ctx, params)
}

// GetMCPServerConfigs returns the configured MCP server definitions.
func (s *SDK) GetMCPServerConfigs(ctx context.Context) (*MCPGetServerConfigsResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetMCPServerConfigs(ctx)
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
	if !s.IsStarted() {
		return s.Start(ctx)
	}
	return nil
}
