package word_handler

import (
	"context"
	"fmt"

	"Jarvis_2.0/backend/go/pkg/tools/edit_office/editor"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/unidoc/unioffice/v2/measurement"
)

// WordHandler handles all Word-related tool requests.
type WordHandler struct{}

// NewWordHandler creates a new WordHandler.
func NewWordHandler() (*WordHandler, error) {
	return &WordHandler{}, nil
}

// --- Tool Handlers ---

func (h *WordHandler) HandleNewDocument(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return nil, err
	}

	doc := editor.NewWordDocument()
	doc.AddParagraph()
	if err := doc.SaveToFile(path); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to create new document: %v", err)}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Successfully created new Word document at %s", path)}},
	}, nil
}

func (h *WordHandler) HandleAddParagraph(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return nil, err
	}
	text, err := req.RequireString("text")
	if err != nil {
		return nil, err
	}
	isBold := req.GetBool("is_bold", false)
	isItalic := req.GetBool("is_italic", false)
	fontSize := req.GetFloat("font_size", 0)

	doc, err := editor.OpenWordDocument(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to open document: %v", err)}},
			IsError: true,
		}, nil
	}

	para := doc.AddParagraph()
	run := para.AddRun()
	run.AddText(text)

	props := run.Properties()
	if isBold {
		props.SetBold(true)
	}
	if isItalic {
		props.SetItalic(true)
	}
	if fontSize > 0 {
		props.SetFontSize(measurement.Distance(fontSize))
	}

	if err := doc.SaveToFile(path); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to save document: %v", err)}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Successfully added a paragraph to %s", path)}},
	}, nil
}

func (h *WordHandler) HandleAddTable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return nil, err
	}
	rows, err := req.RequireInt("rows")
	if err != nil {
		return nil, err
	}
	cols, err := req.RequireInt("cols")
	if err != nil {
		return nil, err
	}

	doc, err := editor.OpenWordDocument(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to open document: %v", err)}},
			IsError: true,
		}, nil
	}

	table := doc.AddTable()
	for i := 0; i < rows; i++ {
		row := table.AddRow()
		for j := 0; j < cols; j++ {
			row.AddCell().AddParagraph()
		}
	}

	if err := doc.SaveToFile(path); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to save document: %v", err)}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Successfully added a %dx%d table to %s", rows, cols, path)}},
	}, nil
}
