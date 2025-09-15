package main

import (
	"Jarvis_2.0/backend/go/pkg/tools/edit_office/excel_handler"
	"flag"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"log"
)

func main() {
	// Define command-line flags
	transport := flag.String("transport", "stdio", "Transport method: stdio, sse, or httpstream")
	port := flag.String("port", "8081", "Port for HTTP-based transports (sse, httpstream)")
	flag.Parse()

	h, err := excel_handler.NewExcelHandler()
	if err != nil {
		log.Fatalf("failed to create excel handler: %v", err)
	}

	s := server.NewMCPServer("excel-editor", "1.0.0")

	// --- Register Tools ---
	s.AddTool(mcp.NewTool("excel_new_workbook",
		mcp.WithDescription("Creates a new, blank Excel workbook."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to save the new workbook.")),
	), h.HandleNewWorkbook)

	s.AddTool(mcp.NewTool("excel_add_sheet",
		mcp.WithDescription("Adds a new sheet to an existing workbook."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to the workbook.")),
		mcp.WithString("sheet_name", mcp.Required(), mcp.Description("Name for the new sheet.")),
	), h.HandleAddSheet)

	s.AddTool(mcp.NewTool("excel_set_cell_value",
		mcp.WithDescription("Sets the value of a specific cell."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to the workbook.")),
		mcp.WithString("sheet_name", mcp.Required(), mcp.Description("Name of the sheet to modify.")),
		mcp.WithString("cell_ref", mcp.Required(), mcp.Description("Cell reference, e.g., 'A1'.")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Value to set (string, number, or YYYY-MM-DD date).")),
	), h.HandleSetCellValue)

	// Start server based on transport selection
	switch *transport {
	case "sse":
		log.Printf("Starting Excel MCP server with SSE transport on port %s", *port)
		sseServer := server.NewSSEServer(s)
		if err := sseServer.Start(":" + *port); err != nil {
			log.Fatalf("SSE server error: %v", err)
		}
	case "httpstream":
		log.Printf("Starting Excel MCP server with StreamableHTTP transport on port %s", *port)
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(":" + *port); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	case "stdio":
		log.Println("Starting Excel MCP server with STDIO transport")
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
// SSE transport on port 8081
//go run main.go -transport=sse -port=8081
//
// StreamableHTTP transport on port 9000
//go run main.go -transport=httpstream -port=9000
