package state

import (
	"encoding/json"
	"sync"

	"github.com/starspace46/ufo-mcp-go/internal/events"
)

// LedState represents the current state of all LEDs on the UFO
type LedState struct {
	Top    [15]string `json:"top"`    // hex colors for top ring
	Bottom [15]string `json:"bottom"` // hex colors for bottom ring
	LogoOn bool       `json:"logoOn"` // logo LED state
	Effect string     `json:"effect"` // currently running effect name
	Dim    int        `json:"dim"`    // brightness level 0-255
	
	// Animation state in milliseconds
	TopWhirlMs    int        `json:"topWhirlMs,omitempty"`    // top ring rotation speed in ms
	BottomWhirlMs int        `json:"bottomWhirlMs,omitempty"` // bottom ring rotation speed in ms
	TopMorph      *MorphData `json:"topMorph,omitempty"`      // top ring morph settings
	BottomMorph   *MorphData `json:"bottomMorph,omitempty"`   // bottom ring morph settings
}

// MorphData represents morph animation settings in milliseconds
type MorphData struct {
	BrightnessMs int `json:"brightnessMs"` // duration at full brightness
	FadeMs       int `json:"fadeMs"`       // fade transition duration
}

// EffectStackItem represents an effect in the stack
type EffectStackItem struct {
	Name      string                 // Effect name
	Pattern   string                 // Effect pattern
	Context   map[string]interface{} // Additional context (duration, perpetual, etc)
}

// Manager manages the shadow LED state with thread-safe operations
type Manager struct {
	mu          sync.RWMutex
	state       *LedState
	effectStack []EffectStackItem
	broadcaster *events.Broadcaster
}

// NewManager creates a new state manager
func NewManager(broadcaster *events.Broadcaster) *Manager {
	// Initialize with default state (all LEDs off)
	defaultState := &LedState{
		Top:    [15]string{},
		Bottom: [15]string{},
		LogoOn: false,
		Effect: "",
		Dim:    255, // Default to full brightness
	}

	// Initialize all LEDs to black (off)
	for i := 0; i < 15; i++ {
		defaultState.Top[i] = "000000"
		defaultState.Bottom[i] = "000000"
	}

	return &Manager{
		state:       defaultState,
		broadcaster: broadcaster,
	}
}

// Snapshot returns a copy of the current LED state
func (m *Manager) Snapshot() *LedState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a deep copy
	stateCopy := &LedState{
		LogoOn: m.state.LogoOn,
		Effect: m.state.Effect,
		Dim:    m.state.Dim,
	}

	// Copy arrays
	copy(stateCopy.Top[:], m.state.Top[:])
	copy(stateCopy.Bottom[:], m.state.Bottom[:])

	return stateCopy
}

// UpdateBrightness updates the brightness level
func (m *Manager) UpdateBrightness(level int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state.Dim != level {
		m.state.Dim = level
		m.broadcaster.PublishDimChanged(level)
	}
}

// UpdateLogo updates the logo LED state
func (m *Manager) UpdateLogo(on bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.LogoOn = on
}

// UpdateRingSegments updates specific segments on a ring
func (m *Manager) UpdateRingSegments(ring string, segments []string, background string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var targetRing *[15]string
	if ring == "top" {
		targetRing = &m.state.Top
	} else if ring == "bottom" {
		targetRing = &m.state.Bottom
	} else {
		return // Invalid ring name
	}

	// If background color is specified, fill all LEDs with it first
	if background != "" {
		for i := 0; i < 15; i++ {
			targetRing[i] = background
		}
	}

	// Apply segment colors
	for i, color := range segments {
		if i < 15 && color != "" {
			targetRing[i] = color
		}
	}

	// Emit ring update event
	m.broadcaster.PublishRingUpdate(ring, map[string]interface{}{
		"segments":   segments,
		"background": background,
	})
}

// UpdateEffect updates the currently running effect
func (m *Manager) UpdateEffect(effectName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.Effect = effectName
}

// ClearEffect clears the currently running effect
func (m *Manager) ClearEffect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.Effect = ""
}

// Reset resets all LEDs to off state
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Turn off all LEDs
	for i := 0; i < 15; i++ {
		m.state.Top[i] = "000000"
		m.state.Bottom[i] = "000000"
	}
	m.state.LogoOn = false
	m.state.Effect = ""
	// Keep current brightness level

	// Emit events for the reset
	m.broadcaster.PublishRingUpdate("top", map[string]interface{}{
		"reset": true,
	})
	m.broadcaster.PublishRingUpdate("bottom", map[string]interface{}{
		"reset": true,
	})
}

