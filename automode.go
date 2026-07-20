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

// AutomodeStatus is the persisted lifecycle status of an auto-mode session.
type AutomodeStatus string

const (
	AutomodeStatusRunning   AutomodeStatus = "running"
	AutomodeStatusPaused    AutomodeStatus = "paused"
	AutomodeStatusCompleted AutomodeStatus = "completed"
	AutomodeStatusCancelled AutomodeStatus = "cancelled"
	AutomodeStatusFailed    AutomodeStatus = "failed"
)

// AutomodeCheckpoint describes the latest persisted checkpoint.
type AutomodeCheckpoint struct {
	Commit    string `json:"commit"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// AutomodeState is the persisted session state returned by the CLI.
type AutomodeState struct {
	SessionID        string              `json:"sessionId"`
	Status           AutomodeStatus      `json:"status"`
	CurrentIteration int                 `json:"currentIteration"`
	MaxIterations    int                 `json:"maxIterations"`
	FilesCreated     int                 `json:"filesCreated"`
	FilesModified    int                 `json:"filesModified"`
	Branch           *string             `json:"branch,omitempty"`
	LastCheckpoint   *AutomodeCheckpoint `json:"lastCheckpoint,omitempty"`
}

// AutomodeStatusResult combines live flags with optional persisted state.
type AutomodeStatusResult struct {
	Active bool           `json:"active"`
	Paused bool           `json:"paused"`
	State  *AutomodeState `json:"state,omitempty"`
}

// AutomodePauseResult reports the business result of a pause request.
type AutomodePauseResult struct {
	Success bool    `json:"success"`
	Error   *string `json:"error,omitempty"`
}
