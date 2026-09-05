package autohand

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// AgentStep is a completed tool step whose results have been persisted by the CLI.
type AgentStep struct {
	StepNumber  int                   `json:"stepNumber"`
	Thought     string                `json:"thought,omitempty"`
	ToolCalls   []AgentStepToolCall   `json:"toolCalls"`
	ToolResults []AgentStepToolResult `json:"toolResults"`
}

// AgentStepToolCall describes a tool invocation in a completed step.
type AgentStepToolCall struct {
	ID   string                     `json:"id,omitempty"`
	Tool string                     `json:"tool"`
	Args map[string]json.RawMessage `json:"args"`
}

// AgentStepToolResult describes the persisted outcome of a tool invocation.
type AgentStepToolResult struct {
	Tool    string `json:"tool"`
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

// StopCondition decides whether to pause after a completed step. Honor ctx when
// performing asynchronous work; returning an error stops before surfacing it.
type StopCondition func(ctx context.Context, steps []AgentStep) (bool, error)

// IsStepCount stops after count completed tool steps within the current prompt.
func IsStepCount(count int) (StopCondition, error) {
	if count < 1 {
		return nil, fmt.Errorf("step count must be positive")
	}
	return func(_ context.Context, steps []AgentStep) (bool, error) { return len(steps) >= count, nil }, nil
}

// HasToolCall stops when a completed step includes the named tool.
func HasToolCall(tool string) (StopCondition, error) {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return nil, fmt.Errorf("tool name must not be empty")
	}
	return func(_ context.Context, steps []AgentStep) (bool, error) {
		if len(steps) > 0 {
			for _, call := range steps[len(steps)-1].ToolCalls {
				if call.Tool == tool {
					return true, nil
				}
			}
		}
		return false, nil
	}, nil
}

// StepEndEvent exposes the CLI's host decision boundary.
type StepEndEvent struct {
	Type      string    `json:"type"`
	StepID    string    `json:"stepId"`
	Step      AgentStep `json:"step"`
	Timestamp string    `json:"timestamp"`
}

func (e StepEndEvent) eventType() string { return "step_end" }

// MarshalJSON keeps host callbacks out of the wire payload.
func (p PromptParams) MarshalJSON() ([]byte, error) {
	type wirePrompt PromptParams
	var stopWhen *struct {
		Mode string `json:"mode"`
	}
	if len(p.StopWhen) > 0 {
		stopWhen = &struct {
			Mode string `json:"mode"`
		}{Mode: "host"}
	}
	return json.Marshal(struct {
		wirePrompt
		StopWhen *struct {
			Mode string `json:"mode"`
		} `json:"stopWhen,omitempty"`
	}{wirePrompt(p), stopWhen})
}

// StepDecision acknowledges a completed step when driving RPCClient directly.
func (c *RPCClient) StepDecision(ctx context.Context, stepID string, stop bool) error {
	if strings.TrimSpace(stepID) == "" {
		return fmt.Errorf("step ID must not be empty")
	}
	result, err := rpcRequest[struct {
		Success *bool `json:"success"`
	}](ctx, c, "autohand.stepDecision", struct {
		StepID string `json:"stepId"`
		Stop   bool   `json:"stop"`
	}{stepID, stop})
	if err != nil {
		return err
	}
	if result.Success == nil || !*result.Success {
		return fmt.Errorf("CLI rejected step decision for %s", stepID)
	}
	return nil
}

func parseStepEnd(params json.RawMessage) (StepEndEvent, error) {
	var event StepEndEvent
	if err := json.Unmarshal(params, &event); err != nil {
		return event, err
	}
	if event.StepID == "" || event.Timestamp == "" || event.Step.StepNumber < 1 || event.Step.ToolCalls == nil || event.Step.ToolResults == nil {
		return event, fmt.Errorf("invalid step envelope")
	}
	for _, call := range event.Step.ToolCalls {
		if call.Tool == "" || call.Args == nil {
			return event, fmt.Errorf("invalid step tool call")
		}
	}
	var payload struct {
		Step struct {
			ToolResults []struct {
				Success *bool `json:"success"`
			} `json:"toolResults"`
		} `json:"step"`
	}
	if err := json.Unmarshal(params, &payload); err != nil {
		return event, err
	}
	for i, result := range event.Step.ToolResults {
		if result.Tool == "" || payload.Step.ToolResults[i].Success == nil {
			return event, fmt.Errorf("invalid step tool result")
		}
	}
	event.Type = "step_end"
	return event, nil
}
