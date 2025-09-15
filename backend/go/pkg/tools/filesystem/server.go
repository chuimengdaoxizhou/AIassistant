package filesystem

import (
	"Jarvis_2.0/backend/go/pkg/tools/filesystem/handler"
	"flag"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"log"
)

// Version 是服务的版本号
var Version = "2.0"

//STDIO transport (default) with current directory access
//go run main.go
//go run main.go -transport=stdio -allowed-dirs="/home/user/documents"
//
//SSE transport on port 8084
//go run main.go -transport=sse -port=8084 -allowed-dirs="/home/user/projects"
//
//StreamableHTTP transport on port 9000
//go run main.go -transport=httpstream -port=9000 -allowed-dirs="/tmp"

func main() {
	// Define command-line flags
	transport := flag.String("transport", "stdio", "Transport method: stdio, sse, or httpstream")
	port := flag.String("port", "8084", "Port for HTTP-based transports (sse, httpstream)")
	allowedDirs := flag.String("allowed-dirs", ".", "Comma-separated list of allowed directories")
	flag.Parse()

	// Parse allowed directories
	dirs := []string{*allowedDirs}
	if *allowedDirs != "." {
		// You might want to implement CSV parsing here for multiple directories
		// For now, using single directory
	}

	// Create filesystem server
	s, err := NewFilesystemServer(dirs)
	if err != nil {
		log.Fatalf("failed to create filesystem server: %v", err)
	}

	// Start server based on transport selection
	switch *transport {
	case "sse":
		log.Printf("Starting Filesystem MCP server with SSE transport on port %s", *port)
		sseServer := server.NewSSEServer(s)
		if err := sseServer.Start(":" + *port); err != nil {
			log.Fatalf("SSE server error: %v", err)
		}
	case "httpstream":
		log.Printf("Starting Filesystem MCP server with StreamableHTTP transport on port %s", *port)
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(":" + *port); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	case "stdio":
		log.Println("Starting Filesystem MCP server with STDIO transport")
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("STDIO server error: %v", err)
		}
	default:
		log.Fatalf("Unknown transport: %s. Use stdio, sse, or httpstream", *transport)
	}
}

