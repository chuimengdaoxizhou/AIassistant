package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"Jarvis_2.0/backend/go/pkg/tools/readfile/marky"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// STDIO transport (default)
//go run main.go
//go run main.go -transport=stdio
//
// SSE transport on port 8085
//go run main.go -transport=sse -port=8085
//
// StreamableHTTP transport on port 9000
//go run main.go -transport=httpstream -port=9000

func main() {
	// Define command-line flags
	transport := flag.String("transport", "stdio", "Transport method: stdio, sse, or httpstream")
	port := flag.String("port", "8085", "Port for HTTP-based transports (sse, httpstream)")
	flag.Parse()

	// Create a new MCP server
	s := server.NewMCPServer(
		"Marky",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Add tool
	tool := mcp.NewTool("convert_to_markdown",
		mcp.WithDescription("Convert a file to markdown format"),
		mcp.WithString("input",
			mcp.Required(),
			mcp.Description("Path to the input file to convert to markdown"),
		),
		mcp.WithString("output",
			mcp.Description("Path to the output markdown file"),
		),
	)

	// Add tool handler
	s.AddTool(tool, convertToMarkdown)

	// Start server based on transport selection
	switch *transport {
	case "sse":
		log.Printf("Starting Marky MCP server with SSE transport on port %s", *port)
		sseServer := server.NewSSEServer(s)
		if err := sseServer.Start(":" + *port); err != nil {
			log.Fatalf("SSE server error: %v", err)
		}
	case "httpstream":
		log.Printf("Starting Marky MCP server with StreamableHTTP transport on port %s", *port)
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(":" + *port); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	case "stdio":
		log.Println("Starting Marky MCP server with STDIO transport")
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("STDIO server error: %v", err)
		}
	default:
		log.Fatalf("Unknown transport: %s. Use stdio, sse, or httpstream", *transport)
	}
}

func convertToMarkdown(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	inputFile, err := request.RequireString("input")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	outputFile := request.GetString("output", "console")

	m := marky.New()
	result, err := m.Convert(inputFile)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to convert file: %v", err)), nil
	}

	if outputFile != "console" {
		if err := os.WriteFile(outputFile, []byte(result), 0o644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to write output file: %v", err)), nil
		}
	}

	return mcp.NewToolResultText(result), nil
}
