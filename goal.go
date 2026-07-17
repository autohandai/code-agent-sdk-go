package autohand

// GoalStatus is the lifecycle state of a persistent goal.
type GoalStatus string

const (
	GoalStatusActive        GoalStatus = "active"
	GoalStatusPaused        GoalStatus = "paused"
	GoalStatusBudgetLimited GoalStatus = "budgetLimited"
	GoalStatusComplete      GoalStatus = "complete"
)

type GoalState struct {
	GoalID                     string     `json:"goalId"`
	Objective                  string     `json:"objective"`
	Status                     GoalStatus `json:"status"`
	TokenBudget                *int       `json:"tokenBudget,omitempty"`
	TimeBudgetSeconds          *int       `json:"timeBudgetSeconds,omitempty"`
	MinTokensBeforeWrapUp      *int       `json:"minTokensBeforeWrapUp,omitempty"`
	MinTimeSecondsBeforeWrapUp *int       `json:"minTimeSecondsBeforeWrapUp,omitempty"`
	TokensUsed                 int        `json:"tokensUsed"`
	TimeUsedSeconds            int        `json:"timeUsedSeconds"`
	CreatedAt                  string     `json:"createdAt"`
	UpdatedAt                  string     `json:"updatedAt"`
}

type QueuedGoal struct {
	QueueID                    string   `json:"queueId"`
	Objective                  string   `json:"objective"`
	TokenBudget                *int     `json:"tokenBudget,omitempty"`
	TimeBudgetSeconds          *int     `json:"timeBudgetSeconds,omitempty"`
	MinTokensBeforeWrapUp      *int     `json:"minTokensBeforeWrapUp,omitempty"`
	MinTimeSecondsBeforeWrapUp *int     `json:"minTimeSecondsBeforeWrapUp,omitempty"`
	Source                     string   `json:"source"`
	Template                   string   `json:"template,omitempty"`
	TemplateFlags              []string `json:"templateFlags,omitempty"`
	TemplateArgs               []string `json:"templateArgs,omitempty"`
	CreatedAt                  string   `json:"createdAt"`
}

type CompletedGoal struct {
	GoalID          string     `json:"goalId"`
	Objective       string     `json:"objective"`
	Status          GoalStatus `json:"status"`
	TokensUsed      int        `json:"tokensUsed"`
	TimeUsedSeconds int        `json:"timeUsedSeconds"`
	CreatedAt       string     `json:"createdAt"`
	CompletedAt     string     `json:"completedAt"`
}

type GoalSnapshot struct {
	Version   int             `json:"version"`
	Goal      *GoalState      `json:"goal"`
	Queue     []QueuedGoal    `json:"queue"`
	Completed []CompletedGoal `json:"completed"`
	UpdatedAt string          `json:"updatedAt"`
}

type GoalTemplateMetadata struct {
	Name                 string   `json:"name"`
	Path                 string   `json:"path"`
	Description          string   `json:"description,omitempty"`
	Aliases              []string `json:"aliases"`
	AllowCommands        []string `json:"allowCommands"`
	RequiredPlaceholders []string `json:"requiredPlaceholders"`
	RequiredFlags        []string `json:"requiredFlags"`
	RequiresArgs         bool     `json:"requiresArgs"`
}

type GoalCreateParams struct {
	Objective                  string `json:"objective"`
	TokenBudget                *int   `json:"token_budget,omitempty"`
	TimeBudgetSeconds          *int   `json:"time_budget_seconds,omitempty"`
	MinTokensBeforeWrapUp      *int   `json:"min_tokens_before_wrap_up,omitempty"`
	MinTimeSecondsBeforeWrapUp *int   `json:"min_time_seconds_before_wrap_up,omitempty"`
}

// GoalUpdateParams uses pointer-to-pointer budgets so callers can distinguish
// omitted fields from explicit null, which clears a persisted budget.
type GoalUpdateParams struct {
	Objective                  *string     `json:"objective,omitempty"`
	Status                     *GoalStatus `json:"status,omitempty"`
	TokenBudget                **int       `json:"token_budget,omitempty"`
	TimeBudgetSeconds          **int       `json:"time_budget_seconds,omitempty"`
	MinTokensBeforeWrapUp      **int       `json:"min_tokens_before_wrap_up,omitempty"`
	MinTimeSecondsBeforeWrapUp **int       `json:"min_time_seconds_before_wrap_up,omitempty"`
}

type GoalTelemetry struct {
	TimeRemainingSeconds *int `json:"timeRemainingSeconds,omitempty"`
	TokensRemaining      *int `json:"tokensRemaining,omitempty"`
	CompletionFloorMet   bool `json:"completionFloorMet"`
}

type GoalMutationResult struct {
	OK           bool           `json:"ok"`
	Goal         *GoalState     `json:"goal"`
	Queue        []QueuedGoal   `json:"queue"`
	Telemetry    *GoalTelemetry `json:"telemetry,omitempty"`
	Message      string         `json:"message,omitempty"`
	Queued       *QueuedGoal    `json:"queued,omitempty"`
	Started      *GoalState     `json:"started,omitempty"`
	Completed    *CompletedGoal `json:"completed,omitempty"`
	CompletedRun *CompletedGoal `json:"completedRun,omitempty"`
	Dequeued     *QueuedGoal    `json:"dequeued,omitempty"`
	Removed      []QueuedGoal   `json:"removed,omitempty"`
}

const GoalWrittenCompletedHook HookEvent = "goal-written:completed"
