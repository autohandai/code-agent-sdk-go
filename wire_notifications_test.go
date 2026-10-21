package autohand

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestWireNotificationsPreservePacketsAndHandlers(t *testing.T) {
	sdk := NewSDK(&Config{})
	var packets []WireNotification
	typed := 0
	sdk.client.transport.OnNotification("autohand.pipe.output", func(json.RawMessage) { typed++ })
	sdk.OnNotification(func(WireNotification) { panic("observer failure") })
	sdk.OnNotification(func(packet WireNotification) {
		if len(packet.Params) > 0 {
			packet.Params[0] = 'X'
		}
	})
	remove := (&Agent{sdk: sdk}).OnNotification(func(packet WireNotification) { packets = append(packets, packet) })
	payloads := []string{`{"type":"progress","future":false}`, `[1,null,false]`, `null`}
	for _, payload := range payloads {
		sdk.client.transport.handleLine(`{"jsonrpc":"2.0","method":"autohand.pipe.output","params":` + payload + `}`)
	}
	if typed != 3 {
		t.Fatalf("typed handler count: %d", typed)
	}
	var got []string
	for _, packet := range packets {
		if packet.Method != "autohand.pipe.output" {
			t.Fatal(packet.Method)
		}
		got = append(got, string(packet.Params))
	}
	if !reflect.DeepEqual(got, payloads) {
		t.Fatalf("packets: %v", got)
	}
	remove()
	sdk.client.transport.handleLine(`{"jsonrpc":"2.0","method":"autohand.future","params":null}`)
	if len(packets) != 3 {
		t.Fatal("unsubscribe failed")
	}
}
