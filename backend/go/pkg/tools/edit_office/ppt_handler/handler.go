package ppt_handler

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/z-Wind/Jarvis_2.0/backend/go/pkg/tools/edit_office/editor"
)

// PPTHandler handles all PowerPoint-related tool requests.	ype PPTHandler struct{}

// NewPPTHandler creates a new PPTHandler.
func NewPPTHandler() (*PPTHandler, error) {
	return &PPTHandler{}, nil
}

// --- Tool Handlers ---

func (h *PPTHandler) HandleNewPresentation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return nil, err
	}

	ppt := editor.NewPowerPointPresentation()
	layout, err := ppt.GetLayoutByName("Title Slide")
	if err != nil {
		// If default layout not found, add a blank slide instead
		ppt.AddSlide()
	} else {
		_, slideErr := ppt.AddSlideWithLayout(layout)
		if slideErr != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to add default title slide with layout: %v", slideErr)}},
				IsError: true,
			}, nil
		}
	}

	if err := ppt.SaveToFile(path); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to create new presentation: %v", err)}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Successfully created new PowerPoint presentation at %s", path)}},
	}, nil
}

func (h *PPTHandler) HandleAddSlide(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return nil, err
	}
	layoutName := req.GetString("layout_name", "")

	ppt, err := editor.OpenPowerPointPresentation(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to open presentation: %v", err)}},
			IsError: true,
		}, nil
	}

	if layoutName != "" {
		layout, err := ppt.GetLayoutByName(layoutName)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("could not find layout '%s': %v", layoutName, err)}},
				IsError: true,
			}, nil
		}
		_, slideErr := ppt.AddSlideWithLayout(layout)
		if slideErr != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to add slide with layout '%s': %v", layoutName, slideErr)}},
				IsError: true,
			}, nil
		}
	} else {
		ppt.AddSlide()
	}

	if err := ppt.SaveToFile(path); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to save presentation: %v", err)}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Successfully added a new slide to %s", path)}},
	}, nil
}

func (h *PPTHandler) HandleAddTextBox(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return nil, err
	}
	slideIndex, err := req.RequireInt("slide_index")
	if err != nil {
		return nil, err
	}
	text, err := req.RequireString("text")
	if err != nil {
		return nil, err
	}

	ppt, err := editor.OpenPowerPointPresentation(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to open presentation: %v", err)}},
			IsError: true,
		}, nil
	}
	slides := ppt.Slides()
	if slideIndex >= len(slides) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("slide index %d is out of bounds", slideIndex)}},
			IsError: true,
		}, nil
	}
	slide := slides[slideIndex]

	tb := slide.AddTextBox()
	tb.SetText(text)

	if err := ppt.SaveToFile(path); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to save presentation: %v", err)}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Successfully added text box to slide %d in %s", slideIndex, path)}},
	}, nil
}