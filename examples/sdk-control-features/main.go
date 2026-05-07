// sdk-control-features tests that SDK control methods are wired correctly.
//
// Usage:
//
//	go run ./examples/sdk-control-features
package main

import (
	"fmt"
	"log"
	"reflect"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func main() {
	fmt.Println("Testing SDK Control Features...\n")

	sdk := autohand.NewSDK(&autohand.Config{})
	fmt.Println("SDK initialized with control options")

	// Test that the SDK has the new control methods
	controlMethods := []string{
		"SetPermissionMode", "SetPlanMode", "EnablePlanMode", "DisablePlanMode",
		"SetModel", "SetMaxThinkingTokens", "ApplyFlagSettings",
		"SupportedModels", "GetContextUsage", "ReloadPlugins",
		"AccountInfo", "ToggleMCPServer", "ReconnectMCPServer", "SetMCPServers",
	}

	var missing []string
	sdkType := reflect.TypeOf(sdk)
	for _, method := range controlMethods {
		_, hasMethod := sdkType.MethodByName(method)
		if hasMethod {
			fmt.Printf("  SDK has method: %s\n", method)
		} else {
			fmt.Printf("  SDK missing method: %s\n", method)
			missing = append(missing, method)
		}
	}

	// Test that the RPC client has the new methods
	fmt.Println("\nTesting RPC Client methods...")
	client := sdk.client
	clientType := reflect.TypeOf(client)
	rpcMethods := []string{
		"SetPermissionMode", "SetPlanMode", "SetModel", "SetMaxThinkingTokens",
		"ApplyFlagSettings", "GetSupportedModels", "GetSupportedCommands",
		"GetContextUsage", "ReloadPlugins", "GetAccountInfo",
		"ToggleMCPServer", "ReconnectMCPServer", "SetMCPServers",
	}

	var rpcMissing []string
	for _, method := range rpcMethods {
		_, hasMethod := clientType.MethodByName(method)
		if hasMethod {
			fmt.Printf("  RPC Client has method: %s\n", method)
		} else {
			fmt.Printf("  RPC Client missing method: %s\n", method)
			rpcMissing = append(rpcMissing, method)
		}
	}

	if len(missing) > 0 {
		log.Fatalf("Missing SDK methods: %v", missing)
	}
	if len(rpcMissing) > 0 {
		log.Fatalf("Missing RPC methods: %v", rpcMissing)
	}

	fmt.Println("\nAll SDK control features are wired correctly")
}
