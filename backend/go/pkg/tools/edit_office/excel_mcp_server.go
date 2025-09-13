package main

import (
	"log"

	"Jarvis_2.0/backend/go/pkg/tools/edit_office/excel_handler"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
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

	log.Println("Starting Excel MCP server on :8081")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}
