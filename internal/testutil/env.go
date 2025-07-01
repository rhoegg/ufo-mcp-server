package testutil

import (
	"os"
	"testing"
)

// WithCleanEnv runs a test function with UFO_IP unset and restores the original value afterwards
func WithCleanEnv(t *testing.T, testFunc func()) {
	t.Helper()
	originalIP := os.Getenv("UFO_IP")
	os.Unsetenv("UFO_IP")
	t.Cleanup(func() {
		if originalIP != "" {
			os.Setenv("UFO_IP", originalIP)
		}
	})
	testFunc()
}

// WithUFOIP runs a test function with UFO_IP set to the specified value and restores the original afterwards
func WithUFOIP(t *testing.T, ip string, testFunc func()) {
	t.Helper()
	originalIP := os.Getenv("UFO_IP")
	os.Setenv("UFO_IP", ip)
	t.Cleanup(func() {
		if originalIP != "" {
			os.Setenv("UFO_IP", originalIP)
		} else {
			os.Unsetenv("UFO_IP")
		}
	})
	testFunc()
}

// SetupTestEnv is a test helper that saves the current UFO_IP and returns a cleanup function.
// This is useful when you need more control over when the environment is set/unset.
func SetupTestEnv(t *testing.T) (setIP func(string), cleanup func()) {
	t.Helper()
	originalIP := os.Getenv("UFO_IP")
	
	cleanup = func() {
		if originalIP != "" {
			os.Setenv("UFO_IP", originalIP)
		} else {
			os.Unsetenv("UFO_IP")
		}
	}
	
	setIP = func(ip string) {
		os.Setenv("UFO_IP", ip)
	}
	
	// Register cleanup with t.Cleanup so it runs even if test panics
	t.Cleanup(cleanup)
	
	return setIP, cleanup
}