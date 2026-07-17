package autohand

import "fmt"

// AutoresearchOptimizationDirection controls whether lower or higher metric values improve.
type AutoresearchOptimizationDirection string

const (
	AutoresearchLower  AutoresearchOptimizationDirection = "lower"
	AutoresearchHigher AutoresearchOptimizationDirection = "higher"
)

// AutoresearchSubagentOptions controls optional subagent participation.
type AutoresearchSubagentOptions struct {
	IdeaGeneration      *bool `json:"ideaGeneration,omitempty"`
	MeasurementAnalysis *bool `json:"measurementAnalysis,omitempty"`
	Finalization        *bool `json:"finalization,omitempty"`
}

// AutoresearchSecondaryObjective defines an additional optimization metric.
type AutoresearchSecondaryObjective struct {
	Name      string                            `json:"name"`
	Unit      string                            `json:"unit"`
	Direction AutoresearchOptimizationDirection `json:"direction"`
}

// AutoresearchConstraintOperator is a supported numeric constraint comparison.
type AutoresearchConstraintOperator string

const (
	AutoresearchConstraintLessThan           AutoresearchConstraintOperator = "<"
	AutoresearchConstraintLessThanOrEqual    AutoresearchConstraintOperator = "<="
	AutoresearchConstraintGreaterThan        AutoresearchConstraintOperator = ">"
	AutoresearchConstraintGreaterThanOrEqual AutoresearchConstraintOperator = ">="
)

// AutoresearchConstraint defines a metric threshold that candidates must satisfy.
type AutoresearchConstraint struct {
	MetricName string                         `json:"metricName"`
	Operator   AutoresearchConstraintOperator `json:"operator"`
	Threshold  float64                        `json:"threshold"`
}

// AutoresearchSamplingOptions controls adaptive benchmark sampling.
type AutoresearchSamplingOptions struct {
	MinSamples          int     `json:"minSamples,omitempty"`
	MaxSamples          int     `json:"maxSamples,omitempty"`
	ConfidenceThreshold float64 `json:"confidenceThreshold,omitempty"`
}

// AutoresearchRetentionOptions controls replay-artifact retention.
type AutoresearchRetentionOptions struct {
	MaxArtifactBytes   int64 `json:"maxArtifactBytes,omitempty"`
	MaxArtifactAgeDays int   `json:"maxArtifactAgeDays,omitempty"`
}

// AutoresearchStartParams configures and starts or resumes a persisted experiment loop.
type AutoresearchStartParams struct {
	Objective            string                            `json:"objective"`
	MaxIterations        int                               `json:"maxIterations,omitempty"`
	TimeoutMS            int                               `json:"timeoutMs,omitempty"`
	MetricName           string                            `json:"metricName,omitempty"`
	MetricUnit           string                            `json:"metricUnit,omitempty"`
	Direction            AutoresearchOptimizationDirection `json:"direction,omitempty"`
	MeasureCommand       string                            `json:"measureCommand,omitempty"`
	MeasureScript        string                            `json:"measureScript,omitempty"`
	ChecksCommand        string                            `json:"checksCommand,omitempty"`
	ChecksScript         string                            `json:"checksScript,omitempty"`
	FilesInScope         []string                          `json:"filesInScope,omitempty"`
	Subagents            *AutoresearchSubagentOptions      `json:"subagents,omitempty"`
	SecondaryObjectives  []AutoresearchSecondaryObjective  `json:"secondaryObjectives,omitempty"`
	Constraints          []AutoresearchConstraint          `json:"constraints,omitempty"`
	Sampling             *AutoresearchSamplingOptions      `json:"sampling,omitempty"`
	Retention            *AutoresearchRetentionOptions     `json:"retention,omitempty"`
	EnvironmentAllowlist []string                          `json:"environmentAllowlist,omitempty"`
}

// AutoresearchMetricAggregate summarizes repeated measurements robustly.
type AutoresearchMetricAggregate struct {
	Median      float64 `json:"median"`
	MAD         float64 `json:"mad"`
	SampleCount int     `json:"sampleCount"`
}