// NewFilesystemServer 创建一个新的 MCP 文件系统服务。
// allowedDirs 参数指定了允许访问的目录列表，以确保安全。
func NewFilesystemServer(allowedDirs []string) (*server.MCPServer, error) {

	// 创建文件系统处理器
	h, err := handler.NewFilesystemHandler(allowedDirs)
	if err != nil {
		return nil, err
	}

	// 创建一个新的 MCP 服务实例
	s := server.NewMCPServer(
		"secure-filesystem-server", // 服务名称
		Version,                    // 服务版本
		server.WithResourceCapabilities(true, true), // 启用资源能力
	)

	// 注册资源处理器
	s.AddResource(mcp.NewResource(
		"file://",     // 资源 URI 前缀
		"File System", // 资源名称
		mcp.WithResourceDescription("Access to files and directories on the local file system"), // 资源描述
	), h.HandleReadResource) // 处理器函数

	// 注册工具处理器
	s.AddTool(mcp.NewTool(
		"read_file",
		mcp.WithDescription("Read the complete contents of a file from the file system."),
		mcp.WithString("path",
			mcp.Description("Path to the file to read"),
			mcp.Required(),
		),
	), h.HandleReadFile)

	s.AddTool(mcp.NewTool(
		"write_file",
		mcp.WithDescription("Create a new file or overwrite an existing file with new content."),
		mcp.WithString("path",
			mcp.Description("Path where to write the file"),
			mcp.Required(),
		),
		mcp.WithString("content",
			mcp.Description("Content to write to the file"),
			mcp.Required(),
		),
	), h.HandleWriteFile)

	s.AddTool(mcp.NewTool(
		"list_directory",
		mcp.WithDescription("Get a detailed listing of all files and directories in a specified path."),
		mcp.WithString("path",
			mcp.Description("Path of the directory to list"),
			mcp.Required(),
		),
	), h.HandleListDirectory)

	s.AddTool(mcp.NewTool(
		"create_directory",
		mcp.WithDescription("Create a new directory or ensure a directory exists."),
		mcp.WithString("path",
			mcp.Description("Path of the directory to create"),
			mcp.Required(),
		),
	), h.HandleCreateDirectory)

	s.AddTool(mcp.NewTool(
		"copy_file",
		mcp.WithDescription("Copy files and directories."),
		mcp.WithString("source",
			mcp.Description("Source path of the file or directory"),
			mcp.Required(),
		),
		mcp.WithString("destination",
			mcp.Description("Destination path"),
			mcp.Required(),
		),
	), h.HandleCopyFile)

	s.AddTool(mcp.NewTool(
		"move_file",
		mcp.WithDescription("Move or rename files and directories."),
		mcp.WithString("source",
			mcp.Description("Source path of the file or directory"),
			mcp.Required(),
		),
		mcp.WithString("destination",
			mcp.Description("Destination path"),
			mcp.Required(),
		),
	), h.HandleMoveFile)

	s.AddTool(mcp.NewTool(
		"search_files",
		mcp.WithDescription("Recursively search for files and directories matching a pattern."),
		mcp.WithString("path",
			mcp.Description("Starting path for the search"),
			mcp.Required(),
		),
		mcp.WithString("pattern",
			mcp.Description("Search pattern to match against file names"),
			mcp.Required(),
		),
	), h.HandleSearchFiles)

	s.AddTool(mcp.NewTool(
		"get_file_info",
		mcp.WithDescription("Retrieve detailed metadata about a file or directory."),
		mcp.WithString("path",
			mcp.Description("Path to the file or directory"),
			mcp.Required(),
		),
	), h.HandleGetFileInfo)

	s.AddTool(mcp.NewTool(
		"list_allowed_directories",
		mcp.WithDescription("Returns the list of directories that this server is allowed to access."),
	), h.HandleListAllowedDirectories)

	s.AddTool(mcp.NewTool(
		"read_multiple_files",
		mcp.WithDescription("Read the contents of multiple files in a single operation."),
		mcp.WithArray("paths",
			mcp.Description("List of file paths to read"),
			mcp.Required(),
			mcp.Items(map[string]any{"type": "string"}),
		),
	), h.HandleReadMultipleFiles)

	s.AddTool(mcp.NewTool(
		"tree",
		mcp.WithDescription("Returns a hierarchical JSON representation of a directory structure."),
		mcp.WithString("path",
			mcp.Description("Path of the directory to traverse"),
			mcp.Required(),
		),
		mcp.WithNumber("depth",
			mcp.Description("Maximum depth to traverse (default: 3)"),
		),
		mcp.WithBoolean("follow_symlinks",
			mcp.Description("Whether to follow symbolic links (default: false)"),
		),
	), h.HandleTree)

	s.AddTool(mcp.NewTool(
		"delete_file",
		mcp.WithDescription("Delete a file or directory from the file system."),
		mcp.WithString("path",
			mcp.Description("Path to the file or directory to delete"),
			mcp.Required(),
		),
		mcp.WithBoolean("recursive",
			mcp.Description("Whether to recursively delete directories (default: false)"),
		),
	), h.HandleDeleteFile)

	s.AddTool(mcp.NewTool(
		"modify_file",
		mcp.WithDescription("Update file by finding and replacing text. Provides a simple pattern matching interface without needing exact character positions."),
		mcp.WithString("path",
			mcp.Description("Path to the file to modify"),
			mcp.Required(),
		),
		mcp.WithString("find",
			mcp.Description("Text to search for (exact match or regex pattern)"),
			mcp.Required(),
		),
		mcp.WithString("replace",
			mcp.Description("Text to replace with"),
			mcp.Required(),
		),
		mcp.WithBoolean("all_occurrences",
			mcp.Description("Replace all occurrences of the matching text (default: true)"),
		),
		mcp.WithBoolean("regex",
			mcp.Description("Treat the find pattern as a regular expression (default: false)"),
		),
	), h.HandleModifyFile)

	s.AddTool(mcp.NewTool(
		"search_within_files",
		mcp.WithDescription("Search for text within file contents. Unlike search_files which only searches file names, this tool scans the actual contents of text files for matching substrings. Binary files are automatically excluded from the search. Reports file paths and line numbers where matches are found."),
		mcp.WithString("path",
			mcp.Description("Starting path for the search (must be a directory)"),
			mcp.Required(),
		),
		mcp.WithString("substring",
			mcp.Description("Text to search for within file contents"),
			mcp.Required(),
		),
		mcp.WithNumber("depth",
			mcp.Description("Maximum directory depth to search (default: unlimited)"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of results to return (default: 1000)"),
		),
	), h.HandleSearchWithinFiles)

	return s, nil
}
