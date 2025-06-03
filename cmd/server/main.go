package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/effects"
	"github.com/starspace46/ufo-mcp-go/internal/events"
	"github.com/starspace46/ufo-mcp-go/internal/state"
	"github.com/starspace46/ufo-mcp-go/internal/tools"
	"github.com/starspace46/ufo-mcp-go/internal/version"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	ServerName    = "dynatrace-ufo"
	ServerVersion = "1.0.1"
)

func main() {
	var transport string
	var port string
	var ufoIP string
	var effectsFile string

	flag.StringVar(&transport, "t", "stdio", "Transport type (stdio or http)")
	flag.StringVar(&transport, "transport", "stdio", "Transport type (stdio or http)")
	flag.StringVar(&port, "port", "8080", "HTTP port when using http transport")
	flag.StringVar(&ufoIP, "ufo-ip", os.Getenv("UFO_IP"), "UFO device IP address")
	flag.StringVar(&effectsFile, "effects-file", "/data/effects.json", "Path to effects JSON file")
	flag.Parse()

	// Default UFO IP if not set
	if ufoIP == "" {
		ufoIP = "ufo"
		os.Setenv("UFO_IP", ufoIP)
	}

	log.Printf("Starting MCP UFO Server")
	log.Printf("UFO IP: %s", ufoIP)
	log.Printf("Effects file: %s", effectsFile)
	log.Printf("Transport: %s", transport)

	// Initialize core components
	deviceClient := device.NewClient()
	broadcaster := events.NewBroadcaster()
	effectsStore := effects.NewStore(effectsFile)
	stateManager := state.NewManager(broadcaster)

	// Load effects (creates seed effects if file doesn't exist)
	if err := effectsStore.Load(); err != nil {
		log.Fatalf("Failed to load effects: %v", err)
	}

	// Create MCP server
	mcpServer := createMCPServer(deviceClient, broadcaster, effectsStore, stateManager)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		broadcaster.Close()
		cancel()
	}()

	// Start server based on transport type
	if transport == "http" {
		startHTTPServer(mcpServer, port, ctx)
	} else {
		startStdioServer(mcpServer)
	}
}

func createMCPServer(deviceClient *device.Client, broadcaster *events.Broadcaster, effectsStore *effects.Store, stateManager *state.Manager) *server.MCPServer {
	// Create server with capabilities
	mcpServer := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithToolCapabilities(true), // Tools can change
		server.WithResourceCapabilities(true, false), // Resources, no subscription yet
		server.WithLogging(),
		server.WithInstructions(`This MCP server provides control over a Dynatrace UFO lighting device. 

Available capabilities:
- Send raw API commands to the UFO
- Control lighting effects and patterns  
- Manage brightness and logo
- Store and manage custom lighting effects
- Real-time event streaming for state changes

Resources:
- ufo://status - Get UFO device status
- ufo://ledstate - Get current LED colors, brightness, logo state, and running effect (shadow state)

Use sendRawApi for direct UFO control or the high-level tools for common operations.
To check current LED colors, read the ufo://ledstate resource.`),
	)

	// Register tools
	registerTools(mcpServer, deviceClient, broadcaster, effectsStore, stateManager)

	// Register resources
	registerResources(mcpServer, deviceClient, stateManager)

	return mcpServer
}