// AutoresearchEvaluationSample is one benchmark observation.
type AutoresearchEvaluationSample struct {
	Sequence     int                `json:"sequence"`
	Metrics      map[string]float64 `json:"metrics"`
	OutputObject string             `json:"outputObject"`
	DurationMS   int64              `json:"durationMs"`
	Timestamp    string             `json:"timestamp"`
}

// AutoresearchChecksResult records the candidate validation command outcome.
type AutoresearchChecksResult struct {
	Passed       bool   `json:"passed"`
	OutputObject string `json:"outputObject,omitempty"`
}

// AutoresearchExecutionResult records how an evaluation ended.
type AutoresearchExecutionResult struct {
	Outcome      AutoresearchExecutionOutcome `json:"outcome"`
	Error        string                       `json:"error,omitempty"`
	OutputObject string                       `json:"outputObject,omitempty"`
}

// AutoresearchExecutionOutcome identifies how an evaluation ended.
type AutoresearchExecutionOutcome string

const (
	AutoresearchExecutionPassed          AutoresearchExecutionOutcome = "passed"
	AutoresearchExecutionBenchmarkFailed AutoresearchExecutionOutcome = "benchmark_failed"
	AutoresearchExecutionChecksFailed    AutoresearchExecutionOutcome = "checks_failed"
	AutoresearchExecutionCancelled       AutoresearchExecutionOutcome = "cancelled"
)

// AutoresearchEvaluationRecord is an immutable persisted benchmark evaluation.
type AutoresearchEvaluationRecord struct {
	SchemaVersion int                                    `json:"schemaVersion"`
	Type          string                                 `json:"type"`
	ID            string                                 `json:"id"`
	AttemptID     string                                 `json:"attemptId"`
	Timestamp     string                                 `json:"timestamp"`
	Context       map[string]interface{}                 `json:"context"`
	EvaluatorMode AutoresearchEvaluatorMode              `json:"evaluatorMode"`
	Samples       []AutoresearchEvaluationSample         `json:"samples"`
	Aggregates    map[string]AutoresearchMetricAggregate `json:"aggregates"`
	Checks        AutoresearchChecksResult               `json:"checks"`
	Execution     AutoresearchExecutionResult            `json:"execution"`
	DriftWarnings []string                               `json:"driftWarnings"`
}

// AutoresearchConstraintResult is a constraint evaluation with conservative confidence bounds.
type AutoresearchConstraintResult struct {
	AutoresearchConstraint
	ConservativeValue float64 `json:"conservativeValue"`
	Passed            bool    `json:"passed"`
	Conclusive        bool    `json:"conclusive"`
}

// AutoresearchDecisionRecord is an immutable policy decision over an evaluation.
type AutoresearchDecisionRecord struct {
	SchemaVersion      int                            `json:"schemaVersion"`
	Type               string                         `json:"type"`
	ID                 string                         `json:"id"`
	AttemptID          string                         `json:"attemptId"`
	Timestamp          string                         `json:"timestamp"`
	Context            map[string]interface{}         `json:"context"`
	PolicyVersion      string                         `json:"policyVersion"`
	EvaluationID       string                         `json:"evaluationId"`
	Source             AutoresearchDecisionSource     `json:"source"`
	ConstraintResults  []AutoresearchConstraintResult `json:"constraintResults"`
	PrimaryImprovement float64                        `json:"primaryImprovement"`
	Confidence         float64                        `json:"confidence"`
	Outcome            AutoresearchDecisionOutcome    `json:"outcome"`
	Materialized       bool                           `json:"materialized"`
	Explanation        string                         `json:"explanation"`
}

// AutoresearchDecisionSource identifies the evidence path for a decision.
type AutoresearchDecisionSource string

const (
	AutoresearchDecisionOriginal AutoresearchDecisionSource = "original"
	AutoresearchDecisionReplay   AutoresearchDecisionSource = "replay"
	AutoresearchDecisionRescore  AutoresearchDecisionSource = "rescore"
)

// AutoresearchDecisionOutcome is the current policy's candidate disposition.
type AutoresearchDecisionOutcome string

const (
	AutoresearchDecisionAccepted     AutoresearchDecisionOutcome = "accepted"
	AutoresearchDecisionRejected     AutoresearchDecisionOutcome = "rejected"
	AutoresearchDecisionInconclusive AutoresearchDecisionOutcome = "inconclusive"
	AutoresearchDecisionChecksFailed AutoresearchDecisionOutcome = "checks_failed"
	AutoresearchDecisionCrashed      AutoresearchDecisionOutcome = "crashed"
)