// SetActiveEffect updates the currently running effect
func (m *Manager) SetActiveEffect(effectName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.Effect = effectName
}

// ToJSON serializes the current state to JSON
func (m *Manager) ToJSON() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.Marshal(m.state)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// PushEffect pushes a new effect onto the stack
func (m *Manager) PushEffect(name, pattern string, context map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add to stack
	m.effectStack = append(m.effectStack, EffectStackItem{
		Name:    name,
		Pattern: pattern,
		Context: context,
	})

	// Update current effect
	m.state.Effect = name
}

// PopEffect removes the current effect from the stack and returns the new current effect
func (m *Manager) PopEffect() *EffectStackItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.effectStack) == 0 {
		m.state.Effect = ""
		return nil
	}

	// Remove the top effect
	m.effectStack = m.effectStack[:len(m.effectStack)-1]

	// Update current effect to the new top of stack
	if len(m.effectStack) > 0 {
		current := &m.effectStack[len(m.effectStack)-1]
		m.state.Effect = current.Name
		return current
	}

	m.state.Effect = ""
	return nil
}

// GetCurrentEffect returns the current effect from the top of the stack
func (m *Manager) GetCurrentEffect() *EffectStackItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.effectStack) == 0 {
		return nil
	}

	current := m.effectStack[len(m.effectStack)-1]
	return &current
}

// GetEffectStackDepth returns the number of effects on the stack
func (m *Manager) GetEffectStackDepth() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.effectStack)
}

// UpdateTopRing updates all LEDs on the top ring
func (m *Manager) UpdateTopRing(colors []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update top ring colors
	for i := 0; i < 15 && i < len(colors); i++ {
		if colors[i] == "" {
			m.state.Top[i] = "000000" // Default to black
		} else {
			m.state.Top[i] = colors[i]
		}
	}

	// Fill remaining LEDs with black if colors array is shorter
	for i := len(colors); i < 15; i++ {
		m.state.Top[i] = "000000"
	}

	// Emit ring update event
	m.broadcaster.PublishRingUpdate("top", map[string]interface{}{
		"colors": colors,
	})
}

// UpdateBottomRing updates all LEDs on the bottom ring
func (m *Manager) UpdateBottomRing(colors []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update bottom ring colors
	for i := 0; i < 15 && i < len(colors); i++ {
		if colors[i] == "" {
			m.state.Bottom[i] = "000000" // Default to black
		} else {
			m.state.Bottom[i] = colors[i]
		}
	}

	// Fill remaining LEDs with black if colors array is shorter
	for i := len(colors); i < 15; i++ {
		m.state.Bottom[i] = "000000"
	}

	// Emit ring update event
	m.broadcaster.PublishRingUpdate("bottom", map[string]interface{}{
		"colors": colors,
	})
}

// ParseRingCommand parses a ring pattern command and updates state accordingly
// This handles patterns like "top_init=1&top=ff0000|00ff00|0000ff&top_bg=ffffff"
func (m *Manager) ParseRingCommand(ring string, query string) {
	// This is a simplified parser - in production you'd want more robust parsing
	// For now, we'll just handle basic color updates
	
	// Extract segments from the query
	segments := extractSegments(query, ring)
	background := extractBackground(query, ring)
	
	if len(segments) > 0 || background != "" {
		m.UpdateRingSegments(ring, segments, background)
	}
}

// Helper function to extract segment colors from query
func extractSegments(query, ring string) []string {
	// This is a placeholder - implement proper parsing based on UFO API format
	// For now, return empty to avoid errors
	return []string{}
}

// Helper function to extract background color from query
func extractBackground(query, ring string) string {
	// This is a placeholder - implement proper parsing based on UFO API format
	return ""
}

// UpdateWhirl updates the whirl (rotation) speed for a ring
func (m *Manager) UpdateWhirl(ring string, speedMs int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ring == "top" {
		m.state.TopWhirlMs = speedMs
	} else if ring == "bottom" {
		m.state.BottomWhirlMs = speedMs
	}
}

// UpdateMorph updates the morph settings for a ring
func (m *Manager) UpdateMorph(ring string, brightnessMs, fadeMs int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	morphData := &MorphData{
		BrightnessMs: brightnessMs,
		FadeMs:       fadeMs,
	}

	if ring == "top" {
		m.state.TopMorph = morphData
	} else if ring == "bottom" {
		m.state.BottomMorph = morphData
	}
}

// ClearAnimations clears all animation settings
func (m *Manager) ClearAnimations() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.TopWhirlMs = 0
	m.state.BottomWhirlMs = 0
	m.state.TopMorph = nil
	m.state.BottomMorph = nil
}