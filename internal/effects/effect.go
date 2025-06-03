package effects

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Effect represents a lighting effect configuration
type Effect struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"`
	Duration    int    `json:"duration"` // Duration in milliseconds (was seconds in v1)
	Perpetual   bool   `json:"perpetual"`
}

// Store manages the collection of lighting effects
type Store struct {
	mu      sync.RWMutex
	effects map[string]*Effect
	file    string
}

// NewStore creates a new effect store
func NewStore(filePath string) *Store {
	return &Store{
		effects: make(map[string]*Effect),
		file:    filePath,
	}
}

// Load reads effects from the JSON file
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, try to copy from default location
			if err := s.copyDefaultEffects(); err != nil {
				// If that fails, create empty effects
				s.effects = make(map[string]*Effect)
				return s.saveUnsafe()
			}
			// Load the copied file - need to unlock first to avoid deadlock
			s.mu.Unlock()
			err := s.Load()
			s.mu.Lock()
			return err
		}
		return fmt.Errorf("reading effects file: %w", err)
	}

	var effects []*Effect
	if err := json.Unmarshal(data, &effects); err != nil {
		return fmt.Errorf("parsing effects JSON: %w", err)
	}

	s.effects = make(map[string]*Effect)
	for _, effect := range effects {
		// Migration: if duration seems to be in seconds (< 1000), convert to milliseconds
		if effect.Duration > 0 && effect.Duration < 1000 {
			effect.Duration *= 1000
		}
		s.effects[effect.Name] = effect
	}

	return nil
}

// Save writes effects to the JSON file
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnsafe()
}

// saveUnsafe saves without acquiring lock (internal use)
func (s *Store) saveUnsafe() error {
	effects := make([]*Effect, 0, len(s.effects))
	for _, effect := range s.effects {
		effects = append(effects, effect)
	}

	data, err := json.MarshalIndent(effects, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling effects: %w", err)
	}

	// Ensure directory exists - get directory from file path
	dir := filepath.Dir(s.file)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	return os.WriteFile(s.file, data, 0644)
}

// List returns all effects
func (s *Store) List() []*Effect {
	s.mu.RLock()
	defer s.mu.RUnlock()

	effects := make([]*Effect, 0, len(s.effects))
	for _, effect := range s.effects {
		effects = append(effects, effect)
	}
	return effects
}

// Get retrieves an effect by name
func (s *Store) Get(name string) (*Effect, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	effect, exists := s.effects[name]
	return effect, exists
}

// Add creates a new effect
func (s *Store) Add(effect *Effect) error {
	if effect.Name == "" {
		return fmt.Errorf("effect name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.effects[effect.Name]; exists {
		return fmt.Errorf("effect with name '%s' already exists", effect.Name)
	}

	// Set default duration if not specified
	if effect.Duration <= 0 {
		effect.Duration = 10000 // 10 seconds in milliseconds
	}

	s.effects[effect.Name] = effect
	return s.saveUnsafe()
}

// Update modifies an existing effect
func (s *Store) Update(effect *Effect) error {
	if effect.Name == "" {
		return fmt.Errorf("effect name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.effects[effect.Name]; !exists {
		return fmt.Errorf("effect with name '%s' does not exist", effect.Name)
	}

	// Set default duration if not specified
	if effect.Duration <= 0 {
		effect.Duration = 10000 // 10 seconds in milliseconds
	}

	s.effects[effect.Name] = effect
	return s.saveUnsafe()
}

// Delete removes an effect by name
func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.effects[name]; !exists {
		return fmt.Errorf("effect with name '%s' does not exist", name)
	}

	delete(s.effects, name)
	return s.saveUnsafe()
}

// copyDefaultEffects tries to copy the default effects.json from various locations
func (s *Store) copyDefaultEffects() error {
	// Try different locations where the default effects.json might be
	possiblePaths := []string{
		"/etc/ufo-mcp/effects.json",
		"/usr/share/ufo-mcp/effects.json",
		"./data/effects.json",
		"../data/effects.json",
		"../../data/effects.json",
	}

	for _, path := range possiblePaths {
		if data, err := os.ReadFile(path); err == nil {
			// Ensure directory exists
			dir := filepath.Dir(s.file)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("creating effects directory: %w", err)
			}
			
			// Copy the file
			if err := os.WriteFile(s.file, data, 0644); err != nil {
				return fmt.Errorf("writing effects file: %w", err)
			}
			
			return nil
		}
	}

	return fmt.Errorf("default effects.json not found in any standard location")
}
