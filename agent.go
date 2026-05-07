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

	mu         sync.Mutex
	events     []Event
	waiters    []chan struct{}
	completed  bool
	aborted    bool
	runErr     error
	text       string
	started    bool
	startOnce  sync.Once
	resultCh   chan RunResult
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
