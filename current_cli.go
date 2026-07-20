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
