package autohand

import (
	"context"
	"fmt"
)

// BrowserCapabilitiesSetParams negotiates browser extension tools before a task.
type BrowserCapabilitiesSetParams struct {
	ProtocolVersion  int      `json:"protocolVersion"`
	ExtensionVersion string   `json:"extensionVersion"`
	Tools            []string `json:"tools"`
}

// BrowserCapabilitiesSetResult contains the CLI-selected protocol and tool list.
type BrowserCapabilitiesSetResult struct {
	Enabled         bool     `json:"enabled"`
	ProtocolVersion int      `json:"protocolVersion"`
	Tools           []string `json:"tools"`
}

// SetBrowserCapabilities negotiates the CLI browser tool protocol.
func (c *RPCClient) SetBrowserCapabilities(ctx context.Context, params *BrowserCapabilitiesSetParams) (*BrowserCapabilitiesSetResult, error) {
	if params == nil {
		return nil, fmt.Errorf("browser capabilities are required")
	}
	result, err := rpcRequest[BrowserCapabilitiesSetResult](ctx, c, "autohand.browserCapabilities.set", params)
	if err != nil {
		return nil, err
	}
	if result.ProtocolVersion != 1 && result.ProtocolVersion != 2 {
		return nil, fmt.Errorf("invalid browser capability protocol version: %d", result.ProtocolVersion)
	}
	return result, nil
}

// SetBrowserCapabilities negotiates browser tools for this SDK session.
func (s *SDK) SetBrowserCapabilities(ctx context.Context, params *BrowserCapabilitiesSetParams) (*BrowserCapabilitiesSetResult, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return s.client.SetBrowserCapabilities(ctx, params)
}

// SetBrowserCapabilities negotiates browser tools for this agent.
func (a *Agent) SetBrowserCapabilities(ctx context.Context, params *BrowserCapabilitiesSetParams) (*BrowserCapabilitiesSetResult, error) {
	return a.sdk.SetBrowserCapabilities(ctx, params)
}
