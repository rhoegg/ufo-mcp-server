package state

import (
	"encoding/json"
	"strings"
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

func TestEffectStack(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Test 1: Push and pop single effect
	t.Run("SingleEffect", func(t *testing.T) {
		manager.PushEffect("rainbow", "top=rainbow&bottom=rainbow", map[string]interface{}{
			"perpetual": true,
		})

		// Check current effect
		current := manager.GetCurrentEffect()
		if current == nil || current.Name != "rainbow" {
			t.Error("Expected rainbow effect to be current")
		}
		if manager.GetEffectStackDepth() != 1 {
			t.Errorf("Expected stack depth 1, got %d", manager.GetEffectStackDepth())
		}

		// Pop effect
		popped := manager.PopEffect()
		if popped != nil {
			t.Error("Expected nil when popping last effect")
		}
		if manager.GetEffectStackDepth() != 0 {
			t.Errorf("Expected stack depth 0, got %d", manager.GetEffectStackDepth())
		}
	})

	// Test 2: Multiple effects stack
	t.Run("MultipleEffects", func(t *testing.T) {
		// Clear stack first
		for manager.GetEffectStackDepth() > 0 {
			manager.PopEffect()
		}

		// Push multiple effects
		manager.PushEffect("ocean", "pattern1", map[string]interface{}{"perpetual": true})
		manager.PushEffect("alert", "pattern2", map[string]interface{}{"duration": 5000})
		manager.PushEffect("fire", "pattern3", map[string]interface{}{"perpetual": true})

		if manager.GetEffectStackDepth() != 3 {
			t.Errorf("Expected stack depth 3, got %d", manager.GetEffectStackDepth())
		}

		// Current should be fire
		current := manager.GetCurrentEffect()
		if current == nil || current.Name != "fire" {
			t.Error("Expected fire effect to be current")
		}

		// Pop fire, should return to alert
		popped := manager.PopEffect()
		if popped == nil || popped.Name != "alert" {
			t.Error("Expected alert effect after popping fire")
		}

		// Pop alert, should return to ocean
		popped = manager.PopEffect()
		if popped == nil || popped.Name != "ocean" {
			t.Error("Expected ocean effect after popping alert")
		}

		// Pop ocean, should return nil
		popped = manager.PopEffect()
		if popped != nil {
			t.Error("Expected nil after popping last effect")
		}
	})

	// Test 3: Base state restoration
	t.Run("BaseStateRestoration", func(t *testing.T) {
		// Clear stack
		for manager.GetEffectStackDepth() > 0 {
			manager.PopEffect()
		}

		// Set base state
		manager.SetBaseState("top_init=1&bottom_init=1&top=0|15|00ff00")

		// Push an effect
		manager.PushEffect("temp", "pattern", map[string]interface{}{})

		// Pop the effect - should return base state
		popped := manager.PopEffect()
		if popped == nil {
			t.Error("Expected base state to be returned")
		}
		if popped.Name != "__base_state__" {
			t.Errorf("Expected __base_state__, got %s", popped.Name)
		}
		if popped.Pattern != "top_init=1&bottom_init=1&top=0|15|00ff00" {
			t.Errorf("Expected base state pattern, got %s", popped.Pattern)
		}
	})

	// Test 4: Synthetic effects (configureLighting)
	t.Run("SyntheticEffects", func(t *testing.T) {
		// Clear stack and base state
		for manager.GetEffectStackDepth() > 0 {
			manager.PopEffect()
		}
		manager.SetBaseState("")

		// Simulate sequence: playEffect X, configureLighting Y, configureLighting Z, playEffect A
		manager.PushEffect("X", "patternX", map[string]interface{}{"perpetual": true})
		manager.PushEffect("__config__", "patternY", map[string]interface{}{"synthetic": true})
		manager.PushEffect("__config__", "patternZ", map[string]interface{}{"synthetic": true})
		manager.PushEffect("A", "patternA", map[string]interface{}{"duration": 5000})

		if manager.GetEffectStackDepth() != 4 {
			t.Errorf("Expected stack depth 4, got %d", manager.GetEffectStackDepth())
		}

		// Pop A, should return to Z
		popped := manager.PopEffect()
		if popped == nil || popped.Pattern != "patternZ" {
			t.Error("Expected to return to configureLighting Z")
		}

		// Pop Z, should return to Y
		popped = manager.PopEffect()
		if popped == nil || popped.Pattern != "patternY" {
			t.Error("Expected to return to configureLighting Y")
		}

		// Pop Y, should return to X
		popped = manager.PopEffect()
		if popped == nil || popped.Name != "X" {
			t.Error("Expected to return to effect X")
		}
	})
}

func TestBuildStateQuery(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	t.Run("EmptyState", func(t *testing.T) {
		manager := NewManager(broadcaster)
		query := manager.BuildStateQuery()
		
		// Should have default values
		if !strings.Contains(query, "dim=255") {
			t.Error("Expected dim=255 in query")
		}
		if !strings.Contains(query, "top_init=1") {
			t.Error("Expected top_init=1 in query")
		}
		if !strings.Contains(query, "bottom_init=1") {
			t.Error("Expected bottom_init=1 in query")
		}
		if !strings.Contains(query, "logo=off") {
			t.Error("Expected logo=off in query")
		}
	})

	t.Run("WithSegments", func(t *testing.T) {
		manager := NewManager(broadcaster)
		
		// Set some LEDs
		colors := []string{"FF0000", "FF0000", "00FF00", "00FF00", "00FF00"}
		manager.UpdateRingSegments("top", colors, "")
		
		query := manager.BuildStateQuery()
		
		// Should have segments
		if !strings.Contains(query, "top=0|2|FF0000|2|3|00FF00") {
			t.Errorf("Expected segment pattern, got query: %s", query)
		}
	})

	t.Run("WithWhirlAndMorph", func(t *testing.T) {
		manager := NewManager(broadcaster)
		
		// Set animations
		manager.UpdateWhirl("top", 300)
		manager.UpdateMorph("bottom", &MorphData{
			BrightnessMs: 1000,
			FadeMs:       333,
		})
		
		query := manager.BuildStateQuery()
		
		// Should have whirl
		if !strings.Contains(query, "top_whirl=300") {
			t.Errorf("Expected top_whirl=300, got query: %s", query)
		}
		
		// Should have morph (150|10 based on conversion)
		if !strings.Contains(query, "bottom_morph=150|10") {
			t.Errorf("Expected bottom_morph=150|10, got query: %s", query)
		}
	})

	t.Run("CompleteState", func(t *testing.T) {
		manager := NewManager(broadcaster)
		
		// Set a complete state
		manager.UpdateBrightness(128)
		manager.UpdateLogo(true)
		manager.UpdateRingSegments("top", []string{"FF0000", "00FF00", "0000FF"}, "")
		manager.UpdateRingSegments("bottom", []string{"FFFF00"}, "")
		manager.UpdateWhirl("top", 200)
		manager.UpdateWhirl("bottom", 250)
		manager.UpdateMorph("top", &MorphData{BrightnessMs: 500, FadeMs: 1000})
		
		query := manager.BuildStateQuery()
		
		// Verify all components
		expectedParts := []string{
			"dim=128",
			"top_init=1",
			"top=0|1|FF0000|1|1|00FF00|2|1|0000FF",
			"top_whirl=200",
			"top_morph=75|3",  // 500ms/6.67=75 ticks, 3333/1000=3.3≈3 speed
			"bottom_init=1",
			"bottom=0|1|FFFF00",
			"bottom_whirl=250",
			"logo=on",
		}
		
		for _, part := range expectedParts {
			if !strings.Contains(query, part) {
				t.Errorf("Expected '%s' in query: %s", part, query)
			}
		}
	})

	t.Run("ConsecutiveColors", func(t *testing.T) {
		manager := NewManager(broadcaster)
		
		// Set consecutive same colors
		colors := []string{"FF0000", "FF0000", "FF0000", "00FF00", "00FF00"}
		manager.UpdateRingSegments("top", colors, "")
		
		query := manager.BuildStateQuery()
		
		// Should group consecutive colors
		if !strings.Contains(query, "top=0|3|FF0000|3|2|00FF00") {
			t.Errorf("Expected grouped segments, got query: %s", query)
		}
	})
}

func TestUpdateWhirl(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()
	
	manager := NewManager(broadcaster)
	
	// Test top ring whirl
	manager.UpdateWhirl("top", 300)
	state := manager.Snapshot()
	if state.TopWhirlMs != 300 {
		t.Errorf("Expected TopWhirlMs=300, got %d", state.TopWhirlMs)
	}
	
	// Test bottom ring whirl
	manager.UpdateWhirl("bottom", 450)
	state = manager.Snapshot()
	if state.BottomWhirlMs != 450 {
		t.Errorf("Expected BottomWhirlMs=450, got %d", state.BottomWhirlMs)
	}
	
	// Test invalid ring name (should be ignored)
	manager.UpdateWhirl("invalid", 100)
	// No error expected, just ignored
}

func TestUpdateMorph(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()
	
	manager := NewManager(broadcaster)
	
	// Test top ring morph
	topMorph := &MorphData{
		BrightnessMs: 1000,
		FadeMs:       500,
	}
	manager.UpdateMorph("top", topMorph)
	state := manager.Snapshot()
	
	if state.TopMorph == nil {
		t.Fatal("Expected TopMorph to be set")
	}
	if state.TopMorph.BrightnessMs != 1000 {
		t.Errorf("Expected TopMorph.BrightnessMs=1000, got %d", state.TopMorph.BrightnessMs)
	}
	if state.TopMorph.FadeMs != 500 {
		t.Errorf("Expected TopMorph.FadeMs=500, got %d", state.TopMorph.FadeMs)
	}
	
	// Test bottom ring morph
	bottomMorph := &MorphData{
		BrightnessMs: 2000,
		FadeMs:       333,
	}
	manager.UpdateMorph("bottom", bottomMorph)
	state = manager.Snapshot()
	
	if state.BottomMorph == nil {
		t.Fatal("Expected BottomMorph to be set")
	}
	if state.BottomMorph.BrightnessMs != 2000 {
		t.Errorf("Expected BottomMorph.BrightnessMs=2000, got %d", state.BottomMorph.BrightnessMs)
	}
}
