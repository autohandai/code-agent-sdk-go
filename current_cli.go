package autohand

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// PermissionAcknowledgedResult reports whether the CLI matched a pending
// permission request and extended its response deadline.
type PermissionAcknowledgedResult struct {
	Success bool `json:"success"`
}

// DirectoryAccessResponseResult reports whether a pending directory access
// request was found and resolved.
type DirectoryAccessResponseResult struct {
	Success bool `json:"success"`
}

// DirectoryAccessAcknowledgedResult reports whether the CLI matched a pending
// directory access request and extended its response deadline.
type DirectoryAccessAcknowledgedResult struct {
	Success bool `json:"success"`
}

// ChangesDecisionAction identifies how a multi-file preview batch should be
// applied.
type ChangesDecisionAction string

const (
	ChangesAcceptAll      ChangesDecisionAction = "accept_all"
	ChangesRejectAll      ChangesDecisionAction = "reject_all"
	ChangesAcceptSelected ChangesDecisionAction = "accept_selected"
)

// ChangesDecisionParams selects the disposition of a pending preview batch.
type ChangesDecisionParams struct {
	BatchID           string                `json:"batchId"`
	Action            ChangesDecisionAction `json:"action"`
	SelectedChangeIDs []string              `json:"selectedChangeIds,omitempty"`
}

func (p *ChangesDecisionParams) validate() error {
	if p == nil || strings.TrimSpace(p.BatchID) == "" {
		return fmt.Errorf("decide changes: batch ID is required")
	}
	switch p.Action {
	case ChangesAcceptAll, ChangesRejectAll:
		if len(p.SelectedChangeIDs) != 0 {
			return fmt.Errorf("decide changes: selected change IDs require %q", ChangesAcceptSelected)
		}
	case ChangesAcceptSelected:
		if len(p.SelectedChangeIDs) == 0 {
			return fmt.Errorf("decide changes: at least one selected change ID is required")
		}
		for _, id := range p.SelectedChangeIDs {
			if strings.TrimSpace(id) == "" {
				return fmt.Errorf("decide changes: selected change IDs cannot be blank")
			}
		}
	default:
		return fmt.Errorf("decide changes: unsupported action %q", p.Action)
	}
	return nil
}

// ChangesDecisionError describes one proposed change that could not be
// applied.
type ChangesDecisionError struct {
	ChangeID string `json:"changeId"`
	Error    string `json:"error"`
}

// ChangesDecisionResult summarizes application of a preview batch.
type ChangesDecisionResult struct {
	Success      bool                   `json:"success"`
	AppliedCount int                    `json:"appliedCount"`
	SkippedCount int                    `json:"skippedCount"`
	Errors       []ChangesDecisionError `json:"errors,omitempty"`
}

// SessionHistoryStatus is the terminal or live state stored for a session.
type SessionHistoryStatus string

const (
	SessionHistoryActive    SessionHistoryStatus = "active"
	SessionHistoryCompleted SessionHistoryStatus = "completed"
	SessionHistoryCrashed   SessionHistoryStatus = "crashed"
)

// GetHistoryParams controls pagination of stored sessions. Zero values ask the
// CLI to use its defaults.
type GetHistoryParams struct {
	Page     int `json:"page,omitempty"`
	PageSize int `json:"pageSize,omitempty"`
}

func (p *GetHistoryParams) validate() error {
	if p == nil {
		return nil
	}
	if p.Page < 0 {
		return fmt.Errorf("get history: page cannot be negative")
	}
	if p.PageSize < 0 {
		return fmt.Errorf("get history: page size cannot be negative")
	}
	return nil
}

// SessionHistoryEntry is a summary of one stored session.
type SessionHistoryEntry struct {
	SessionID    string               `json:"sessionId"`
	CreatedAt    string               `json:"createdAt"`
	LastActiveAt string               `json:"lastActiveAt"`
	ProjectName  string               `json:"projectName"`
	Model        string               `json:"model"`
	MessageCount int                  `json:"messageCount"`
	Status       SessionHistoryStatus `json:"status"`
}

// GetHistoryResult contains one page of stored sessions.
type GetHistoryResult struct {
	Sessions    []SessionHistoryEntry `json:"sessions"`
	CurrentPage int                   `json:"currentPage"`
	TotalPages  int                   `json:"totalPages"`
	TotalItems  int                   `json:"totalItems"`
}

// GetSessionResult is a discriminated result for a session lookup. Use a type
// switch to distinguish SessionDetails from SessionLookupFailure.
type GetSessionResult interface {
	sessionLookupResult()
	Succeeded() bool
}

