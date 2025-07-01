package device

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/starspace46/ufo-mcp-go/internal/testutil"
)


func TestNewClient(t *testing.T) {
	// Test default UFO_IP
	testutil.WithCleanEnv(t, func() {
		client := NewClient()
		if client.baseURL != "http://ufo" {
			t.Errorf("expected default baseURL 'http://ufo', got %s", client.baseURL)
		}
	})

	// Test custom UFO_IP
	testutil.WithUFOIP(t, "192.168.1.100", func() {
		client := NewClient()
		if client.baseURL != "http://192.168.1.100" {
			t.Errorf("expected baseURL 'http://192.168.1.100', got %s", client.baseURL)
		}
	})
}

func TestSendRawQuery(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api" {
			t.Errorf("expected path '/api', got %s", r.URL.Path)
		}

		query := r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK: " + query))
	}))
	defer server.Close()

	// Override UFO_IP to point to test server
	testutil.WithUFOIP(t, server.URL[7:], func() { // Remove "http://" prefix
		client := NewClient()

		tests := []struct {
			name     string
			query    string
			expected string
		}{
			{"simple query", "effect=rainbow", "OK: effect=rainbow"},
			{"query with leading ?", "?dim=100", "OK: dim=100"},
			{"query with leading /", "/logo=on", "OK: logo=on"},
			{"empty query", "", "OK: "},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				resp, err := client.SendRawQuery(context.Background(), tt.query)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, resp)
				}
			})
		}
	})
}

func TestSetBrightness(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.RawQuery
		if query == "dim=128" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	testutil.WithUFOIP(t, server.URL[7:], func() {
		client := NewClient()

		// Test valid brightness
		err := client.SetBrightness(context.Background(), 128)
		if err != nil {
			t.Errorf("unexpected error for valid brightness: %v", err)
		}

		// Test invalid brightness - too low
		err = client.SetBrightness(context.Background(), -1)
		if err == nil {
			t.Error("expected error for brightness < 0")
		}

		// Test invalid brightness - too high
		err = client.SetBrightness(context.Background(), 256)
		if err == nil {
			t.Error("expected error for brightness > 255")
		}
	})
}
