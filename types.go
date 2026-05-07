// Package autohand provides a Go SDK for interacting with the Autohand CLI
// through JSON-RPC 2.0 over stdio. It supports streaming events, permission
// management, model switching, and full lifecycle control of agent sessions.
package autohand

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// JSON-RPC 2.0 types
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      int         `json:"id"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
	ID      int             `json:"id"`
}

type jsonRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type jsonRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *jsonRPCError) Error() string {
	return e.Message
}

// transportResponse carries either a JSON-RPC result or error.
type transportResponse struct {
	result json.RawMessage
	err    *jsonRPCError
}

// ProviderName represents supported LLM providers.
type ProviderName string

const (
	ProviderOpenRouter ProviderName = "openrouter"
	ProviderOllama     ProviderName = "ollama"
	ProviderLlamacpp   ProviderName = "llamacpp"
	ProviderOpenAI     ProviderName = "openai"
	ProviderMLX        ProviderName = "mlx"
	ProviderLLMGateway ProviderName = "llmgateway"
	ProviderAzure      ProviderName = "azure"
	ProviderZai        ProviderName = "zai"
	ProviderXai        ProviderName = "xai"
	ProviderCerebras   ProviderName = "cerebras"
	ProviderDeepSeek   ProviderName = "deepseek"
	ProviderVertexAI   ProviderName = "vertexai"
	ProviderNvidia     ProviderName = "nvidia"
)

// PermissionMode represents CLI-3 permission modes.
type PermissionMode string

const (
	PermissionInteractive  PermissionMode = "interactive"
	PermissionUnrestricted PermissionMode = "unrestricted"
	PermissionRestricted   PermissionMode = "restricted"
	PermissionExternal     PermissionMode = "external"
)

// PermissionDecision represents a permission prompt decision.
type PermissionDecision string

const (
	DecisionAllowOnce          PermissionDecision = "allow_once"
	DecisionDenyOnce           PermissionDecision = "deny_once"
	DecisionAllowSession       PermissionDecision = "allow_session"
	DecisionDenySession        PermissionDecision = "deny_session"
	DecisionAllowAlwaysProject PermissionDecision = "allow_always_project"
	DecisionAllowAlwaysUser    PermissionDecision = "allow_always_user"
	DecisionDenyAlwaysProject  PermissionDecision = "deny_always_project"
	DecisionDenyAlwaysUser     PermissionDecision = "deny_always_user"
	DecisionAlternative        PermissionDecision = "alternative"
)

// DecisionScope is the persistence scope for permission helpers.
type DecisionScope string

const (
	ScopeOnce    DecisionScope = "once"
	ScopeSession DecisionScope = "session"
	ScopeProject DecisionScope = "project"
	ScopeUser    DecisionScope = "user"
)

// Tool enumerates CLI tool names.
type Tool string

const (
	ToolReadFile          Tool = "read_file"
	ToolWriteFile         Tool = "write_file"
	ToolAppendFile        Tool = "append_file"
	ToolApplyPatch        Tool = "apply_patch"
	ToolFind              Tool = "find"
	ToolSearch            Tool = "search"
	ToolSearchReplace     Tool = "search_replace"
	ToolSearchWithContext Tool = "search_with_context"
	ToolSemanticSearch    Tool = "semantic_search"
	ToolListTree          Tool = "list_tree"
	ToolFileStats         Tool = "file_stats"
	ToolCreateDirectory   Tool = "create_directory"
	ToolDeletePath        Tool = "delete_path"
	ToolRenamePath        Tool = "rename_path"
	ToolCopyPath          Tool = "copy_path"
	ToolMultiFileEdit     Tool = "multi_file_edit"
	ToolRunCommand        Tool = "run_command"
	ToolCustomCommand     Tool = "custom_command"
	ToolGitStatus         Tool = "git_status"
	ToolGitDiff           Tool = "git_diff"
	ToolGitDiffRange      Tool = "git_diff_range"
	ToolGitLog            Tool = "git_log"
	ToolGitAdd            Tool = "git_add"
	ToolGitCommit         Tool = "git_commit"
	ToolGitBranch         Tool = "git_branch"
	ToolGitSwitch         Tool = "git_switch"
	ToolGitStash          Tool = "git_stash"
	ToolGitStashList      Tool = "git_stash_list"
	ToolGitStashPop       Tool = "git_stash_pop"
	ToolGitStashApply     Tool = "git_stash_apply"
	ToolGitStashDrop      Tool = "git_stash_drop"
	ToolGitMerge          Tool = "git_merge"
	ToolGitRebase         Tool = "git_rebase"
	ToolGitCherryPick     Tool = "git_cherry_pick"
	ToolGitFetch          Tool = "git_fetch"
	ToolGitPull           Tool = "git_pull"
	ToolGitPush           Tool = "git_push"
	ToolAutoCommit        Tool = "auto_commit"
	ToolGitApplyPatch     Tool = "git_apply_patch"
	ToolGitWorktreeList   Tool = "git_worktree_list"
	ToolGitWorktreeAdd    Tool = "git_worktree_add"
	ToolGitWorktreeRemove Tool = "git_worktree_remove"
	ToolWebSearch         Tool = "web_search"
	ToolNotebookRead      Tool = "notebook_read"
	ToolNotebookEdit      Tool = "notebook_edit"
	ToolAddDependency     Tool = "add_dependency"
	ToolRemoveDependency  Tool = "remove_dependency"
	ToolSaveMemory        Tool = "save_memory"
	ToolRecallMemory      Tool = "recall_memory"
	ToolPlan              Tool = "plan"
	ToolTodoWrite         Tool = "todo_write"
	ToolFormatFile        Tool = "format_file"
	ToolFormatDirectory   Tool = "format_directory"
	ToolListFormatters    Tool = "list_formatters"
)

// Config holds all SDK configuration.
type Config struct {
	CWD       string
	CLIPath   string
	Debug     bool
	Timeout   int // milliseconds

	Model         string
	FallbackModel string
	MaxTurns      int
	MaxBudgetUSD  float64
	Temperature   float64

	Provider ProviderName
	APIKey   string
	BaseURL  string

	PermissionMode PermissionMode
	Permissions    *PermissionSettings
	Yolo           string
	YoloTimeout    int
	PlanMode       bool

	AutoMode    bool
	Unrestricted bool
	AutoCommit  bool
	AutoSkill   bool

	MaxIterations int
	MaxRuntime    int
	MaxCost       float64

	SysPrompt        string
	AppendSysPrompt  string
	Instructions     string

	Skills      []SkillRef
	SkillRefs   []SkillRef

	ContextCompact bool
	MaxTokens      int
	CompressionThreshold    float64
	SummarizationThreshold  float64

	PersistSession  bool
	SessionID       string
	Resume          bool
	Continue        bool
	SessionPath     string
	AutoSaveInterval int

	AgentsMdEnable     bool
	AgentsMdCreate     bool
	AgentsMdPath       string
	AgentsMdAutoUpdate bool

	AdditionalDirectories []string
	AddDir                []string
	ExtraArgs             []string

	Env    map[string]string
	EnvVars map[string]string

	Thinking string
	Effort   string

	MCPServers map[string]MCPServerConfig
	Hooks      *HooksSettings
	Plugins    []string

	OutputFormat string
	AgentsMd     *AgentsMdSettings

	PathToClaudeCodeExecutable string
	SpawnClaudeCodeProcess     *bool
	DebugFile                  string
	StrictMCPConfig            bool
	Betas                      []string
	TaskBudget                 float64

	OpenAIAuthMode     string
	ReasoningEffort    string
	ChatGPTAccessToken string
	ChatGPTAccountID   string

	AzureAuthMethod    string
	AzureTenantID      string
	AzureClientID      string
	AzureClientSecret  string
	AzureResourceName  string
	AzureDeploymentName string
	AzureAPIVersion    string

	Port int

	CanUseTool              func(toolName string) bool
	EnableFileCheckpointing bool
	OnElicitation           func(params interface{}) interface{}
}

// PermissionSettings holds fine-grained permission configuration.
type PermissionSettings struct {
	Mode            PermissionMode
	AllowList       []string
	DenyList        []string
	Rules           []PermissionRule
	RememberSession bool
	DenyPatterns    []string
	AllowPatterns   []string
	AvailableTools  []string
	ExcludedTools   []string
	AllPathsAllowed bool
	AllUrlsAllowed  bool
}

// PermissionRule defines a single permission rule.
type PermissionRule struct {
	Tool    string
	Pattern string
	Action  string
}

// SkillRef is a skill reference (name or path).
type SkillRef struct {
	Name  string
	Path  string
	Scope string
}

// SkillSettings holds skill configuration.
type SkillSettings struct {
	AutoSkill      bool
	Skills         []SkillRef
	Sources        []string
	InstallMissing bool
}

// AgentsMdSettings holds AGENTS.md configuration.
type AgentsMdSettings struct {
	Enable           bool
	Enabled          bool
	Create           bool
	CreateDefault    bool
	Path             string
	AutoUpdate       bool
	IncludeTechStack bool
	IncludeCommands  bool
	IncludeSkills    bool
	IncludeConventions bool
}

// MCPServerConfig holds MCP server configuration.
type MCPServerConfig struct {
	Transport   string
	Command     string
	Args        []string
	URL         string
	Env         map[string]string
	Headers     map[string]string
	AutoConnect bool
}

// HooksSettings holds hooks configuration.
type HooksSettings struct {
	Enabled bool
	Hooks   []HookDefinition
}

// HookEvent represents hook event types.
type HookEvent string

// HookDefinition defines a lifecycle hook.
type HookDefinition struct {
	Event       HookEvent
	Command     string
	Description string
	Enabled     bool
	Timeout     int
	Async       bool
	Matcher     string
	Filter      *HookFilter
}

// HookFilter limits when a hook fires.
type HookFilter struct {
	Tool []string
	Path []string
}

// PromptParams holds prompt request parameters.
type PromptParams struct {
	Message       string
	Context       *PromptContext
	Images        []ImageAttachment
	ThinkingLevel string
	AgentsMd      interface{}
}

// PromptContext holds context for a prompt.
type PromptContext struct {
	Files     []string
	Selection *Selection
	AgentsMd  *AgentsMdContext
}

// Selection represents a text selection.
type Selection struct {
	File      string
	StartLine int
	EndLine   int
	Text      string
}

// AgentsMdContext holds AGENTS.md context for a prompt.
type AgentsMdContext struct {
	Content string
	Path    string
	Auto    bool
}

// ImageAttachment represents an image attachment.
type ImageAttachment struct {
	Data     string
	MimeType string
	Filename string
}

// GetStateResult holds agent state.
type GetStateResult struct {
	Status         string
	SessionID      string
	Model          string
	Workspace      string
	ContextPercent float64
	MessageCount   int
}

// GetMessagesResult holds message history.
type GetMessagesResult struct {
	Messages []RPCMessage
}

// RPCMessage represents a conversation message.
type RPCMessage struct {
	ID        string
	Role      string
	Content   string
	Timestamp string
	ToolCalls []ToolCall
}

// ToolCall represents a tool call within a message.
type ToolCall struct {
	ID   string
	Name string
	Args map[string]interface{}
}

// ModelInfo holds model information.
type ModelInfo struct {
	ID          string
	DisplayName string
	Description string
}

// AgentInfo holds subagent information.
type AgentInfo struct {
	ID          string
	Name        string
	Description string
	Tools       []string
}

// ContextUsage holds context window breakdown.
type ContextUsage struct {
	SystemPrompt int
	Tools        int
	Messages     int
	MCPTools     int
	MemoryFiles  int
	Total        int
}

// AccountInfo holds account information.
type AccountInfo struct {
	Email            string
	Organization     string
	SubscriptionType string
}

// SessionStats holds session statistics.
type SessionStats struct {
	TotalCost     float64
	TotalTokens   int
	InputTokens   int
	OutputTokens  int
	RequestCount  int
	Duration      float64
	ToolCallCount int
	StartedAt     string
	EndedAt       string
}

// SessionMetadata holds session metadata.
type SessionMetadata struct {
	SessionID    string
	CreatedAt    string
	LastActiveAt string
	ClosedAt     string
	ProjectPath  string
	ProjectName  string
	Model        string
	MessageCount int
	Summary      string
	Status       string
	ExitCode     int
	Type         string
}

// RunResult holds the result of a run.
type RunResult struct {
	ID     string
	Status string
	Text   string
	Events []Event
}

// Event is the union of all SDK events.
type Event interface {
	eventType() string
}

// AgentStartEvent is emitted when the agent starts.
type AgentStartEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Model     string `json:"model"`
	Workspace string `json:"workspace"`
	Timestamp string `json:"timestamp"`
}

func (e AgentStartEvent) eventType() string { return "agent_start" }

// AgentEndEvent is emitted when the agent ends.
type AgentEndEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
}

func (e AgentEndEvent) eventType() string { return "agent_end" }

// TurnStartEvent is emitted when a turn starts.
type TurnStartEvent struct {
	Type      string `json:"type"`
	TurnID    string `json:"turnId"`
	Timestamp string `json:"timestamp"`
}

func (e TurnStartEvent) eventType() string { return "turn_start" }

// TurnEndEvent is emitted when a turn ends.
type TurnEndEvent struct {
	Type           string  `json:"type"`
	TurnID         string  `json:"turnId"`
	Timestamp      string  `json:"timestamp"`
	TokensUsed     int     `json:"tokensUsed,omitempty"`
	DurationMs     int     `json:"durationMs,omitempty"`
	ContextPercent float64 `json:"contextPercent,omitempty"`
}

func (e TurnEndEvent) eventType() string { return "turn_end" }

// MessageStartEvent is emitted when message generation starts.
type MessageStartEvent struct {
	Type      string `json:"type"`
	MessageID string `json:"messageId"`
	Role      string `json:"role"`
	Timestamp string `json:"timestamp"`
}

func (e MessageStartEvent) eventType() string { return "message_start" }

// MessageUpdateEvent is emitted for streaming message deltas.
type MessageUpdateEvent struct {
	Type      string `json:"type"`
	MessageID string `json:"messageId,omitempty"`
	Delta     string `json:"delta"`
	Thought   string `json:"thought,omitempty"`
	Timestamp string `json:"timestamp"`
}

func (e MessageUpdateEvent) eventType() string { return "message_update" }

// MessageEndEvent is emitted when message generation ends.
type MessageEndEvent struct {
	Type      string `json:"type"`
	MessageID string `json:"messageId"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