// SessionDetails contains the complete stored session payload.
type SessionDetails struct {
	SessionID     string       `json:"sessionId"`
	ProjectName   string       `json:"projectName"`
	Model         string       `json:"model"`
	MessageCount  int          `json:"messageCount"`
	Status        string       `json:"status"`
	CreatedAt     string       `json:"createdAt"`
	LastActiveAt  string       `json:"lastActiveAt"`
	Summary       string       `json:"summary,omitempty"`
	Messages      []RPCMessage `json:"messages"`
	WorkspaceRoot string       `json:"workspaceRoot"`
}

func (SessionDetails) sessionLookupResult() {}
func (SessionDetails) Succeeded() bool      { return true }

// SessionLookupFailure reports why a stored session could not be loaded.
type SessionLookupFailure struct {
	Error string `json:"error,omitempty"`
}

func (SessionLookupFailure) sessionLookupResult() {}
func (SessionLookupFailure) Succeeded() bool      { return false }

// SessionAttachResult reports the active session after an attach request.
type SessionAttachResult struct {
	Success       bool   `json:"success"`
	SessionID     string `json:"sessionId,omitempty"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
	MessageCount  int    `json:"messageCount,omitempty"`
	Error         string `json:"error,omitempty"`
}

// YoloSetParams configures unrestricted mode. The current CLI treats any
// non-empty pattern as enabled and an empty pattern as disabled.
type YoloSetParams struct {
	Pattern        string `json:"pattern"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty"`
}

func (p *YoloSetParams) validate() error {
	if p == nil {
		return fmt.Errorf("set YOLO mode: params are required")
	}
	if p.TimeoutSeconds < 0 {
		return fmt.Errorf("set YOLO mode: timeout seconds cannot be negative")
	}
	return nil
}

// YoloSetResult reports the effective expiry when unrestricted mode is timed.
type YoloSetResult struct {
	Success   bool `json:"success"`
	ExpiresIn *int `json:"expiresIn,omitempty"`
}

// MCPInputSchema describes object-shaped arguments accepted by a VS Code MCP
// tool.
type MCPInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// MCPVSCodeTool is a tool descriptor supplied by a VS Code extension.
type MCPVSCodeTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	ServerName  string          `json:"serverName"`
	InputSchema *MCPInputSchema `json:"inputSchema,omitempty"`
}

// MCPSetVSCodeToolsParams replaces the CLI's extension-provided MCP tools.
// An empty list clears previously registered tools.
type MCPSetVSCodeToolsParams struct {
	Tools []MCPVSCodeTool `json:"tools"`
}

func (p *MCPSetVSCodeToolsParams) validate() error {
	if p == nil {
		return fmt.Errorf("set VS Code MCP tools: params are required")
	}
	if p.Tools == nil {
		return fmt.Errorf("set VS Code MCP tools: tools must be an initialized list")
	}
	for i, tool := range p.Tools {
		if strings.TrimSpace(tool.Name) == "" || strings.TrimSpace(tool.Description) == "" || strings.TrimSpace(tool.ServerName) == "" {
			return fmt.Errorf("set VS Code MCP tools: tool %d requires name, description, and server name", i)
		}
		if tool.InputSchema != nil && tool.InputSchema.Type != "object" {
			return fmt.Errorf("set VS Code MCP tools: tool %d input schema type must be object", i)
		}
	}
	return nil
}

// MCPSetVSCodeToolsResult reports whether the tool set was accepted.
type MCPSetVSCodeToolsResult struct {
	Success bool `json:"success"`
}

// MCPInvocationResponseParams completes a VS Code MCP invocation requested by
// the CLI.
type MCPInvocationResponseParams struct {
	RequestID string  `json:"requestId"`
	Success   bool    `json:"success"`
	Result    *string `json:"result,omitempty"`
	Error     string  `json:"error,omitempty"`
}

func (p *MCPInvocationResponseParams) validate() error {
	if p == nil || strings.TrimSpace(p.RequestID) == "" {
		return fmt.Errorf("respond to MCP invocation: request ID is required")
	}
	if p.Success && p.Error != "" {
		return fmt.Errorf("respond to MCP invocation: successful responses cannot include an error")
	}
	if !p.Success {
		if strings.TrimSpace(p.Error) == "" {
			return fmt.Errorf("respond to MCP invocation: failed responses require an error")
		}
		if p.Result != nil {
			return fmt.Errorf("respond to MCP invocation: failed responses cannot include a result")
		}
	}
	return nil
}

// MCPInvocationResponseResult reports whether a pending invocation accepted
// the response.
type MCPInvocationResponseResult struct {
	Success bool `json:"success"`
}

// LearningAuditStatus classifies overlap between project skills.
type LearningAuditStatus string

const (
	LearningAuditRedundant   LearningAuditStatus = "redundant"
	LearningAuditOutdated    LearningAuditStatus = "outdated"
	LearningAuditConflicting LearningAuditStatus = "conflicting"
)

// LearnRecommendParams controls whether recommendation analysis performs a
// deeper project scan.
type LearnRecommendParams struct {
	Deep bool `json:"deep,omitempty"`
}

