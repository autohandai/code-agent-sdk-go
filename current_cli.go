package autohand

import (
	"context"
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
