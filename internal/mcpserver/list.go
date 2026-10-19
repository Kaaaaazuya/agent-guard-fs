package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// listOutput は list_files の構造化出力。
type listOutput struct {
	Path      string        `json:"path"`
	Entries   []entryOutput `json:"entries"`
	Truncated bool          `json:"truncated"`
}

// entryOutput は list_files の 1 エントリ。
type entryOutput struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func listFilesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:         "list_files",
		Description:  "List the direct children of a directory that the policy allows. Entries the policy hides are omitted.",
		InputSchema:  schemaFor[pathArgs](),
		OutputSchema: schemaFor[listOutput](),
		Annotations:  &mcp.ToolAnnotations{ReadOnlyHint: true},
	}
}

func (s *Server) listFiles(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var in pathArgs
	if err := decode(req, &in); err != nil {
		return invalidArguments(), nil
	}
	res, err := s.guard.List(ctx, in.Path)
	if err != nil {
		return errorResult(err), nil
	}

	entries := make([]entryOutput, 0, len(res.Entries))
	lines := make([]string, 0, len(res.Entries))
	for _, e := range res.Entries {
		entries = append(entries, entryOutput{Name: e.Name, Type: e.Type})
		lines = append(lines, fmt.Sprintf("%s (%s)", e.Name, e.Type))
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: strings.Join(lines, "\n")}},
		StructuredContent: listOutput{Path: res.Path, Entries: entries, Truncated: res.Truncated},
	}, nil
}