// LearningAuditEntry describes one existing skill concern.
type LearningAuditEntry struct {
	Skill  string              `json:"skill"`
	Status LearningAuditStatus `json:"status"`
	Reason string              `json:"reason"`
}

// LearningRecommendation is a scored registry recommendation.
type LearningRecommendation struct {
	Slug   string  `json:"slug"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

// LearnRecommendResult contains the CLI's project learning analysis.
type LearnRecommendResult struct {
	Success         bool                     `json:"success"`
	ProjectSummary  string                   `json:"projectSummary"`
	Audit           []LearningAuditEntry     `json:"audit"`
	Recommendations []LearningRecommendation `json:"recommendations"`
	GapAnalysis     *string                  `json:"gapAnalysis"`
	Error           string                   `json:"error,omitempty"`
}

// AcknowledgePermission confirms that a permission request reached the SDK
// client. Callers must still answer the request with PermissionResponse.
func (c *RPCClient) AcknowledgePermission(ctx context.Context, requestID string) (*PermissionAcknowledgedResult, error) {
	if strings.TrimSpace(requestID) == "" {
		return nil, fmt.Errorf("acknowledge permission: request ID is required")
	}
	return rpcRequest[PermissionAcknowledgedResult](ctx, c, "autohand.permissionAcknowledged", map[string]string{
		"requestId": requestID,
	})
}

// AcknowledgePermission confirms receipt of a permission request.
func (s *SDK) AcknowledgePermission(ctx context.Context, requestID string) (*PermissionAcknowledgedResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.AcknowledgePermission(ctx, requestID)
}

// RespondToDirectoryAccess resolves a pending request for access to an
// additional directory.
func (c *RPCClient) RespondToDirectoryAccess(ctx context.Context, requestID string, granted bool) (*DirectoryAccessResponseResult, error) {
	if strings.TrimSpace(requestID) == "" {
		return nil, fmt.Errorf("respond to directory access: request ID is required")
	}
	return rpcRequest[DirectoryAccessResponseResult](ctx, c, "autohand.directoryAccessResponse", struct {
		RequestID string `json:"requestId"`
		Granted   bool   `json:"granted"`
	}{RequestID: requestID, Granted: granted})
}

// RespondToDirectoryAccess allows or denies a pending directory access
// request.
func (s *SDK) RespondToDirectoryAccess(ctx context.Context, requestID string, granted bool) (*DirectoryAccessResponseResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.RespondToDirectoryAccess(ctx, requestID, granted)
}

// AcknowledgeDirectoryAccess confirms receipt of a directory access request.
func (c *RPCClient) AcknowledgeDirectoryAccess(ctx context.Context, requestID string) (*DirectoryAccessAcknowledgedResult, error) {
	if strings.TrimSpace(requestID) == "" {
		return nil, fmt.Errorf("acknowledge directory access: request ID is required")
	}
	return rpcRequest[DirectoryAccessAcknowledgedResult](ctx, c, "autohand.directoryAccessAcknowledged", map[string]string{
		"requestId": requestID,
	})
}

// AcknowledgeDirectoryAccess confirms that a directory access request reached
// the SDK client.
func (s *SDK) AcknowledgeDirectoryAccess(ctx context.Context, requestID string) (*DirectoryAccessAcknowledgedResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.AcknowledgeDirectoryAccess(ctx, requestID)
}

// DecideChanges applies or rejects a multi-file preview batch.
func (c *RPCClient) DecideChanges(ctx context.Context, params *ChangesDecisionParams) (*ChangesDecisionResult, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	return rpcRequest[ChangesDecisionResult](ctx, c, "autohand.changesDecision", params)
}

// DecideChanges applies or rejects a multi-file preview batch.
func (s *SDK) DecideChanges(ctx context.Context, params *ChangesDecisionParams) (*ChangesDecisionResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.DecideChanges(ctx, params)
}

// GetHistory returns a page of stored CLI sessions.
func (c *RPCClient) GetHistory(ctx context.Context, params *GetHistoryParams) (*GetHistoryResult, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	if params == nil {
		params = &GetHistoryParams{}
	}
	return rpcRequest[GetHistoryResult](ctx, c, "autohand.getHistory", params)
}

// GetHistory returns a page of stored CLI sessions.
func (s *SDK) GetHistory(ctx context.Context, params *GetHistoryParams) (*GetHistoryResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetHistory(ctx, params)
}

// GetSession returns either complete session details or an explicit lookup
// failure. Malformed success payloads are rejected instead of becoming
// partially initialized details.
func (c *RPCClient) GetSession(ctx context.Context, sessionID string) (GetSessionResult, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("get session: session ID is required")
	}
	raw, err := c.transport.Request(ctx, "autohand.getSession", map[string]string{"sessionId": sessionID})
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Success *bool `json:"success"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("unmarshal autohand.getSession result: %w", err)
	}
	if envelope.Success == nil {
		return nil, fmt.Errorf("unmarshal autohand.getSession result: missing success discriminator")
	}
	if !*envelope.Success {
		var failure SessionLookupFailure
		if err := json.Unmarshal(raw, &failure); err != nil {
			return nil, fmt.Errorf("unmarshal autohand.getSession failure: %w", err)
		}
		return failure, nil
	}
	var details SessionDetails
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil, fmt.Errorf("unmarshal autohand.getSession details: %w", err)
	}
	if strings.TrimSpace(details.SessionID) == "" || strings.TrimSpace(details.WorkspaceRoot) == "" || details.Messages == nil {
		return nil, fmt.Errorf("unmarshal autohand.getSession details: missing required session fields")
	}
	return details, nil
}

