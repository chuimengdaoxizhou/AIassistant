package excel_handler

import (
	"context"
	"fmt"
	"time"

	"Jarvis_2.0/backend/go/pkg/tools/edit_office/editor"
	"github.com/mark3labs/mcp-go/mcp"
)

// ExcelHandler handles all Excel-related tool requests.
type ExcelHandler struct{}

// NewExcelHandler creates a new ExcelHandler.
func NewExcelHandler() (*ExcelHandler, error) {
	return &ExcelHandler{}, nil
}

// --- Tool Handlers ---

func (h *ExcelHandler) HandleNewWorkbook(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return nil, err
	}

	wb := editor.NewExcelWorkbook()
	wb.AddSheet("Sheet1")
	if err := wb.SaveToFile(path); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to create new workbook: %v", err)}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Successfully created new Excel workbook at %s", path)}},
	}, nil
}

func (h *ExcelHandler) HandleAddSheet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return nil, err
	}
	sheetName, err := req.RequireString("sheet_name")
	if err != nil {
		return nil, err
	}

	wb, err := editor.OpenExcelWorkbook(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to open workbook: %v", err)}},
			IsError: true,
		}, nil
	}
	wb.AddSheet(sheetName)
	if err := wb.SaveToFile(path); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to save workbook: %v", err)}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Successfully added sheet '%s' to %s", sheetName, path)}},
	}, nil
}

func (h *ExcelHandler) HandleSetCellValue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return nil, err
	}
	sheetName, err := req.RequireString("sheet_name")
	if err != nil {
		return nil, err
	}
	cellRef, err := req.RequireString("cell_ref")
	if err != nil {
		return nil, err
	}
	value := req.GetRawArguments().(map[string]any)["value"]

	wb, err := editor.OpenExcelWorkbook(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to open workbook: %v", err)}},
			IsError: true,
		}, nil
	}
	sheet, err := wb.GetSheet(sheetName)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to get sheet: %v", err)}},
			IsError: true,
		}, nil
	}
	cell := sheet.Cell(cellRef)

	switch v := value.(type) {
	case string:
		if t, err := time.Parse("2006-01-02", v); err == nil {
			cell.SetDate(t)
		} else {
			cell.SetString(v)
		}
	case float64:
		cell.SetNumber(v)
	default:
		cell.SetString(fmt.Sprintf("%v", v))
	}

	if err := wb.SaveToFile(path); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("failed to save workbook: %v", err)}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Successfully set cell %s to %v", cellRef, value)}},
	}, nil
}
