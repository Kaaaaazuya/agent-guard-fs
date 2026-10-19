package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// deleteOutput は delete_file の構造化出力。
type deleteOutput struct {
	Path string `json:"path"`
}

func deleteFileTool() *mcp.Tool {
	return &mcp.Tool{
		Name:         "delete_file",
		Description:  "Delete a regular file that the policy allows. Directories and links are not deleted.",
		InputSchema:  schemaFor[pathArgs](),
		OutputSchema: schemaFor[deleteOutput](),
		Annotations:  &mcp.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}
}

func (s *Server) deleteFile(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var in pathArgs
	if err := decode(req, &in); err != nil {
		return invalidArguments(), nil
	}
	deleted, err := s.guard.Delete(ctx, in.Path)
	if err != nil {
		return errorResult(err), nil
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("deleted %s", deleted)}},
		StructuredContent: deleteOutput{Path: deleted},
	}, nil
}