// AutoresearchMaterializationState describes the retained state of an attempt.
type AutoresearchMaterializationState string

const (
	AutoresearchMaterializationBaseline  AutoresearchMaterializationState = "baseline"
	AutoresearchMaterializationCommitted AutoresearchMaterializationState = "committed"
	AutoresearchMaterializationRetained  AutoresearchMaterializationState = "retained"
	AutoresearchMaterializationReverted  AutoresearchMaterializationState = "reverted"
	AutoresearchMaterializationNone      AutoresearchMaterializationState = "none"
)

// AutoresearchHistoryAttempt describes one persisted candidate.
type AutoresearchHistoryAttempt struct {
	AttemptID        string                           `json:"attemptId"`
	Description      string                           `json:"description"`
	Timestamp        string                           `json:"timestamp"`
	Legacy           bool                             `json:"legacy"`
	Replayable       bool                             `json:"replayable"`
	Pinned           bool                             `json:"pinned"`
	LatestEvaluation *AutoresearchEvaluationRecord    `json:"latestEvaluation,omitempty"`
	LatestDecision   *AutoresearchDecisionRecord      `json:"latestDecision,omitempty"`
	Materialization  AutoresearchMaterializationState `json:"materialization"`
}

// AutoresearchState is the persisted loop state.
type AutoresearchState struct {
	Active        bool   `json:"active"`
	Goal          string `json:"goal"`
	Iteration     int    `json:"iteration"`
	MaxIterations int    `json:"maxIterations"`
}

// AutoresearchStartResult is returned when a loop is initialized or resumed.
type AutoresearchStartResult struct {
	Success          bool                         `json:"success"`
	Message          string                       `json:"message,omitempty"`
	Instruction      string                       `json:"instruction,omitempty"`
	Active           *bool                        `json:"active,omitempty"`
	State            *AutoresearchState           `json:"state,omitempty"`
	StatusText       string                       `json:"statusText,omitempty"`
	RunsLogged       int                          `json:"runsLogged,omitempty"`
	Attempts         []AutoresearchHistoryAttempt `json:"attempts,omitempty"`
	ParetoAttemptIDs []string                     `json:"paretoAttemptIds,omitempty"`
	Error            string                       `json:"error,omitempty"`
}

// AutoresearchStatusResult reports persisted loop progress.
type AutoresearchStatusResult struct {
	Success          bool                         `json:"success"`
	Active           bool                         `json:"active"`
	State            *AutoresearchState           `json:"state,omitempty"`
	StatusText       string                       `json:"statusText"`
	RunsLogged       int                          `json:"runsLogged"`
	Attempts         []AutoresearchHistoryAttempt `json:"attempts,omitempty"`
	ParetoAttemptIDs []string                     `json:"paretoAttemptIds,omitempty"`
	Error            string                       `json:"error,omitempty"`
}

// AutoresearchStopResult reports the paused persisted loop.
type AutoresearchStopResult struct {
	Success          bool                         `json:"success"`
	Message          string                       `json:"message,omitempty"`
	Active           *bool                        `json:"active,omitempty"`
	State            *AutoresearchState           `json:"state,omitempty"`
	StatusText       string                       `json:"statusText,omitempty"`
	RunsLogged       int                          `json:"runsLogged,omitempty"`
	Attempts         []AutoresearchHistoryAttempt `json:"attempts,omitempty"`
	ParetoAttemptIDs []string                     `json:"paretoAttemptIds,omitempty"`
	Error            string                       `json:"error,omitempty"`
}

// AutoresearchHistoryResult lists persisted attempts.
type AutoresearchHistoryResult struct {
	Success  bool                         `json:"success"`
	Attempts []AutoresearchHistoryAttempt `json:"attempts"`
	Error    string                       `json:"error,omitempty"`
}

// AutoresearchEvaluatorMode selects the original or current evaluator.
type AutoresearchEvaluatorMode string

const (
	AutoresearchEvaluatorOriginal AutoresearchEvaluatorMode = "original"
	AutoresearchEvaluatorCurrent  AutoresearchEvaluatorMode = "current"
)

