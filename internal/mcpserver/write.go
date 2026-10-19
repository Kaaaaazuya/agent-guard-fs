package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// writeArgs は write_file の引数。
type writeArgs struct {
	Path    string `json:"path" jsonschema:"the path relative to the root, such as src/main.go"`
	Content string `json:"content" jsonschema:"the full UTF-8 text to store at the path"`
}

// writeOutput は write_file の構造化出力。
type writeOutput struct {
	Path    string `json:"path"`
	Bytes   int    `json:"bytes"`
	Created bool   `json:"created"`
}

func writeFileTool() *mcp.Tool {
	return &mcp.Tool{
		Name:         "write_file",
		Description:  "Create or replace a UTF-8 text file that the policy allows. Parent directories are not created.",
		InputSchema:  schemaFor[writeArgs](),
		OutputSchema: schemaFor[writeOutput](),
		Annotations:  &mcp.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}
}

func (s *Server) writeFile(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var in writeArgs
	if err := decode(req, &in); err != nil {
		return invalidArguments(), nil
	}
	res, err := s.guard.Write(ctx, in.Path, []byte(in.Content))
	if err != nil {
		return errorResult(err), nil
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("wrote %d bytes to %s", res.Bytes, res.Path)}},
		StructuredContent: writeOutput{Path: res.Path, Bytes: res.Bytes, Created: res.Created},
	}, nil
}
