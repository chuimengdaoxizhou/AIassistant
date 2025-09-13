package main

import (
	"log"

	"Jarvis_2.0/backend/go/pkg/tools/edit_office/ppt_handler"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
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

	log.Println("Starting PowerPoint MCP server on :8083")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}