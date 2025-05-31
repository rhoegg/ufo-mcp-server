package events

import (
	"testing"
	"time"
)

func TestBroadcaster_SubscribeUnsubscribe(t *testing.T) {
	b := NewBroadcaster()
	defer b.Close()

	// Test subscribe
	sub1 := b.Subscribe("client1")
	if sub1.ID != "client1" {
		t.Errorf("expected subscriber ID 'client1', got %s", sub1.ID)
	}

	if b.GetSubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", b.GetSubscriberCount())
	}

	// Test multiple subscribers
	_ = b.Subscribe("client2")
	if b.GetSubscriberCount() != 2 {
		t.Errorf("expected 2 subscribers, got %d", b.GetSubscriberCount())
	}

	// Test unsubscribe
	b.Unsubscribe("client1")
	if b.GetSubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber after unsubscribe, got %d", b.GetSubscriberCount())
	}

	// Test unsubscribe non-existent
	b.Unsubscribe("nonexistent")
	if b.GetSubscriberCount() != 1 {
		t.Errorf("subscriber count changed after unsubscribing non-existent")
	}

	b.Unsubscribe("client2")
	if b.GetSubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", b.GetSubscriberCount())
	}
}

func TestBroadcaster_PublishEvents(t *testing.T) {
	b := NewBroadcaster()
	defer b.Close()

	sub := b.Subscribe("test_client")

	// Test basic publish
	testEvent := Event{
		Type: "test_event",
		Data: map[string]interface{}{"test": "data"},
	}

	b.Publish(testEvent)

	// Wait for event
	select {
	case received := <-sub.Channel:
		if received.Type != "test_event" {
			t.Errorf("expected event type 'test_event', got %s", received.Type)
		}
		if received.Data["test"] != "data" {
			t.Errorf("expected data 'data', got %v", received.Data["test"])
		}
		if received.Timestamp.IsZero() {
			t.Error("timestamp was not set")
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestBroadcaster_SpecificEvents(t *testing.T) {
	b := NewBroadcaster()
	defer b.Close()

	sub := b.Subscribe("test_client")

	tests := []struct {
		name     string
		publish  func()
		expected string
	}{
		{
			name: "effect started",
			publish: func() {
				b.PublishEffectStarted("rainbow", 15)
			},
			expected: EventEffectStarted,
		},
		{
			name: "effect stopped",
			publish: func() {
				b.PublishEffectStopped("rainbow", "completed")
			},
			expected: EventEffectStopped,
		},
		{
			name: "progress",
			publish: func() {
				b.PublishProgress("rainbow", 5, 15)
			},
			expected: EventProgress,
		},
		{
			name: "dim changed",
			publish: func() {
				b.PublishDimChanged(128)
			},
			expected: EventDimChanged,
		},
		{
			name: "raw executed",
			publish: func() {
				b.PublishRawExecuted("effect=rainbow", "OK")
			},
			expected: EventRawExecuted,
		},
		{
			name: "button press",
			publish: func() {
				b.PublishButtonPress()
			},
			expected: EventButtonPress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.publish()

			select {
			case received := <-sub.Channel:
				if received.Type != tt.expected {
					t.Errorf("expected event type %s, got %s", tt.expected, received.Type)
				}
			case <-time.After(1 * time.Second):
				t.Error("timeout waiting for event")
			}
		})
	}
}

func TestBroadcaster_MultipleSubscribers(t *testing.T) {
	b := NewBroadcaster()
	defer b.Close()

	sub1 := b.Subscribe("client1")
	sub2 := b.Subscribe("client2")

	testEvent := Event{Type: "broadcast_test"}
	b.Publish(testEvent)

	// Both subscribers should receive the event
	for i, sub := range []*Subscriber{sub1, sub2} {
		select {
		case received := <-sub.Channel:
			if received.Type != "broadcast_test" {
				t.Errorf("subscriber %d: expected event type 'broadcast_test', got %s", i+1, received.Type)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("subscriber %d: timeout waiting for event", i+1)
		}
	}
}

func TestEvent_ToSSEData(t *testing.T) {
	event := Event{
		Type:      "test_event",
		Timestamp: time.Unix(1640995200, 0), // Fixed timestamp for testing
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	sseData := event.ToSSEData()

	if sseData[:6] != "data: " {
		t.Error("SSE data should start with 'data: '")
	}

	if sseData[len(sseData)-2:] != "\n\n" {
		t.Error("SSE data should end with double newline")
	}

	// Should contain the event type
	if !contains(sseData, "test_event") {
		t.Error("SSE data should contain event type")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
