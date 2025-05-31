package state

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/starspace46/ufo-mcp-go/internal/events"
)

func TestNewManager(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Check initial state
	state := manager.Snapshot()
	if state.LogoOn {
		t.Error("Logo should be off initially")
	}
	if state.Effect != "" {
		t.Error("No effect should be running initially")
	}
	if state.Dim != 255 {
		t.Errorf("Expected brightness 255, got %d", state.Dim)
	}

	// Check all LEDs are off
	for i := 0; i < 15; i++ {
		if state.Top[i] != "000000" {
			t.Errorf("Top LED %d should be off, got %s", i, state.Top[i])
		}
		if state.Bottom[i] != "000000" {
			t.Errorf("Bottom LED %d should be off, got %s", i, state.Bottom[i])
		}
	}
}

func TestUpdateBrightness(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Subscribe to events
	sub := broadcaster.Subscribe("test")
	defer broadcaster.Unsubscribe("test")

	// Update brightness
	manager.UpdateBrightness(128)

	// Check state
	state := manager.Snapshot()
	if state.Dim != 128 {
		t.Errorf("Expected brightness 128, got %d", state.Dim)
	}

	// Check event was published (with timeout)
	select {
	case event := <-sub.Channel:
		if event.Type != events.EventDimChanged {
			t.Errorf("Expected dim_changed event, got %s", event.Type)
		}
		if level, ok := event.Data["level"].(int); !ok || level != 128 {
			t.Errorf("Expected level 128 in event, got %v", event.Data["level"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected dim_changed event, got none")
	}
}

func TestUpdateLogo(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Turn logo on
	manager.UpdateLogo(true)
	state := manager.Snapshot()
	if !state.LogoOn {
		t.Error("Logo should be on")
	}

	// Turn logo off
	manager.UpdateLogo(false)
	state = manager.Snapshot()
	if state.LogoOn {
		t.Error("Logo should be off")
	}
}

func TestUpdateRingSegments(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Subscribe to events
	sub := broadcaster.Subscribe("test")
	defer broadcaster.Unsubscribe("test")

	// Update top ring with specific colors
	segments := []string{"ff0000", "00ff00", "0000ff"}
	manager.UpdateRingSegments("top", segments, "")

	// Check state
	state := manager.Snapshot()
	if state.Top[0] != "ff0000" {
		t.Errorf("Expected top[0] to be ff0000, got %s", state.Top[0])
	}
	if state.Top[1] != "00ff00" {
		t.Errorf("Expected top[1] to be 00ff00, got %s", state.Top[1])
	}
	if state.Top[2] != "0000ff" {
		t.Errorf("Expected top[2] to be 0000ff, got %s", state.Top[2])
	}

	// Check event was published (with timeout)
	select {
	case event := <-sub.Channel:
		if event.Type != events.EventRingUpdate {
			t.Errorf("Expected ring_update event, got %s", event.Type)
		}
		if ring, ok := event.Data["ring"].(string); !ok || ring != "top" {
			t.Errorf("Expected ring 'top' in event, got %v", event.Data["ring"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected ring_update event, got none")
	}
}

func TestUpdateRingWithBackground(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Update bottom ring with background color
	segments := []string{"ff0000", "", "0000ff"} // Empty segment should use background
	background := "00ff00"
	manager.UpdateRingSegments("bottom", segments, background)

	// Check state
	state := manager.Snapshot()
	if state.Bottom[0] != "ff0000" {
		t.Errorf("Expected bottom[0] to be ff0000, got %s", state.Bottom[0])
	}
	if state.Bottom[1] != "00ff00" {
		t.Errorf("Expected bottom[1] to be 00ff00 (background), got %s", state.Bottom[1])
	}
	if state.Bottom[2] != "0000ff" {
		t.Errorf("Expected bottom[2] to be 0000ff, got %s", state.Bottom[2])
	}
	// Check that background filled remaining LEDs
	for i := 3; i < 15; i++ {
		if state.Bottom[i] != "00ff00" {
			t.Errorf("Expected bottom[%d] to be 00ff00 (background), got %s", i, state.Bottom[i])
		}
	}
}

func TestEffectManagement(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Set effect
	manager.UpdateEffect("rainbow")
	state := manager.Snapshot()
	if state.Effect != "rainbow" {
		t.Errorf("Expected effect 'rainbow', got %s", state.Effect)
	}

	// Clear effect
	manager.ClearEffect()
	state = manager.Snapshot()
	if state.Effect != "" {
		t.Errorf("Expected no effect, got %s", state.Effect)
	}
}

func TestReset(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Set some state
	manager.UpdateRingSegments("top", []string{"ff0000", "00ff00", "0000ff"}, "")
	manager.UpdateLogo(true)
	manager.UpdateEffect("test")
	manager.UpdateBrightness(100)

	// Reset
	manager.Reset()

	// Check state
	state := manager.Snapshot()
	if state.LogoOn {
		t.Error("Logo should be off after reset")
	}
	if state.Effect != "" {
		t.Error("Effect should be cleared after reset")
	}
	if state.Dim != 100 {
		t.Error("Brightness should not change after reset")
	}

	// Check all LEDs are off
	for i := 0; i < 15; i++ {
		if state.Top[i] != "000000" {
			t.Errorf("Top LED %d should be off after reset, got %s", i, state.Top[i])
		}
		if state.Bottom[i] != "000000" {
			t.Errorf("Bottom LED %d should be off after reset, got %s", i, state.Bottom[i])
		}
	}
}

func TestToJSON(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Set some state
	manager.UpdateRingSegments("top", []string{"ff0000"}, "")
	manager.UpdateLogo(true)
	manager.UpdateBrightness(200)

	// Get JSON
	jsonStr, err := manager.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	// Parse JSON to verify
	var state LedState
	if err := json.Unmarshal([]byte(jsonStr), &state); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if !state.LogoOn {
		t.Error("Logo should be on in JSON")
	}
	if state.Dim != 200 {
		t.Errorf("Expected brightness 200 in JSON, got %d", state.Dim)
	}
	if state.Top[0] != "ff0000" {
		t.Errorf("Expected top[0] to be ff0000 in JSON, got %s", state.Top[0])
	}
}

func TestConcurrency(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Run concurrent updates
	done := make(chan bool, 4)

	go func() {
		for i := 0; i < 100; i++ {
			manager.UpdateBrightness(i % 256)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			manager.UpdateLogo(i%2 == 0)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			manager.UpdateEffect("test")
			manager.ClearEffect()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = manager.Snapshot()
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 4; i++ {
		<-done
	}

	// If we get here without deadlock or panic, concurrency is working
}