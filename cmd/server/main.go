package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/effects"
	"github.com/starspace46/ufo-mcp-go/internal/events"
	"github.com/starspace46/ufo-mcp-go/internal/state"
	"github.com/starspace46/ufo-mcp-go/internal/tools"
)

const (
	ServerName    = "dynatrace-ufo"
	ServerVersion = "1.0.0"
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

	// setLogo tool
	setLogoTool := tools.NewSetLogoTool(deviceClient, broadcaster, stateManager)
	mcpServer.AddTool(setLogoTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return setLogoTool.Execute(ctx, request.GetArguments())
	})

	// setBrightness tool
	setBrightnessTool := tools.NewSetBrightnessTool(deviceClient, broadcaster, stateManager)
	mcpServer.AddTool(setBrightnessTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return setBrightnessTool.Execute(ctx, request.GetArguments())
	})

	// setRingPattern tool
	setRingPatternTool := tools.NewSetRingPatternTool(deviceClient, broadcaster, stateManager)
	mcpServer.AddTool(setRingPatternTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return setRingPatternTool.Execute(ctx, request.GetArguments())
	})

	// getLedState tool
	getLedStateTool := tools.NewGetLedStateTool(stateManager)
	mcpServer.AddTool(getLedStateTool.Definition(), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return getLedStateTool.Execute(ctx, request.GetArguments())
	})

	// TODO: Add remaining 5 tools here:
	// - playEffect
	// - stopEffects
	// - addEffect
	// - updateEffect
	// - deleteEffect
	// - listEffects
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

func startHTTPServer(mcpServer *server.MCPServer, port string, ctx context.Context) {
	httpServer := server.NewStreamableHTTPServer(mcpServer)

	// Start HTTP server in a goroutine
	go func() {
		addr := ":" + port
		log.Printf("HTTP server listening on %s/mcp", addr)
		if err := httpServer.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("HTTP server stopped")
}

func startStdioServer(mcpServer *server.MCPServer) {
	log.Printf("Starting stdio server...")
	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Stdio server error: %v", err)
	}
}