func (e MessageEndEvent) eventType() string { return "message_end" }

// ToolStartEvent is emitted when a tool starts executing.
type ToolStartEvent struct {
	Type      string                 `json:"type"`
	ToolID    string                 `json:"toolId"`
	ToolName  string                 `json:"toolName"`
	Args      map[string]interface{} `json:"args"`
	Timestamp string                 `json:"timestamp"`
}

func (e ToolStartEvent) eventType() string { return "tool_start" }

// ToolUpdateEvent is emitted for streaming tool output.
type ToolUpdateEvent struct {
	Type      string `json:"type"`
	ToolID    string `json:"toolId"`
	Output    string `json:"output"`
	Stream    string `json:"stream"`
	Timestamp string `json:"timestamp"`
}

func (e ToolUpdateEvent) eventType() string { return "tool_update" }

// ToolEndEvent is emitted when a tool finishes.
type ToolEndEvent struct {
	Type      string `json:"type"`
	ToolID    string `json:"toolId"`
	ToolName  string `json:"toolName"`
	Success   bool   `json:"success"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
}

func (e ToolEndEvent) eventType() string { return "tool_end" }

// FileModifiedEvent is emitted when a file is modified.
type FileModifiedEvent struct {
	Type       string `json:"type"`
	FilePath   string `json:"filePath"`
	ChangeType string `json:"changeType"`
	ToolID     string `json:"toolId"`
	Timestamp  string `json:"timestamp"`
}

func (e FileModifiedEvent) eventType() string { return "file_modified" }

// PermissionRequestEvent is emitted when the agent requests permission.
type PermissionRequestEvent struct {
	Type        string              `json:"type"`
	RequestID   string              `json:"requestId"`
	Tool        string              `json:"tool"`
	Description string              `json:"description"`
	Context     PermissionContext   `json:"context"`
	Options     []string            `json:"options,omitempty"`
	Timestamp   string              `json:"timestamp"`
}

func (e PermissionRequestEvent) eventType() string { return "permission_request" }

// PermissionContext holds permission request context.
type PermissionContext struct {
	Command string   `json:"command,omitempty"`
	Path    string   `json:"path,omitempty"`
	Args    []string `json:"args,omitempty"`
}

// ErrorEvent is emitted on errors.
type ErrorEvent struct {
	Type        string `json:"type"`
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Recoverable bool   `json:"recoverable"`
	Timestamp   string `json:"timestamp"`
}

func (e ErrorEvent) eventType() string { return "error" }

// DetectProviderFromModel infers the provider from a model identifier.
func DetectProviderFromModel(model string) ProviderName {
	if model == "" {
		return ProviderOpenRouter
	}
	m := strings.ToLower(model)
	if strings.Contains(m, "glm") || strings.Contains(m, "z-ai") {
		return ProviderZai
	}
	if strings.Contains(m, "/") && !strings.Contains(m, "gpt") && !strings.Contains(m, "claude") {
		return ProviderOpenRouter
	}
	if strings.Contains(m, "gpt") || strings.Contains(m, "o1") || strings.Contains(m, "chatgpt") {
		return ProviderOpenAI
	}
	if strings.Contains(m, "claude") {
		return ProviderOpenRouter
	}
	if strings.Contains(m, "azure") || strings.HasPrefix(m, "gpt-4") || strings.HasPrefix(m, "gpt-5") {
		return ProviderAzure
	}
	if strings.Contains(m, "grok") {
		return ProviderXai
	}
	if strings.Contains(m, "deepseek") {
		return ProviderDeepSeek
	}
	if strings.Contains(m, "gemini") || strings.Contains(m, "vertex") {
		return ProviderVertexAI
	}
	if strings.Contains(m, "nvidia") {
		return ProviderNvidia
	}
	if strings.Contains(m, "cerebras") {
		return ProviderCerebras
	}
	if strings.Contains(m, "llama") || strings.Contains(m, "mistral") || strings.Contains(m, "codellama") {
		return ProviderOllama
	}
	return ProviderOpenRouter
}

// allowDecision maps a scope to an allow decision.
func allowDecision(scope DecisionScope) PermissionDecision {
	switch scope {
	case ScopeOnce:
		return DecisionAllowOnce
	case ScopeSession:
		return DecisionAllowSession
	case ScopeProject:
		return DecisionAllowAlwaysProject
	case ScopeUser:
		return DecisionAllowAlwaysUser
	default:
		return DecisionAllowOnce
	}
}

// denyDecision maps a scope to a deny decision.
func denyDecision(scope DecisionScope) PermissionDecision {
	switch scope {
	case ScopeOnce:
		return DecisionDenyOnce
	case ScopeSession:
		return DecisionDenySession
	case ScopeProject:
		return DecisionDenyAlwaysProject
	case ScopeUser:
		return DecisionDenyAlwaysUser
	default:
		return DecisionDenyOnce
	}
}

// IsSkillFilePath returns true if ref looks like a file path to a skill.
func IsSkillFilePath(ref string) bool {
	return strings.HasSuffix(ref, ".md") || strings.Contains(ref, "/") || strings.Contains(ref, "\\")
}

// GetSkillName extracts the skill name from a reference.
func GetSkillName(ref SkillRef) string {
	if ref.Name != "" {
		return ref.Name
	}
	if ref.Path != "" {
		base := filepath.Base(ref.Path)
		parts := strings.Split(base, ".")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

// GetSkillPath extracts the file path from a reference if applicable.
func GetSkillPath(ref SkillRef) string {
	if ref.Path != "" {
		return ref.Path
	}
	return ""
}

// AgentOptions extends Config with high-level agent conveniences.
type AgentOptions struct {
	Config
	Instructions string
}

// JsonParseOptions configures optional JSON validation.
type JsonParseOptions[T any] struct {
	Validate func(interface{}) (T, error)
}

// JsonRunOptions configures JSON-mode runs.
type JsonRunOptions[T any] struct {
	SchemaName         string
	Schema             interface{}
	OutputInstructions string
	JsonParseOptions[T]
}
