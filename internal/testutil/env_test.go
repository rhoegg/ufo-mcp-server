package testutil

import (
	"os"
	"testing"
)

func TestWithCleanEnv(t *testing.T) {
	// Test that UFO_IP is unset within the function
	os.Setenv("UFO_IP", "should_be_removed")
	
	WithCleanEnv(t, func() {
		if val := os.Getenv("UFO_IP"); val != "" {
			t.Errorf("UFO_IP should be unset within WithCleanEnv, got %s", val)
		}
	})
}

func TestWithUFOIP(t *testing.T) {
	// Test that UFO_IP is set to the specified value
	WithUFOIP(t, "test_value", func() {
		if val := os.Getenv("UFO_IP"); val != "test_value" {
			t.Errorf("UFO_IP should be 'test_value', got %s", val)
		}
	})
}

func TestSetupTestEnv(t *testing.T) {
	// Test the setup function
	setIP, _ := SetupTestEnv(t)
	
	// Test setting different values
	setIP("value1")
	if val := os.Getenv("UFO_IP"); val != "value1" {
		t.Errorf("UFO_IP should be 'value1', got %s", val)
	}
	
	setIP("value2")
	if val := os.Getenv("UFO_IP"); val != "value2" {
		t.Errorf("UFO_IP should be 'value2', got %s", val)
	}
}

// Test that cleanup works correctly by using subtests
func TestCleanupBehavior(t *testing.T) {
	originalValue := os.Getenv("UFO_IP")
	defer func() {
		// Restore original value after all tests
		if originalValue != "" {
			os.Setenv("UFO_IP", originalValue)
		} else {
			os.Unsetenv("UFO_IP")
		}
	}()
	
	t.Run("WithCleanEnv restores original", func(t *testing.T) {
		os.Setenv("UFO_IP", "before_test")
		WithCleanEnv(t, func() {
			// UFO_IP is unset here
		})
		// After this subtest completes, cleanup should restore "before_test"
	})
	
	// Check that the value was restored
	if val := os.Getenv("UFO_IP"); val != "before_test" {
		t.Errorf("UFO_IP should be restored to 'before_test' after WithCleanEnv, got %s", val)
	}
	
	t.Run("WithUFOIP restores original", func(t *testing.T) {
		os.Setenv("UFO_IP", "before_test2")
		WithUFOIP(t, "during_test", func() {
			// UFO_IP is "during_test" here
		})
		// After this subtest completes, cleanup should restore "before_test2"
	})
	
	// Check that the value was restored
	if val := os.Getenv("UFO_IP"); val != "before_test2" {
		t.Errorf("UFO_IP should be restored to 'before_test2' after WithUFOIP, got %s", val)
	}
}