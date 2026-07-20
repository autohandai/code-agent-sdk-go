package autohand

import "fmt"

// BrowserHandoffCreateParams configures browser extension routing for a handoff.
type BrowserHandoffCreateParams struct {
	ExtensionID *string `json:"extensionId,omitempty"`
	InstallURL  *string `json:"installUrl,omitempty"`
}

// BrowserHandoffCreateResult describes a newly created browser handoff.
type BrowserHandoffCreateResult struct {
	Token         string `json:"token"`
	SessionID     string `json:"sessionId"`
	WorkspaceRoot string `json:"workspaceRoot"`
	CreatedAt     string `json:"createdAt"`
	ExpiresAt     string `json:"expiresAt"`
	URL           string `json:"url"`
}

// BrowserHandoffAttachParams identifies a browser handoff token.
type BrowserHandoffAttachParams struct {
	Token string `json:"token"`
}

func (p *BrowserHandoffAttachParams) validate() error {
	if p == nil || p.Token == "" {
		return fmt.Errorf("browser handoff token is required")
	}
	return nil
}

// BrowserHandoffAttachResult describes a restored browser handoff session.
type BrowserHandoffAttachResult struct {
	Success       bool    `json:"success"`
	SessionID     *string `json:"sessionId,omitempty"`
	WorkspaceRoot *string `json:"workspaceRoot,omitempty"`
	MessageCount  *int    `json:"messageCount,omitempty"`
}
