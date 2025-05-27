package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/kugouming/mcp-go/mcp"
	"github.com/kugouming/mcp-go/server"
)

// extractTenantFromPath extracts tenant from path like "/api/{tenant}/sse" or "/api/{tenant}/message"
// This is a compatibility function for Go versions < 1.22 that don't have PathValue
func extractTenantFromPath(path string) string {
	// Expected path format: /api/{tenant}/sse or /api/{tenant}/message
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "api" {
		return parts[1]
	}
	return ""
}

func main() {
	var addr string
	flag.StringVar(&addr, "addr", ":8080", "address to listen on")
	flag.Parse()

	mcpServer := server.NewMCPServer("dynamic-path-example", "1.0.0")

	// Add a trivial tool for demonstration
	mcpServer.AddTool(mcp.NewTool("echo"), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(fmt.Sprintf("Echo: %v", req.GetArguments()["message"])), nil
	})

	// Use a dynamic base path based on a path parameter (compatible with Go 1.20+)
	sseServer := server.NewSSEServer(
		mcpServer,
		server.WithDynamicBasePath(func(r *http.Request, sessionID string) string {
			tenant := extractTenantFromPath(r.URL.Path)
			return "/api/" + tenant
		}),
		server.WithBaseURL(fmt.Sprintf("http://localhost%s", addr)),
		server.WithUseFullURLForMessageEndpoint(true),
	)

	mux := http.NewServeMux()
	// Note: For Go < 1.22, we need to handle path patterns manually
	// This is a simplified approach - in production you might want to use a router like gorilla/mux
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/sse") {
			sseServer.SSEHandler().ServeHTTP(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/message") {
			sseServer.MessageHandler().ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	log.Printf("Dynamic SSE server listening on %s", addr)
	log.Printf("Example URLs:")
	log.Printf("  SSE endpoint: http://localhost%s/api/tenant123/sse", addr)
	log.Printf("  Message endpoint: http://localhost%s/api/tenant123/message", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