func registerTools(mcpServer *server.MCPServer, deviceClient *device.Client, broadcaster *events.Broadcaster, effectsStore *effects.Store, stateManager *state.Manager) {
	// sendRawApi tool
	sendRawApiTool := tools.NewSendRawApiTool(deviceClient, broadcaster)
	mcpServer.AddTool(sendRawApiTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return sendRawApiTool.Execute(ctx, request.GetArguments())
	})

	// setLogo tool - REMOVED: Use configureLighting instead for consistent stack behavior
	// setLogoTool := tools.NewSetLogoTool(deviceClient, broadcaster, stateManager)
	// mcpServer.AddTool(setLogoTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 	return setLogoTool.Execute(ctx, request.GetArguments())
	// })

	// setBrightness tool - implemented but not exposed via MCP
	// Brightness can be controlled via dim parameter in patterns

	// setRingPattern tool - REMOVED: Use configureLighting instead for consistent stack behavior
	// setRingPatternTool := tools.NewSetRingPatternTool(deviceClient, broadcaster, stateManager)
	// mcpServer.AddTool(setRingPatternTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 	return setRingPatternTool.Execute(ctx, request.GetArguments())
	// })

	// getLedState tool
	getLedStateTool := tools.NewGetLedStateTool(stateManager)
	mcpServer.AddTool(getLedStateTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return getLedStateTool.Execute(ctx, request.GetArguments())
	})

	// listEffects tool
	listEffectsTool := tools.NewListEffectsTool(effectsStore)
	mcpServer.AddTool(listEffectsTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return listEffectsTool.Execute(ctx, request.GetArguments())
	})

	// Effects CRUD tools are implemented but not exposed via MCP
	// They remain available for internal use or future activation
	// - addEffect
	// - updateEffect  
	// - deleteEffect

	// playEffect tool
	playEffectTool := tools.NewPlayEffectTool(deviceClient, broadcaster, effectsStore, stateManager)
	mcpServer.AddTool(playEffectTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return playEffectTool.Execute(ctx, request.GetArguments())
	})

	// configureLighting tool - unified lighting control
	configureLightingTool := tools.NewConfigureLightingTool(deviceClient, broadcaster, stateManager)
	mcpServer.AddTool(configureLightingTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return configureLightingTool.Execute(ctx, request.GetArguments())
	})

	// stopEffect tool - pops the current effect from the stack and resumes the previous one
	stopEffectTool := tools.NewStopEffectTool(deviceClient, broadcaster, stateManager)
	mcpServer.AddTool(stopEffectTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return stopEffectTool.Execute(ctx, request.GetArguments())
	})
}

func registerResources(mcpServer *server.MCPServer, deviceClient *device.Client, stateManager *state.Manager) {
	// getStatus resource
	mcpServer.AddResource(
		mcp.Resource{
			URI:         "ufo://status",
			Name:        "UFO Status",
			Description: "Current status and configuration of the UFO device",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			status, err := deviceClient.GetStatus(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get UFO status: %w", err)
			}

			// Convert status to JSON string
			statusJSON := fmt.Sprintf(`{
  "timestamp": %v,
  "ufo_response": %q,
  "ufo_ip": %q
}`, status["timestamp"], status["response"], os.Getenv("UFO_IP"))

			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "application/json",
					Text:     statusJSON,
				},
			}, nil
		},
	)

	// getLedState resource
	mcpServer.AddResource(
		mcp.Resource{
			URI:         "ufo://ledstate",
			Name:        "UFO LED State",
			Description: "Current LED state (shadow copy) showing colors, brightness, logo state, and running effect",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			// Get the current LED state
			ledStateJSON, err := stateManager.ToJSON()
			if err != nil {
				return nil, fmt.Errorf("failed to get LED state: %w", err)
			}

			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      request.Params.URI,
					MIMEType: "application/json",
					Text:     ledStateJSON,
				},
			}, nil
		},
	)
}

var startTime = time.Now()

func startHTTPServer(mcpServer *server.MCPServer, port string, ctx context.Context) {
	// Create the MCP streamable HTTP server
	mcpHandler := server.NewStreamableHTTPServer(mcpServer)
	
	// Create a mux to handle both MCP and health check
	mux := http.NewServeMux()
	
	// Mount MCP handler at /mcp
	mux.Handle("/mcp", mcpHandler)
	
	// Add health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		health := map[string]interface{}{
			"status":      "healthy",
			"version":     version.Version,
			"gitCommit":   version.GitCommit,
			"buildTime":   version.BuildTime,
			"specVersion": version.SpecVersion,
			"uptime":      time.Since(startTime).String(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	})
	
	// Create HTTP/2 server
	h2s := &http2.Server{}
	
	// Create server with HTTP/2 support
	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      h2c.NewHandler(mux, h2s),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // No timeout for streaming
		IdleTimeout:  120 * time.Second,
	}
	
	// Start server with graceful shutdown
	go func() {
		log.Printf("HTTP server listening on %s", httpServer.Addr)
		log.Printf("  MCP endpoint: http://localhost%s/mcp", httpServer.Addr)
		log.Printf("  Health check: http://localhost%s/healthz", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	
	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	log.Println("HTTP server stopped")
}

func startStdioServer(mcpServer *server.MCPServer) {
	log.Printf("Starting stdio server...")
	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Stdio server error: %v", err)
	}
}
