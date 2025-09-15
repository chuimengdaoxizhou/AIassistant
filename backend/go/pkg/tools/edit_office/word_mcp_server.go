package main

import (
	"Jarvis_2.0/backend/go/pkg/tools/edit_office/word_handler"
	"flag"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"log"
)

func main() {
	// Define command-line flags
	transport := flag.String("transport", "stdio", "Transport method: stdio, sse, or httpstream")
	port := flag.String("port", "8082", "Port for HTTP-based transports (sse, httpstream)")
	flag.Parse()

	h, err := word_handler.NewWordHandler()
	if err != nil {
		log.Fatalf("failed to create word handler: %v", err)
	}

	s := server.NewMCPServer("word-editor", "1.0.0")

	// --- Register Tools ---
	s.AddTool(mcp.NewTool("word_new_document",
		mcp.WithDescription("Creates a new, blank Word document."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to save the new document.")),
	), h.HandleNewDocument)

	s.AddTool(mcp.NewTool("word_add_paragraph",
		mcp.WithDescription("Adds a new paragraph with formatted text."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to the document.")),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text content of the paragraph.")),
		mcp.WithBoolean("is_bold", mcp.Description("Set text to bold.")),
		mcp.WithBoolean("is_italic", mcp.Description("Set text to italic.")),
		mcp.WithNumber("font_size", mcp.Description("Font size for the text.")),
	), h.HandleAddParagraph)

	s.AddTool(mcp.NewTool("word_add_table",
		mcp.WithDescription("Adds a new table to the document."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to the document.")),
		mcp.WithNumber("rows", mcp.Required(), mcp.Description("Number of rows for the new table.")),
		mcp.WithNumber("cols", mcp.Required(), mcp.Description("Number of columns for the new table.")),
	), h.HandleAddTable)

	// Start server based on transport selection
	switch *transport {
	case "sse":
		log.Printf("Starting Word MCP server with SSE transport on port %s", *port)
		sseServer := server.NewSSEServer(s)
		if err := sseServer.Start(":" + *port); err != nil {
			log.Fatalf("SSE server error: %v", err)
		}
	case "httpstream":
		log.Printf("Starting Word MCP server with StreamableHTTP transport on port %s", *port)
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(":" + *port); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	case "stdio":
		log.Println("Starting Word MCP server with STDIO transport")
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("STDIO server error: %v", err)
		}
	default:
		log.Fatalf("Unknown transport: %s. Use stdio, sse, or httpstream", *transport)
	}
}

// STDIO transport (default)
//go run main.go
//go run main.go -transport=stdio
//
// SSE transport on port 8082
//go run main.go -transport=sse -port=8082
//
// StreamableHTTP transport on port 9000
//go run main.go -transport=httpstream -port=9000