// AutoresearchReplayParams selects a candidate and evaluator for isolated replay.
type AutoresearchReplayParams struct {
	AttemptID string                    `json:"attemptId"`
	Evaluator AutoresearchEvaluatorMode `json:"evaluator,omitempty"`
}

// AutoresearchReplayResult reports replay measurements and a new decision.
type AutoresearchReplayResult struct {
	Success       bool                           `json:"success"`
	AttemptID     string                         `json:"attemptId,omitempty"`
	EvaluatorMode AutoresearchEvaluatorMode      `json:"evaluatorMode,omitempty"`
	Metrics       map[string]float64             `json:"metrics,omitempty"`
	Samples       []AutoresearchEvaluationSample `json:"samples,omitempty"`
	Decision      *AutoresearchDecisionRecord    `json:"decision,omitempty"`
	DriftWarnings []string                       `json:"driftWarnings,omitempty"`
	Error         string                         `json:"error,omitempty"`
}

// AutoresearchRescoreParams selects exactly one attempt or all attempts.
type AutoresearchRescoreParams struct {
	AttemptID string `json:"attemptId,omitempty"`
	All       bool   `json:"all,omitempty"`
}

// AutoresearchRescoreAttempt selects one attempt for rescoring.
func AutoresearchRescoreAttempt(attemptID string) *AutoresearchRescoreParams {
	return &AutoresearchRescoreParams{AttemptID: attemptID}
}

// AutoresearchRescoreAll selects every attempt for rescoring.
func AutoresearchRescoreAll() *AutoresearchRescoreParams {
	return &AutoresearchRescoreParams{All: true}
}

// Validate rejects ambiguous or empty rescore selections.
func (p *AutoresearchRescoreParams) Validate() error {
	if p == nil {
		return fmt.Errorf("autoresearch rescore requires exactly one of attemptId or all=true")
	}
	hasAttempt := p.AttemptID != ""
	if hasAttempt == p.All {
		return fmt.Errorf("autoresearch rescore requires exactly one of attemptId or all=true")
	}
	return nil
}

// AutoresearchRescoreResult reports decisions appended by current policy.
type AutoresearchRescoreResult struct {
	Success   bool                         `json:"success"`
	Decisions []AutoresearchDecisionRecord `json:"decisions"`
	Error     string                       `json:"error,omitempty"`
}

// AutoresearchCompareParams selects two attempts to compare.
type AutoresearchCompareParams struct {
	LeftAttemptID  string `json:"leftAttemptId"`
	RightAttemptID string `json:"rightAttemptId"`
}

// AutoresearchComparisonSide contains persisted evidence for one attempt.
type AutoresearchComparisonSide struct {
	AttemptID  string                                 `json:"attemptId"`
	Samples    []AutoresearchEvaluationSample         `json:"samples"`
	Aggregates map[string]AutoresearchMetricAggregate `json:"aggregates"`
	Checks     AutoresearchChecksResult               `json:"checks"`
	Execution  AutoresearchExecutionResult            `json:"execution"`
	Decision   *AutoresearchDecisionRecord            `json:"decision,omitempty"`
}

// AutoresearchComparison contains the left and right attempt evidence.
type AutoresearchComparison struct {
	Left  AutoresearchComparisonSide `json:"left"`
	Right AutoresearchComparisonSide `json:"right"`
}

// AutoresearchCompareResult reports a candidate comparison.
type AutoresearchCompareResult struct {
	Success    bool                    `json:"success"`
	Comparison *AutoresearchComparison `json:"comparison,omitempty"`
	Error      string                  `json:"error,omitempty"`
}

// AutoresearchParetoResult lists the current constraint-passing frontier.
type AutoresearchParetoResult struct {
	Success    bool     `json:"success"`
	AttemptIDs []string `json:"attemptIds"`
	Error      string   `json:"error,omitempty"`
}

// AutoresearchPinParams pins or unpins an attempt's artifacts.
type AutoresearchPinParams struct {
	AttemptID string `json:"attemptId"`
	Pinned    bool   `json:"pinned"`
}

// AutoresearchPinResult reports the updated pin state.
type AutoresearchPinResult struct {
	Success   bool   `json:"success"`
	AttemptID string `json:"attemptId"`
	Pinned    bool   `json:"pinned"`
	Error     string `json:"error,omitempty"`
}

