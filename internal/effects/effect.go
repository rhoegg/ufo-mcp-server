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
	Duration    int    `json:"duration"`
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
			// File doesn't exist, initialize with seed effects
			s.initSeedEffects()
			return s.saveUnsafe()
		}
		return fmt.Errorf("reading effects file: %w", err)
	}

	var effects []*Effect
	if err := json.Unmarshal(data, &effects); err != nil {
		return fmt.Errorf("parsing effects JSON: %w", err)
	}

	s.effects = make(map[string]*Effect)
	for _, effect := range effects {
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
		effect.Duration = 10
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
		effect.Duration = 10
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

// initSeedEffects creates the initial set of effects
func (s *Store) initSeedEffects() {
	seedEffects := []*Effect{
		{
			Name:        "rainbow",
			Description: "Slow moving rainbow",
			Pattern:     "effect=rainbow",
			Duration:    15,
		},
		{
			Name:        "policeLights",
			Description: "Realistic police light bar with rotating red/blue",
			Pattern:     "top_init=1&bottom_init=1&top=10|1|ffffff&top=0|1|0000ff&top=1|1|000080&top=2|1|000040&top=3|1|000020&top=4|1|000010&top=5|1|000008&top=6|1|000004&top_whirl=252&bottom=4|1|ffffff&bottom=15|1|ff0000&bottom=14|1|800000&bottom=13|1|400000&bottom=12|1|200000&bottom=11|1|100000&bottom=10|1|080000&bottom=9|1|040000&bottom_whirl=250|ccw",
			Duration:    30,
		},
		{
			Name:        "breathingGreen",
			Description: "Fade in/out green",
			Pattern:     "top_init=1&bottom_init=1&top=00ff00&bottom=00ff00&top_morph=fade&bottom_morph=fade",
			Duration:    15,
		},
		{
			Name:        "pipelineDemo",
			Description: "Blog demo two-stage colours",
			Pattern:     "top_init=1&bottom_init=1&top=ffaa00&bottom=00aaff",
			Duration:    10,
		},
		{
			Name:        "ipDisplay",
			Description: "Spell IP address",
			Pattern:     "effect=ip",
			Duration:    30,
		},
	}

	for _, effect := range seedEffects {
		s.effects[effect.Name] = effect
	}
}
