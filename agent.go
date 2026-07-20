package autohand

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Run represents a single agent run.
type Run struct {
	id     string
	sdk    *SDK
	params *PromptParams

	mu        sync.Mutex
	events    []Event
	waiters   []chan struct{}
	completed bool
	aborted   bool
	runErr    error
	text      string
	started   bool
	startOnce sync.Once
	resultCh  chan RunResult
}

func newRun(sdk *SDK, params *PromptParams, id string) *Run {
	if id == "" {
		id = fmt.Sprintf("run_%d", time.Now().UnixNano())
	}
	return &Run{
		id:       id,
		sdk:      sdk,
		params:   params,
		resultCh: make(chan RunResult, 1),
	}
}

// Stream returns a channel of events for this run.
func (r *Run) Stream(ctx context.Context) (<-chan Event, error) {
	r.startOnce.Do(func() {
		go r.pump(ctx)
	})

	out := make(chan Event, 256)
	go func() {
		defer close(out)
		idx := 0
		for {
			r.mu.Lock()
			for idx < len(r.events) {
				evt := r.events[idx]
				idx++
				r.mu.Unlock()
				select {
				case out <- evt:
				case <-ctx.Done():
					return
				}
				r.mu.Lock()
			}

			if r.completed {
				r.mu.Unlock()
				if r.runErr != nil {
					return
				}
				return
			}

			waiter := make(chan struct{})
			r.waiters = append(r.waiters, waiter)
			r.mu.Unlock()

			select {
			case <-waiter:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// Wait waits for the run to complete and returns the result.
func (r *Run) Wait(ctx context.Context) (*RunResult, error) {
	r.startOnce.Do(func() {
		go r.pump(ctx)
	})

	select {
	case result := <-r.resultCh:
		return &result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Abort aborts the run.
func (r *Run) Abort(ctx context.Context) error {
	r.mu.Lock()
	r.aborted = true
	r.mu.Unlock()
	return r.sdk.Interrupt(ctx)
}

func (r *Run) pump(ctx context.Context) {
	events, err := r.sdk.StreamPrompt(ctx, r.params)
	if err != nil {
		r.mu.Lock()
		r.runErr = err
		r.completed = true
		r.mu.Unlock()
		r.notify()
		r.resultCh <- RunResult{
			ID:     r.id,
			Status: "error",
			Events: r.copyEvents(),
		}
		return
	}

	for event := range events {
		r.record(event)
		if e, ok := event.(AgentEndEvent); ok && e.Type == "agent_end" {
			break
		}
	}

	r.mu.Lock()
	r.completed = true
	status := "completed"
	if r.aborted {
		status = "aborted"
	}
	r.mu.Unlock()
	r.notify()

	r.resultCh <- RunResult{
		ID:     r.id,
		Status: status,
		Text:   r.text,
		Events: r.copyEvents(),
	}
}

func (r *Run) record(event Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)

	switch e := event.(type) {
	case MessageUpdateEvent:
		r.text += e.Delta
	case MessageEndEvent:
		r.text = e.Content
	}

	r.notifyLocked()
}

func (r *Run) notify() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifyLocked()
}

func (r *Run) notifyLocked() {
	waiters := r.waiters
	r.waiters = nil
	for _, w := range waiters {
		close(w)
	}
}

func (r *Run) copyEvents() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	events := make([]Event, len(r.events))
	copy(events, r.events)
	return events
}

// Agent is a high-level agent session.
type Agent struct {
	sdk *SDK
}

// NewAgent creates and starts an agent session.
func NewAgent(ctx context.Context, cfg *Config) (*Agent, error) {
	sdkCfg := *cfg
	if cfg.Instructions != "" {
		if sdkCfg.AppendSysPrompt != "" {
			sdkCfg.AppendSysPrompt += "\n" + cfg.Instructions
		} else {
			sdkCfg.AppendSysPrompt = cfg.Instructions
		}
	}
	sdk := NewSDK(&sdkCfg)
	if err := sdk.Start(ctx); err != nil {
		return nil, fmt.Errorf("start agent: %w", err)
	}
	return &Agent{sdk: sdk}, nil
}

// NewAgentFromSDK wraps an existing SDK instance.
func NewAgentFromSDK(sdk *SDK) *Agent {
	return &Agent{sdk: sdk}
}

// Send creates a run without waiting for it to finish.
func (a *Agent) Send(ctx context.Context, input interface{}, opts *PromptParams) (*Run, error) {
	params := toPromptParams(input, opts)
	return newRun(a.sdk, params, ""), nil
}

// Run runs a prompt to completion and returns the result.
func (a *Agent) Run(ctx context.Context, input interface{}, opts *PromptParams) (*RunResult, error) {
	run, err := a.Send(ctx, input, opts)
	if err != nil {
		return nil, err
	}
	return run.Wait(ctx)
}

// Stream streams a prompt directly.
func (a *Agent) Stream(ctx context.Context, input interface{}, opts *PromptParams) (<-chan Event, error) {
	run, err := a.Send(ctx, input, opts)
	if err != nil {
		return nil, err
	}
	return run.Stream(ctx)
}

// JSON waits for the run to complete and parses the final text as JSON.
// An optional validator can be provided to type-check and transform the result.
func (r *Run) JSON(opts ...JsonParseOptions[json.RawMessage]) (json.RawMessage, error) {
	_, err := r.Wait(context.Background())
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	text := r.text
	r.mu.Unlock()

	parsed, parseErr := parseJsonText(text)
	if parseErr != nil {
		return nil, &StructuredOutputError{
			Message: fmt.Sprintf("Failed to parse JSON: %s", parseErr.Error()),
			RawText: text,
		}
	}

	for _, opt := range opts {
		if opt.Validate != nil {
			validated, err := opt.Validate(parsed)
			if err != nil {
				return nil, &StructuredOutputError{
					Message: fmt.Sprintf("JSON validation failed: %s", err.Error()),
					RawText: text,
				}
			}
			return validated, nil
		}
	}

	data, err := json.Marshal(parsed)
	if err != nil {
		return nil, &StructuredOutputError{
			Message: fmt.Sprintf("JSON re-marshal failed: %s", err.Error()),
			RawText: text,
		}
	}
	return data, nil
}

// Close closes the agent session.
func (a *Agent) Close() error {
	return a.sdk.Close()
}

// SetPlanMode enables or disables plan mode.
func (a *Agent) SetPlanMode(ctx context.Context, enabled bool) error {
	return a.sdk.SetPlanMode(ctx, enabled)
}

// EnablePlanMode enables plan mode.
func (a *Agent) EnablePlanMode(ctx context.Context) error {
	return a.sdk.EnablePlanMode(ctx)
}

// DisablePlanMode disables plan mode.
func (a *Agent) DisablePlanMode(ctx context.Context) error {
	return a.sdk.DisablePlanMode(ctx)
}

// Reset replaces the active conversation and returns the new session ID.
func (a *Agent) Reset(ctx context.Context) (*ResetResult, error) {
	return a.sdk.Reset(ctx)
}

// CreateBrowserHandoff creates a browser continuation token for the active session.
func (a *Agent) CreateBrowserHandoff(ctx context.Context, params *BrowserHandoffCreateParams) (*BrowserHandoffCreateResult, error) {
	return a.sdk.CreateBrowserHandoff(ctx, params)
}

// AttachBrowserHandoff restores the session referenced by a handoff token.
func (a *Agent) AttachBrowserHandoff(ctx context.Context, params *BrowserHandoffAttachParams) (*BrowserHandoffAttachResult, error) {
	return a.sdk.AttachBrowserHandoff(ctx, params)
}

// AttachLatestBrowserHandoff restores the newest unexpired browser handoff.
func (a *Agent) AttachLatestBrowserHandoff(ctx context.Context) (*BrowserHandoffAttachResult, error) {
	return a.sdk.AttachLatestBrowserHandoff(ctx)
}

// StartAutomode starts an auto-mode task and returns when the CLI accepts it.
func (a *Agent) StartAutomode(ctx context.Context, params *AutomodeStartParams) (*AutomodeStartResult, error) {
	return a.sdk.StartAutomode(ctx, params)
}

// GetAutomodeStatus returns live flags and optional persisted session state.
func (a *Agent) GetAutomodeStatus(ctx context.Context) (*AutomodeStatusResult, error) {
	return a.sdk.GetAutomodeStatus(ctx)
}

// PauseAutomode pauses the active auto-mode session.
func (a *Agent) PauseAutomode(ctx context.Context) (*AutomodePauseResult, error) {
	return a.sdk.PauseAutomode(ctx)
}

// ResumeAutomode resumes a paused auto-mode session.
func (a *Agent) ResumeAutomode(ctx context.Context) (*AutomodeResumeResult, error) {
	return a.sdk.ResumeAutomode(ctx)
}

// CancelAutomode cancels the active auto-mode session.
func (a *Agent) CancelAutomode(ctx context.Context, params *AutomodeCancelParams) (*AutomodeCancelResult, error) {
	return a.sdk.CancelAutomode(ctx, params)
}

// AllowPermission approves a permission request.
func (a *Agent) AllowPermission(ctx context.Context, requestID string, scope DecisionScope) error {
	return a.sdk.AllowPermission(ctx, requestID, scope)
}

// DenyPermission denies a permission request.
func (a *Agent) DenyPermission(ctx context.Context, requestID string, scope DecisionScope) error {
	return a.sdk.DenyPermission(ctx, requestID, scope)
}

// PermissionResponse responds to a permission request.
func (a *Agent) PermissionResponse(ctx context.Context, requestID string, decision PermissionDecision) error {
	return a.sdk.PermissionResponse(ctx, requestID, decision)
}

// Autoresearch starts a high-level slash-command run for the objective.
func (a *Agent) Autoresearch(ctx context.Context, objective string, opts *PromptParams) (*Run, error) {
	return a.Command(ctx, "/autoresearch", []string{objective}, opts)
}

// Command starts any validated slash command as a run.
func (a *Agent) Command(ctx context.Context, command string, args []string, opts *PromptParams) (*Run, error) {
	formatted, err := FormatSlashCommand(command, args...)
	if err != nil {
		return nil, err
	}
	return a.Send(ctx, formatted, opts)
}

func (a *Agent) DeepResearch(ctx context.Context, objective string, opts *PromptParams) (*Run, error) {
	return a.Command(ctx, "/deep-research", []string{objective}, opts)
}

func (a *Agent) GetGoal(ctx context.Context) (*GoalSnapshot, error) { return a.sdk.GetGoal(ctx) }
func (a *Agent) CreateGoal(ctx context.Context, p *GoalCreateParams) (*GoalMutationResult, error) {
	return a.sdk.CreateGoal(ctx, p)
}
func (a *Agent) UpdateGoal(ctx context.Context, p *GoalUpdateParams) (*GoalMutationResult, error) {
	return a.sdk.UpdateGoal(ctx, p)
}
func (a *Agent) QueueGoal(ctx context.Context, p *GoalCreateParams) (*GoalMutationResult, error) {
	return a.sdk.QueueGoal(ctx, p)
}
func (a *Agent) StartQueuedGoal(ctx context.Context) (*GoalMutationResult, error) {
	return a.sdk.StartQueuedGoal(ctx)
}
func (a *Agent) ListGoalTemplates(ctx context.Context) ([]GoalTemplateMetadata, error) {
	return a.sdk.ListGoalTemplates(ctx)
}
func (a *Agent) ClearGoal(ctx context.Context) (*GoalMutationResult, error) {
	return a.sdk.ClearGoal(ctx)
}

// StartAutoresearch initializes or resumes a persisted autoresearch loop.
func (a *Agent) StartAutoresearch(ctx context.Context, params *AutoresearchStartParams) (*AutoresearchStartResult, error) {
	return a.sdk.StartAutoresearch(ctx, params)
}

// GetAutoresearchStatus returns current persisted autoresearch state.
func (a *Agent) GetAutoresearchStatus(ctx context.Context) (*AutoresearchStatusResult, error) {
	return a.sdk.GetAutoresearchStatus(ctx)
}

// StopAutoresearch pauses autoresearch without deleting persisted state.
func (a *Agent) StopAutoresearch(ctx context.Context) (*AutoresearchStopResult, error) {
	return a.sdk.StopAutoresearch(ctx)
}

// GetAutoresearchHistory lists persisted attempts.
func (a *Agent) GetAutoresearchHistory(ctx context.Context) (*AutoresearchHistoryResult, error) {
	return a.sdk.GetAutoresearchHistory(ctx)
}

// ReplayAutoresearch re-evaluates a candidate in an isolated worktree.
func (a *Agent) ReplayAutoresearch(ctx context.Context, params *AutoresearchReplayParams) (*AutoresearchReplayResult, error) {
	return a.sdk.ReplayAutoresearch(ctx, params)
}

// RescoreAutoresearch reapplies current policy to persisted measurements.
func (a *Agent) RescoreAutoresearch(ctx context.Context, params *AutoresearchRescoreParams) (*AutoresearchRescoreResult, error) {
	return a.sdk.RescoreAutoresearch(ctx, params)
}

// CompareAutoresearch compares persisted evidence for two attempts.
func (a *Agent) CompareAutoresearch(ctx context.Context, params *AutoresearchCompareParams) (*AutoresearchCompareResult, error) {
	return a.sdk.CompareAutoresearch(ctx, params)
}

// GetAutoresearchPareto returns the current constraint-passing Pareto frontier.
func (a *Agent) GetAutoresearchPareto(ctx context.Context) (*AutoresearchParetoResult, error) {
	return a.sdk.GetAutoresearchPareto(ctx)
}

// PinAutoresearch pins or unpins a candidate's replay artifacts.
func (a *Agent) PinAutoresearch(ctx context.Context, params *AutoresearchPinParams) (*AutoresearchPinResult, error) {
	return a.sdk.PinAutoresearch(ctx, params)
}

// PruneAutoresearch previews or applies artifact retention.
func (a *Agent) PruneAutoresearch(ctx context.Context, params *AutoresearchPruneParams) (*AutoresearchPruneResult, error) {
	return a.sdk.PruneAutoresearch(ctx, params)
}

// GetSkillsRegistry returns the community skill registry.
func (a *Agent) GetSkillsRegistry(ctx context.Context, params *GetSkillsRegistryParams) (*GetSkillsRegistryResult, error) {
	return a.sdk.GetSkillsRegistry(ctx, params)
}

// InstallSkill installs a registry skill into user or project scope.
func (a *Agent) InstallSkill(ctx context.Context, params *InstallSkillParams) (*InstallSkillResult, error) {
	return a.sdk.InstallSkill(ctx, params)
}

// ListMCPServers returns all known MCP servers and their status.
func (a *Agent) ListMCPServers(ctx context.Context) (*MCPListServersResult, error) {
	return a.sdk.ListMCPServers(ctx)
}

// ListMCPTools returns available MCP tools, optionally filtered by server.
func (a *Agent) ListMCPTools(ctx context.Context, params *MCPListToolsParams) (*MCPListToolsResult, error) {
	return a.sdk.ListMCPTools(ctx, params)
}

// GetMCPServerConfigs returns configured MCP server definitions.
func (a *Agent) GetMCPServerConfigs(ctx context.Context) (*MCPGetServerConfigsResult, error) {
	return a.sdk.GetMCPServerConfigs(ctx)
}

// RunJson runs a prompt and returns the result parsed as JSON.
func (a *Agent) RunJson(ctx context.Context, input interface{}, opts *PromptParams, jsonOpts ...JsonParseOptions[json.RawMessage]) (json.RawMessage, error) {
	run, err := a.Send(ctx, input, opts)
	if err != nil {
		return nil, err
	}
	return run.JSON(jsonOpts...)
}

// StructuredOutputError is returned when JSON parsing or validation fails.
type StructuredOutputError struct {
	Message string
	RawText string
}

func (e *StructuredOutputError) Error() string {
	return e.Message
}

// parseJsonText attempts to extract and parse JSON from text.
// It tries direct parsing first, then fenced code blocks, then embedded JSON.
func parseJsonText(text string) (interface{}, error) {
	// Direct parse
	var direct interface{}
	if err := json.Unmarshal([]byte(text), &direct); err == nil {
		return direct, nil
	}

	// Fenced JSON block
	const fenceStart = "```json"
	const fenceEnd = "```"
	if start := strings.Index(text, fenceStart); start != -1 {
		start += len(fenceStart)
		if end := strings.Index(text[start:], fenceEnd); end != -1 {
			candidate := strings.TrimSpace(text[start : start+end])
			var fenced interface{}
			if err := json.Unmarshal([]byte(candidate), &fenced); err == nil {
				return fenced, nil
			}
		}
	}

	// Any fenced code block
	if start := strings.Index(text, "```"); start != -1 {
		start += len("```")
		if end := strings.Index(text[start:], "```"); end != -1 {
			candidate := strings.TrimSpace(text[start : start+end])
			var fenced interface{}
			if err := json.Unmarshal([]byte(candidate), &fenced); err == nil {
				return fenced, nil
			}
		}
	}

	// Try to find a JSON object or array substring
	objStart := strings.Index(text, "{")
	arrStart := strings.Index(text, "[")
	if objStart != -1 && (arrStart == -1 || objStart < arrStart) {
		// Find matching closing brace
		depth := 0
		for i := objStart; i < len(text); i++ {
			switch text[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					candidate := strings.TrimSpace(text[objStart : i+1])
					var obj interface{}
					if err := json.Unmarshal([]byte(candidate), &obj); err == nil {
						return obj, nil
					}
					break
				}
			}
		}
	}
	if arrStart != -1 {
		depth := 0
		for i := arrStart; i < len(text); i++ {
			switch text[i] {
			case '[':
				depth++
			case ']':
				depth--
				if depth == 0 {
					candidate := strings.TrimSpace(text[arrStart : i+1])
					var arr interface{}
					if err := json.Unmarshal([]byte(candidate), &arr); err == nil {
						return arr, nil
					}
					break
				}
			}
		}
	}

	return nil, fmt.Errorf("no valid JSON found in text")
}

// withJsonInstruction prepends JSON instructions to a message.
func withJsonInstruction(message string, opts JsonRunOptions[json.RawMessage]) string {
	instructions := buildJsonInstruction(opts)
	if instructions == "" {
		return message
	}
	return instructions + "\n\n" + message
}

// buildJsonInstruction builds the JSON instruction text from options.
func buildJsonInstruction(opts JsonRunOptions[json.RawMessage]) string {
	if opts.OutputInstructions != "" {
		return opts.OutputInstructions
	}
	parts := []string{"Return only valid JSON. Do not wrap the response in Markdown."}
	if opts.SchemaName != "" {
		parts = append(parts, fmt.Sprintf("The JSON value should satisfy: %s.", opts.SchemaName))
	}
	if opts.Schema != nil {
		schemaJSON, err := json.MarshalIndent(opts.Schema, "", "  ")
		if err == nil {
			parts = append(parts, fmt.Sprintf("Use this JSON schema or example shape:\n%s", string(schemaJSON)))
		}
	}
	return strings.Join(parts, "\n")
}

func toPromptParams(input interface{}, opts *PromptParams) *PromptParams {
	switch v := input.(type) {
	case string:
		params := &PromptParams{Message: v}
		mergePromptOptions(params, opts)
		return params
	case []string:
		params := &PromptParams{Message: strings.Join(v, "\n")}
		mergePromptOptions(params, opts)
		return params
	case *PromptParams:
		if opts == nil {
			return v
		}
		params := *v
		mergePromptOptions(&params, opts)
		return &params
	case AgentOptions:
		msg := v.Instructions
		if opts != nil && opts.Message != "" {
			msg = opts.Message
		}
		params := &PromptParams{Message: msg}
		mergePromptOptions(params, opts)
		return params
	default:
		return &PromptParams{Message: fmt.Sprintf("%v", v)}
	}
}

func mergePromptOptions(params, opts *PromptParams) {
	if opts == nil {
		return
	}
	if opts.Context != nil {
		params.Context = opts.Context
	}
	if len(opts.Images) > 0 {
		params.Images = opts.Images
	}
	if opts.ThinkingLevel != "" {
		params.ThinkingLevel = opts.ThinkingLevel
	}
	if opts.AgentsMd != nil {
		params.AgentsMd = opts.AgentsMd
	}
}
