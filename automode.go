package autohand

// AutomodeStartParams configures an auto-mode task.
type AutomodeStartParams struct {
	Prompt             string   `json:"prompt"`
	MaxIterations      *int     `json:"maxIterations,omitempty"`
	CompletionPromise  *string  `json:"completionPromise,omitempty"`
	UseWorktree        *bool    `json:"useWorktree,omitempty"`
	CheckpointInterval *int     `json:"checkpointInterval,omitempty"`
	MaxRuntime         *int     `json:"maxRuntime,omitempty"`
	MaxCost            *float64 `json:"maxCost,omitempty"`
}

// AutomodeStartResult reports whether the CLI accepted an auto-mode session.
type AutomodeStartResult struct {
	Success   bool    `json:"success"`
	SessionID *string `json:"sessionId,omitempty"`
	Error     *string `json:"error,omitempty"`
}
