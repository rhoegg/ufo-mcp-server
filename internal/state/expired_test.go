package state

import (
	"testing"

	"github.com/starspace46/ufo-mcp-go/internal/events"
)

func TestMarkEffectExpired(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Push some effects
	manager.PushEffect("effect1", "pattern1", map[string]interface{}{"duration": 5000})
	manager.PushEffect("effect2", "pattern2", map[string]interface{}{"duration": 10000})
	manager.PushEffect("effect3", "pattern3", map[string]interface{}{"perpetual": true})

	// Mark effect2 as expired (it's in the middle of the stack)
	manager.MarkEffectExpired("effect2")

	// Verify effect2 is marked as expired
	// We need to access the internal state to check this
	manager.mu.RLock()
	found := false
	for _, effect := range manager.effectStack {
		if effect.Name == "effect2" && effect.Expired {
			found = true
			break
		}
	}
	manager.mu.RUnlock()

	if !found {
		t.Error("effect2 should be marked as expired")
	}

	// Verify other effects are not marked as expired
	manager.mu.RLock()
	for _, effect := range manager.effectStack {
		if effect.Name != "effect2" && effect.Expired {
			t.Errorf("effect %s should not be marked as expired", effect.Name)
		}
	}
	manager.mu.RUnlock()
}

func TestPopEffectSkipsExpired(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Push effects
	manager.PushEffect("effect1", "pattern1", map[string]interface{}{"duration": 5000})
	manager.PushEffect("effect2", "pattern2", map[string]interface{}{"duration": 10000})
	manager.PushEffect("effect3", "pattern3", map[string]interface{}{"perpetual": true})

	// Mark effect2 as expired
	manager.MarkEffectExpired("effect2")

	// Pop effect3 (top of stack)
	popped := manager.PopEffect()
	if popped == nil || popped.Name != "effect1" {
		t.Errorf("Expected to get effect1 after popping effect3, got %v", popped)
	}

	// Verify stack depth (should be 1, as expired effect2 was skipped)
	if manager.GetEffectStackDepth() != 1 {
		t.Errorf("Expected stack depth 1, got %d", manager.GetEffectStackDepth())
	}

	// Verify current effect is effect1
	current := manager.GetCurrentEffect()
	if current == nil || current.Name != "effect1" {
		t.Errorf("Expected current effect to be effect1, got %v", current)
	}
}

func TestPopEffectAllExpired(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Push effects
	manager.PushEffect("effect1", "pattern1", map[string]interface{}{"duration": 5000})
	manager.PushEffect("effect2", "pattern2", map[string]interface{}{"duration": 10000})
	manager.PushEffect("effect3", "pattern3", map[string]interface{}{"duration": 15000})

	// Mark all except the top as expired
	manager.MarkEffectExpired("effect1")
	manager.MarkEffectExpired("effect2")

	// Pop effect3
	popped := manager.PopEffect()
	if popped != nil {
		t.Errorf("Expected nil when all remaining effects are expired, got %v", popped)
	}

	// Verify stack is empty
	if manager.GetEffectStackDepth() != 0 {
		t.Errorf("Expected stack depth 0, got %d", manager.GetEffectStackDepth())
	}
}

func TestExpiredEffectWithBaseState(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Set a base state
	manager.SetBaseState("base_pattern")

	// Push effects
	manager.PushEffect("effect1", "pattern1", map[string]interface{}{"duration": 5000})
	manager.PushEffect("effect2", "pattern2", map[string]interface{}{"duration": 10000})

	// Mark effect1 as expired
	manager.MarkEffectExpired("effect1")

	// Pop effect2
	popped := manager.PopEffect()

	// Should get the base state since effect1 is expired
	if popped == nil || popped.Name != "__base_state__" {
		t.Errorf("Expected to get base state, got %v", popped)
	}
	if popped != nil && popped.Pattern != "base_pattern" {
		t.Errorf("Expected base pattern 'base_pattern', got %s", popped.Pattern)
	}
}

func TestComplexExpiredScenario(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Simulate: temporary, permanent, temporary (like the user's scenario)
	manager.PushEffect("pipelineDemo", "pattern1", map[string]interface{}{"duration": 10000})
	manager.PushEffect("breathingGreen", "pattern2", map[string]interface{}{"perpetual": true})
	manager.PushEffect("alertPulse", "pattern3", map[string]interface{}{"duration": 5000})

	// Mark the bottom temporary effect as expired (simulating its timer expiring)
	manager.MarkEffectExpired("pipelineDemo")

	// Pop the top effect (alertPulse)
	popped := manager.PopEffect()
	if popped == nil || popped.Name != "breathingGreen" {
		t.Errorf("Expected to get breathingGreen, got %v", popped)
	}

	// Stack should now have breathingGreen and the expired pipelineDemo (still in stack but expired)
	if manager.GetEffectStackDepth() != 2 {
		t.Errorf("Expected stack depth 2 (including expired), got %d", manager.GetEffectStackDepth())
	}

	// Pop again
	popped = manager.PopEffect()
	if popped != nil {
		t.Errorf("Expected nil (no more effects), got %v", popped)
	}

	// Stack should be empty
	if manager.GetEffectStackDepth() != 0 {
		t.Errorf("Expected stack depth 0, got %d", manager.GetEffectStackDepth())
	}
}

func TestMarkNonExistentEffect(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Push an effect
	manager.PushEffect("effect1", "pattern1", map[string]interface{}{})

	// Try to mark a non-existent effect as expired
	manager.MarkEffectExpired("nonexistent")

	// Verify effect1 is still not expired
	manager.mu.RLock()
	for _, effect := range manager.effectStack {
		if effect.Name == "effect1" && effect.Expired {
			t.Error("effect1 should not be marked as expired")
		}
	}
	manager.mu.RUnlock()
}

func TestExpiredEffectNotAtTop(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	manager := NewManager(broadcaster)

	// Push effects
	manager.PushEffect("effect1", "pattern1", map[string]interface{}{"duration": 5000})
	manager.PushEffect("effect2", "pattern2", map[string]interface{}{"perpetual": true})

	// Get current effect (should be effect2)
	current := manager.GetCurrentEffect()
	if current == nil || current.Name != "effect2" {
		t.Errorf("Expected current effect to be effect2, got %v", current)
	}

	// Mark effect1 as expired (it's not at the top)
	manager.MarkEffectExpired("effect1")

	// Current effect should still be effect2
	current = manager.GetCurrentEffect()
	if current == nil || current.Name != "effect2" {
		t.Errorf("Expected current effect to still be effect2, got %v", current)
	}

	// Stack depth should still be 2
	if manager.GetEffectStackDepth() != 2 {
		t.Errorf("Expected stack depth 2, got %d", manager.GetEffectStackDepth())
	}
}