// GetSession returns either complete session details or an explicit lookup
// failure.
func (s *SDK) GetSession(ctx context.Context, sessionID string) (GetSessionResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.GetSession(ctx, sessionID)
}

// AttachSession restores a stored CLI session into the active RPC process.
func (c *RPCClient) AttachSession(ctx context.Context, sessionID string) (*SessionAttachResult, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("attach session: session ID is required")
	}
	return rpcRequest[SessionAttachResult](ctx, c, "autohand.session.attach", map[string]string{
		"sessionId": sessionID,
	})
}

// AttachSession restores a stored CLI session into the active RPC process.
func (s *SDK) AttachSession(ctx context.Context, sessionID string) (*SessionAttachResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.AttachSession(ctx, sessionID)
}

func (c *RPCClient) setYolo(ctx context.Context, method string, params *YoloSetParams) (*YoloSetResult, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	return rpcRequest[YoloSetResult](ctx, c, method, params)
}

// SetYolo sets timed unrestricted mode through the canonical CLI method.
func (c *RPCClient) SetYolo(ctx context.Context, params *YoloSetParams) (*YoloSetResult, error) {
	return c.setYolo(ctx, "autohand.yoloSet", params)
}

// SetYoloAlias sets timed unrestricted mode through the dotted compatibility
// alias exposed by current CLIs.
func (c *RPCClient) SetYoloAlias(ctx context.Context, params *YoloSetParams) (*YoloSetResult, error) {
	return c.setYolo(ctx, "autohand.yolo.set", params)
}

// SetYolo sets timed unrestricted mode through the canonical CLI method.
func (s *SDK) SetYolo(ctx context.Context, params *YoloSetParams) (*YoloSetResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.SetYolo(ctx, params)
}

// SetYoloAlias uses the dotted compatibility alias for timed unrestricted
// mode.
func (s *SDK) SetYoloAlias(ctx context.Context, params *YoloSetParams) (*YoloSetResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.SetYoloAlias(ctx, params)
}

// SetVSCodeMCPTools replaces the extension-provided MCP tool descriptors.
func (c *RPCClient) SetVSCodeMCPTools(ctx context.Context, params *MCPSetVSCodeToolsParams) (*MCPSetVSCodeToolsResult, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	return rpcRequest[MCPSetVSCodeToolsResult](ctx, c, "autohand.mcp.setVscodeTools", params)
}

// SetVSCodeMCPTools replaces the extension-provided MCP tool descriptors.
func (s *SDK) SetVSCodeMCPTools(ctx context.Context, params *MCPSetVSCodeToolsParams) (*MCPSetVSCodeToolsResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.SetVSCodeMCPTools(ctx, params)
}

// RespondToMCPInvocation completes a VS Code MCP invocation.
func (c *RPCClient) RespondToMCPInvocation(ctx context.Context, params *MCPInvocationResponseParams) (*MCPInvocationResponseResult, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	return rpcRequest[MCPInvocationResponseResult](ctx, c, "autohand.mcp.invokeResponse", params)
}

// RespondToMCPInvocation completes a VS Code MCP invocation.
func (s *SDK) RespondToMCPInvocation(ctx context.Context, params *MCPInvocationResponseParams) (*MCPInvocationResponseResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.RespondToMCPInvocation(ctx, params)
}

// RecommendProjectLearning audits project skills and returns scored registry
// recommendations.
func (c *RPCClient) RecommendProjectLearning(ctx context.Context, params *LearnRecommendParams) (*LearnRecommendResult, error) {
	if params == nil {
		params = &LearnRecommendParams{}
	}
	return rpcRequest[LearnRecommendResult](ctx, c, "autohand.learn.recommend", params)
}

// RecommendProjectLearning audits project skills and returns scored registry
// recommendations.
func (s *SDK) RecommendProjectLearning(ctx context.Context, params *LearnRecommendParams) (*LearnRecommendResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.RecommendProjectLearning(ctx, params)
}
