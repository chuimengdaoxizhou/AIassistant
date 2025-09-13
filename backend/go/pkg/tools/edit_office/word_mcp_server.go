package main

import (
	"log"

	"Jarvis_2.0/backend/go/pkg/tools/edit_office/word_handler"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
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

	log.Println("Starting Word MCP server on :8082")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}