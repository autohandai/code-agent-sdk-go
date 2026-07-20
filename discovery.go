package autohand

// CommunitySkill describes a skill published in the Autohand community registry.
type CommunitySkill struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Category      string   `json:"category"`
	Tags          []string `json:"tags,omitempty"`
	Rating        *float64 `json:"rating,omitempty"`
	DownloadCount *int     `json:"downloadCount,omitempty"`
	IsFeatured    *bool    `json:"isFeatured,omitempty"`
	IsCurated     *bool    `json:"isCurated,omitempty"`
}

// SkillCategory summarizes the number of registry skills in a category.
type SkillCategory struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// GetSkillsRegistryParams configures registry discovery.
type GetSkillsRegistryParams struct {
	ForceRefresh *bool `json:"forceRefresh,omitempty"`
}

// GetSkillsRegistryResult is returned by autohand.getSkillsRegistry.
type GetSkillsRegistryResult struct {
	Success    bool             `json:"success"`
	Skills     []CommunitySkill `json:"skills"`
	Categories []SkillCategory  `json:"categories"`
	Error      string           `json:"error,omitempty"`
}

// SkillInstallScope is the destination for an installed skill.
type SkillInstallScope string

const (
	SkillInstallScopeUser    SkillInstallScope = "user"
	SkillInstallScopeProject SkillInstallScope = "project"
)

// InstallSkillParams configures a registry skill installation.
type InstallSkillParams struct {
	SkillName string            `json:"skillName"`
	Scope     SkillInstallScope `json:"scope"`
	Force     *bool             `json:"force,omitempty"`
}

// InstallSkillResult is returned by autohand.installSkill.
type InstallSkillResult struct {
	Success   bool   `json:"success"`
	SkillName string `json:"skillName,omitempty"`
	Path      string `json:"path,omitempty"`
	Error     string `json:"error,omitempty"`
}

// MCPServerInfo describes a known MCP server and its live status.
type MCPServerInfo struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	ToolCount int    `json:"toolCount"`
}

// MCPListServersResult is returned by autohand.mcp.listServers.
type MCPListServersResult struct {
	Servers []MCPServerInfo `json:"servers"`
}

// MCPListToolsParams optionally filters tools by server name.
type MCPListToolsParams struct {
	ServerName string `json:"serverName,omitempty"`
}

// MCPToolInfo describes a tool exposed by an MCP server.
type MCPToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ServerName  string `json:"serverName"`
}

// MCPListToolsResult is returned by autohand.mcp.listTools.
type MCPListToolsResult struct {
	Tools []MCPToolInfo `json:"tools"`
}

// MCPTransport identifies the wire transport used by an MCP server.
type MCPTransport string

const (
	MCPTransportStdio MCPTransport = "stdio"
	MCPTransportSSE   MCPTransport = "sse"
	MCPTransportHTTP  MCPTransport = "http"
)

// MCPServerConfigInfo is a named MCP server configuration returned by the CLI.
type MCPServerConfigInfo struct {
	Name        string            `json:"name"`
	Transport   MCPTransport      `json:"transport"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	URL         string            `json:"url,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	AutoConnect *bool             `json:"autoConnect,omitempty"`
}

// MCPGetServerConfigsResult is returned by autohand.mcp.getServerConfigs.
type MCPGetServerConfigsResult struct {
	Configs []MCPServerConfigInfo `json:"configs"`
}
