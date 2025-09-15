package main

import (
	"Jarvis_2.0/backend/go/pkg/tools/edit_office/ppt_handler"
	"flag"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"log"
)

func main() {
	// Define command-line flags
	transport := flag.String("transport", "stdio", "Transport method: stdio, sse, or httpstream")
	port := flag.String("port", "8083", "Port for HTTP-based transports (sse, httpstream)")
	flag.Parse()

	h, err := ppt_handler.NewPPTHandler()
	if err != nil {
		log.Fatalf("failed to create ppt handler: %v", err)
	}

	s := server.NewMCPServer("ppt-editor", "1.0.0")

	// --- Register Tools ---
	s.AddTool(mcp.NewTool("ppt_new_presentation",
		mcp.WithDescription("Creates a new, blank PowerPoint presentation."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to save the new presentation.")),
	), h.HandleNewPresentation)

	s.AddTool(mcp.NewTool("ppt_add_slide",
		mcp.WithDescription("Adds a new slide to a presentation."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to the presentation.")),
		mcp.WithString("layout_name", mcp.Description("Optional layout name, e.g., 'Title Slide'.")),
	), h.HandleAddSlide)

	s.AddTool(mcp.NewTool("ppt_add_text_box",
		mcp.WithDescription("Adds a text box with content to a slide."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to the presentation.")),
		mcp.WithNumber("slide_index", mcp.Required(), mcp.Description("0-based index of the slide.")),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text content to add.")),
	), h.HandleAddTextBox)

	// Start server based on transport selection
	switch *transport {
	case "sse":
		log.Printf("Starting PowerPoint MCP server with SSE transport on port %s", *port)
		sseServer := server.NewSSEServer(s)
		if err := sseServer.Start(":" + *port); err != nil {
			log.Fatalf("SSE server error: %v", err)
		}
	case "httpstream":
		log.Printf("Starting PowerPoint MCP server with StreamableHTTP transport on port %s", *port)
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(":" + *port); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	case "stdio":
		log.Println("Starting PowerPoint MCP server with STDIO transport")
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
// SSE transport on port 8083
//go run main.go -transport=sse -port=8083
//
// StreamableHTTP transport on port 9000
//go run main.go -transport=httpstream -port=9000