// AutoresearchPruneParams previews pruning unless DryRun=false and Yes=true.
type AutoresearchPruneParams struct {
	DryRun *bool `json:"dryRun,omitempty"`
	Yes    bool  `json:"yes,omitempty"`
}

// AutoresearchPruneCandidate describes artifacts eligible for pruning.
type AutoresearchPruneCandidate struct {
	AttemptID string   `json:"attemptId"`
	Objects   []string `json:"objects"`
	Bytes     int64    `json:"bytes"`
	Protected bool     `json:"protected"`
	Reason    string   `json:"reason"`
}

// AutoresearchPruneResult reports a preview or applied retention operation.
type AutoresearchPruneResult struct {
	Success        bool                         `json:"success"`
	Applied        bool                         `json:"applied"`
	Candidates     []AutoresearchPruneCandidate `json:"candidates"`
	BytesFreed     int64                        `json:"bytesFreed"`
	RemainingBytes int64                        `json:"remainingBytes"`
	Error          string                       `json:"error,omitempty"`
}

// AutoresearchPhase identifies a lifecycle notification.
type AutoresearchPhase string

const (
	AutoresearchPhaseStart  AutoresearchPhase = "start"
	AutoresearchPhaseStatus AutoresearchPhase = "status"
	AutoresearchPhasePause  AutoresearchPhase = "pause"
)

// AutoresearchLifecycleEvent reports start, status, and pause notifications.
type AutoresearchLifecycleEvent struct {
	Type          string                 `json:"type"`
	Phase         AutoresearchPhase      `json:"phase"`
	Active        bool                   `json:"active"`
	Goal          string                 `json:"goal,omitempty"`
	Iteration     int                    `json:"iteration,omitempty"`
	MaxIterations int                    `json:"maxIterations,omitempty"`
	RunsLogged    int                    `json:"runsLogged"`
	StatusText    string                 `json:"statusText"`
	Subcommand    AutoresearchSubcommand `json:"subcommand"`
	Message       string                 `json:"message,omitempty"`
	Timestamp     string                 `json:"timestamp"`
}

// AutoresearchSubcommand identifies the lifecycle command that emitted an event.
type AutoresearchSubcommand string

const (
	AutoresearchSubcommandStart  AutoresearchSubcommand = "start"
	AutoresearchSubcommandResume AutoresearchSubcommand = "resume"
	AutoresearchSubcommandStatus AutoresearchSubcommand = "status"
	AutoresearchSubcommandStop   AutoresearchSubcommand = "stop"
)

func (e AutoresearchLifecycleEvent) eventType() string { return "autoresearch" }

// AutoresearchOperation identifies a replayable-ledger operation.
type AutoresearchOperation string

const (
	AutoresearchOperationHistory AutoresearchOperation = "history"
	AutoresearchOperationReplay  AutoresearchOperation = "replay"
	AutoresearchOperationRescore AutoresearchOperation = "rescore"
	AutoresearchOperationCompare AutoresearchOperation = "compare"
	AutoresearchOperationPareto  AutoresearchOperation = "pareto"
	AutoresearchOperationPin     AutoresearchOperation = "pin"
	AutoresearchOperationPrune   AutoresearchOperation = "prune"
)

// AutoresearchOperationEvent reports ledger operation progress.
type AutoresearchOperationEvent struct {
	Type      string                     `json:"type"`
	Operation AutoresearchOperation      `json:"operation"`
	Phase     AutoresearchOperationPhase `json:"phase"`
	AttemptID string                     `json:"attemptId,omitempty"`
	Success   bool                       `json:"success"`
	Applied   *bool                      `json:"applied,omitempty"`
	Error     string                     `json:"error,omitempty"`
	Timestamp string                     `json:"timestamp"`
}

// AutoresearchOperationPhase identifies operation progress.
type AutoresearchOperationPhase string

const (
	AutoresearchOperationStarted   AutoresearchOperationPhase = "started"
	AutoresearchOperationCompleted AutoresearchOperationPhase = "completed"
	AutoresearchOperationFailed    AutoresearchOperationPhase = "failed"
)

func (e AutoresearchOperationEvent) eventType() string { return "autoresearch" }
