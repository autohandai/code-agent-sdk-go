package autohand

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
