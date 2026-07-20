package autohand

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestDiscoveryRPCMethodsAndTypedResults(t *testing.T) {
	client, requests, cleanup := newRPCTestClient(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	refresh := true
	registry, err := client.GetSkillsRegistry(ctx, &GetSkillsRegistryParams{ForceRefresh: &refresh})
	if err != nil || !registry.Success || len(registry.Skills) != 1 || registry.Skills[0].ID != "skill-1" {
		t.Fatalf("registry = %#v, err = %v", registry, err)
	}
	installed, err := client.InstallSkill(ctx, &InstallSkillParams{SkillName: "review", Scope: SkillInstallScopeProject})
	if err != nil || !installed.Success || installed.SkillName != "review" {
		t.Fatalf("install = %#v, err = %v", installed, err)
	}
	servers, err := client.ListMCPServers(ctx)
	if err != nil || len(servers.Servers) != 1 || servers.Servers[0].ToolCount != 2 {
		t.Fatalf("servers = %#v, err = %v", servers, err)
	}
	tools, err := client.ListMCPTools(ctx, &MCPListToolsParams{ServerName: "github"})
	if err != nil || len(tools.Tools) != 1 || tools.Tools[0].ServerName != "github" {
		t.Fatalf("tools = %#v, err = %v", tools, err)
	}
	configs, err := client.GetMCPServerConfigs(ctx)
	if err != nil || len(configs.Configs) != 1 || configs.Configs[0].Transport != MCPTransportStdio {
		t.Fatalf("configs = %#v, err = %v", configs, err)
	}

	var got []string
	var params []map[string]interface{}
	for range 5 {
		request := <-requests
		got = append(got, request.Method)
		var value map[string]interface{}
		if err := json.Unmarshal(request.Params, &value); err != nil {
			t.Fatal(err)
		}
		params = append(params, value)
	}
	want := []string{
		"autohand.getSkillsRegistry",
		"autohand.installSkill",
		"autohand.mcp.listServers",
		"autohand.mcp.listTools",
		"autohand.mcp.getServerConfigs",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("methods = %v, want %v", got, want)
	}
	if params[0]["forceRefresh"] != true || params[1]["skillName"] != "review" || params[1]["scope"] != "project" || params[3]["serverName"] != "github" {
		t.Fatalf("unexpected params: %#v", params)
	}
}

func TestInstallSkillRejectsInvalidScope(t *testing.T) {
	client := NewRPCClient(&Config{})
	_, err := client.InstallSkill(context.Background(), &InstallSkillParams{SkillName: "review", Scope: "workspace"})
	if err == nil {
		t.Fatal("InstallSkill accepted an unsupported scope")
	}
}
