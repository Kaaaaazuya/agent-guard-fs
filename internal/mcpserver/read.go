package mcpserver

import (
	"context"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// pathArgs は path だけを取るツールの引数。
type pathArgs struct {
	Path string `json:"path" jsonschema:"the path relative to the root, such as src/main.go"`
}

// readOutput は read_file の構造化出力。
type readOutput struct {
	Path string `json:"path"`
	Size int    `json:"size"`
}

func readFileTool() *mcp.Tool {
	return &mcp.Tool{
		Name:         "read_file",
		Description:  "Read a UTF-8 text file that the policy allows.",
		InputSchema:  schemaFor[pathArgs](),
		OutputSchema: schemaFor[readOutput](),
		Annotations:  &mcp.ToolAnnotations{ReadOnlyHint: true},
	}
}

func (s *Server) readFile(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var in pathArgs
	if err := decode(req, &in); err != nil {
		return invalidArguments(), nil
	}
	res, err := s.guard.Read(ctx, in.Path)
	if err != nil {
		return errorResult(err), nil
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(res.Content)}},
		StructuredContent: readOutput{Path: res.Path, Size: len(res.Content)},
	}, nil
}

// schemaFor は T から JSON Schema を作る。失敗は型の誤りなので、起動時に止まるよう panic する。
func schemaFor[T any]() *jsonschema.Schema {
	s, err := jsonschema.For[T](nil)
	if err != nil {
		panic(err)
	}
	return s
}
