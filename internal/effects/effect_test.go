package effects

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_AddUpdateDelete(t *testing.T) {
	// Create temporary file for testing
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_effects.json")

	store := NewStore(filePath)

	// Test Add
	effect := &Effect{
		Name:        "testEffect",
		Description: "A test effect",
		Pattern:     "test=pattern",
		Duration:    5,
	}

	err := store.Add(effect)
	if err != nil {
		t.Fatalf("failed to add effect: %v", err)
	}

	// Test Get
	retrieved, exists := store.Get("testEffect")
	if !exists {
		t.Fatal("effect not found after adding")
	}
	if retrieved.Name != effect.Name || retrieved.Pattern != effect.Pattern {
		t.Errorf("retrieved effect doesn't match added effect")
	}

	// Test duplicate name
	err = store.Add(effect)
	if err == nil {
		t.Error("expected error when adding duplicate effect")
	}

	// Test Update
	effect.Description = "Updated description"
	err = store.Update(effect)
	if err != nil {
		t.Fatalf("failed to update effect: %v", err)
	}

	retrieved, _ = store.Get("testEffect")
	if retrieved.Description != "Updated description" {
		t.Error("effect was not updated properly")
	}

	// Test Delete
	err = store.Delete("testEffect")
	if err != nil {
		t.Fatalf("failed to delete effect: %v", err)
	}

	_, exists = store.Get("testEffect")
	if exists {
		t.Error("effect still exists after deletion")
	}

	// Test delete non-existent
	err = store.Delete("nonExistent")
	if err == nil {
		t.Error("expected error when deleting non-existent effect")
	}
}

func TestStore_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_effects.json")

	// Create store and add effect
	store1 := NewStore(filePath)
	effect := &Effect{
		Name:        "loadSaveTest",
		Description: "Test load/save",
		Pattern:     "test=123",
		Duration:    7,
	}

	err := store1.Add(effect)
	if err != nil {
		t.Fatalf("failed to add effect: %v", err)
	}

	// Create new store and load
	store2 := NewStore(filePath)
	err = store2.Load()
	if err != nil {
		t.Fatalf("failed to load effects: %v", err)
	}

	// Verify effect was loaded
	retrieved, exists := store2.Get("loadSaveTest")
	if !exists {
		t.Fatal("effect not found after loading")
	}
	if retrieved.Pattern != "test=123" {
		t.Errorf("expected pattern 'test=123', got %s", retrieved.Pattern)
	}
}

func TestStore_SeedEffects(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "seed_effects.json")

	store := NewStore(filePath)

	// Load should create seed effects when file doesn't exist
	err := store.Load()
	if err != nil {
		t.Fatalf("failed to load seed effects: %v", err)
	}

	// Verify seed effects exist
	effects := store.List()
	if len(effects) == 0 {
		t.Error("no seed effects were created")
	}

	// Check for specific seed effects
	expectedEffects := []string{"rainbow", "policeLights", "breathingGreen", "pipelineDemo", "ipDisplay"}
	for _, name := range expectedEffects {
		if _, exists := store.Get(name); !exists {
			t.Errorf("seed effect '%s' not found", name)
		}
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("effects file was not created")
	}
}

func TestEffect_DefaultDuration(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "duration_test.json")

	store := NewStore(filePath)

	// Test effect without duration gets default
	effect := &Effect{
		Name:        "noDuration",
		Description: "No duration set",
		Pattern:     "test=noduration",
		Duration:    0, // No duration set
	}

	err := store.Add(effect)
	if err != nil {
		t.Fatalf("failed to add effect: %v", err)
	}

	retrieved, _ := store.Get("noDuration")
	if retrieved.Duration != 10 {
		t.Errorf("expected default duration 10, got %d", retrieved.Duration)
	}
}
