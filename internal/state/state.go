package state

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/starspace46/ufo-mcp-go/internal/device"
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
	baseState   string // Base state pattern to restore when stack is empty
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
		LogoOn:        m.state.LogoOn,
		Effect:        m.state.Effect,
		Dim:           m.state.Dim,
		TopWhirlMs:    m.state.TopWhirlMs,
		BottomWhirlMs: m.state.BottomWhirlMs,
	}
	
	// Copy arrays
	copy(stateCopy.Top[:], m.state.Top[:])
	copy(stateCopy.Bottom[:], m.state.Bottom[:])
	
	// Copy morph data if present
	if m.state.TopMorph != nil {
		stateCopy.TopMorph = &MorphData{
			BrightnessMs: m.state.TopMorph.BrightnessMs,
			FadeMs:       m.state.TopMorph.FadeMs,
		}
	}
	if m.state.BottomMorph != nil {
		stateCopy.BottomMorph = &MorphData{
			BrightnessMs: m.state.BottomMorph.BrightnessMs,
			FadeMs:       m.state.BottomMorph.FadeMs,
		}
	}

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

// BuildStateQuery builds a complete UFO query string from current state
func (m *Manager) BuildStateQuery() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var parts []string
	
	// Add brightness
	parts = append(parts, fmt.Sprintf("dim=%d", m.state.Dim))
	
	// Build top ring
	parts = append(parts, "top_init=1")
	topSegments := m.buildRingSegments(m.state.Top)
	if topSegments != "" {
		parts = append(parts, fmt.Sprintf("top=%s", topSegments))
	}
	
	// Top ring animations
	if m.state.TopWhirlMs > 0 {
		parts = append(parts, fmt.Sprintf("top_whirl=%d", m.state.TopWhirlMs))
	}
	if m.state.TopMorph != nil {
		morphSpec := device.ConvertMorphToDevice((*device.MorphConfig)(m.state.TopMorph))
		parts = append(parts, fmt.Sprintf("top_morph=%s", morphSpec))
	}
	
	// Build bottom ring
	parts = append(parts, "bottom_init=1")
	bottomSegments := m.buildRingSegments(m.state.Bottom)
	if bottomSegments != "" {
		parts = append(parts, fmt.Sprintf("bottom=%s", bottomSegments))
	}
	
	// Bottom ring animations
	if m.state.BottomWhirlMs > 0 {
		parts = append(parts, fmt.Sprintf("bottom_whirl=%d", m.state.BottomWhirlMs))
	}
	if m.state.BottomMorph != nil {
		morphSpec := device.ConvertMorphToDevice((*device.MorphConfig)(m.state.BottomMorph))
		parts = append(parts, fmt.Sprintf("bottom_morph=%s", morphSpec))
	}
	
	// Logo
	if m.state.LogoOn {
		parts = append(parts, "logo=on")
	} else {
		parts = append(parts, "logo=off")
	}
	
	return strings.Join(parts, "&")
}

// buildRingSegments builds segment string from LED array
func (m *Manager) buildRingSegments(ring [15]string) string {
	var segments []string
	
	// Group consecutive LEDs with same color
	i := 0
	for i < 15 {
		color := ring[i]
		if color == "" || color == "000000" {
			i++
			continue
		}
		
		// Count consecutive LEDs with same color
		count := 1
		for j := i + 1; j < 15 && ring[j] == color; j++ {
			count++
		}
		
		segments = append(segments, fmt.Sprintf("%d|%d|%s", i, count, strings.ToUpper(color)))
		i += count
	}
	
	return strings.Join(segments, "|")
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

// SetBaseState updates the base state pattern
func (m *Manager) SetBaseState(pattern string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.baseState = pattern
}

// GetBaseState returns the current base state pattern
func (m *Manager) GetBaseState() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.baseState
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

	// Stack is empty, but we might have a base state to restore
	m.state.Effect = ""
	if m.baseState != "" {
		// Return a synthetic effect that represents the base state
		return &EffectStackItem{
			Name:    "__base_state__",
			Pattern: m.baseState,
			Context: map[string]interface{}{
				"synthetic": true,
				"isBase":    true,
			},
		}
	}
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
func (m *Manager) UpdateWhirl(ring string, whirlMs int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if ring == "top" {
		m.state.TopWhirlMs = whirlMs
	} else if ring == "bottom" {
		m.state.BottomWhirlMs = whirlMs
	}
}

// UpdateMorph updates the morph settings for a ring
func (m *Manager) UpdateMorph(ring string, morph *MorphData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if ring == "top" {
		m.state.TopMorph = morph
	} else if ring == "bottom" {
		m.state.BottomMorph = morph
	}
}