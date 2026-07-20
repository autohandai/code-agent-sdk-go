package autohand

import (
	"context"
	"testing"
	"time"
)

func TestEventSubscriptionsBroadcastIndependently(t *testing.T) {
	client := NewRPCClient(&Config{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := client.Events(ctx)
	second := client.Events(ctx)

	client.queueEvent(ErrorEvent{Type: "error", Message: "boom"})
	for name, events := range map[string]<-chan Event{"first": first, "second": second} {
		select {
		case event := <-events:
			if got := event.(ErrorEvent).Message; got != "boom" {
				t.Fatalf("%s subscriber message = %q", name, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s subscriber did not receive broadcast", name)
		}
	}
}

func TestCanceledSubscriptionDoesNotConsumeFutureEvents(t *testing.T) {
	client := NewRPCClient(&Config{})
	canceledCtx, cancel := context.WithCancel(context.Background())
	canceled := client.Events(canceledCtx)
	cancel()
	select {
	case _, ok := <-canceled:
		if ok {
			t.Fatal("canceled subscription remained open")
		}
	case <-time.After(time.Second):
		t.Fatal("canceled subscription was not removed")
	}

	liveCtx, liveCancel := context.WithCancel(context.Background())
	defer liveCancel()
	live := client.Events(liveCtx)
	client.queueEvent(ErrorEvent{Type: "error", Message: "still-live"})
	select {
	case event := <-live:
		if got := event.(ErrorEvent).Message; got != "still-live" {
			t.Fatalf("live subscriber message = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("future event was lost after cancellation")
	}
}
