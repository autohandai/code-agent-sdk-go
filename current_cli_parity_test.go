package autohand

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestCurrentBrowserCapabilityContract(t *testing.T) {
	client, requests, closeTransport := newRPCTestClient(t)
	defer closeTransport()
	result, err := client.SetBrowserCapabilities(context.Background(), &BrowserCapabilitiesSetParams{ProtocolVersion: 2, ExtensionVersion: "1.0.0", Tools: []string{"browser_navigate"}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Enabled || result.ProtocolVersion != 2 {
		t.Fatalf("result: %+v", result)
	}
	request := <-requests
	if request.Method != "autohand.browserCapabilities.set" {
		t.Fatalf("method: %s", request.Method)
	}
	var params map[string]interface{}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		t.Fatal(err)
	}
	expected := map[string]interface{}{"protocolVersion": float64(2), "extensionVersion": "1.0.0", "tools": []interface{}{"browser_navigate"}}
	if !reflect.DeepEqual(params, expected) {
		t.Fatalf("params: %v", params)
	}
	if ProviderAnthropic != ProviderName("anthropic") {
		t.Fatal("incorrect Anthropic provider ID")
	}
}
