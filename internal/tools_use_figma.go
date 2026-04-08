package internal

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerUseFigmaTool(s *server.MCPServer, node *Node) {
	s.AddTool(mcp.NewTool("use_figma",
		mcp.WithDescription("Run JavaScript in the current Figma desktop file via the plugin bridge. This is the local fallback counterpart to the official Figma MCP use_figma tool."),
		mcp.WithString("code",
			mcp.Required(),
			mcp.Description("JavaScript code to execute. Supports top-level await through the local executor."),
		),
		mcp.WithString("description",
			mcp.Description("A concise description of what the code aims to do."),
		),
		mcp.WithString("fileKey",
			mcp.Description("Compatibility parameter for the official Figma MCP. In local fallback mode this may be ignored."),
		),
		mcp.WithString("skillNames",
			mcp.Description("Optional comma-separated list of skill names being followed."),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := map[string]interface{}{}
		for _, key := range []string{"code", "description", "fileKey", "skillNames"} {
			if value, ok := req.GetArguments()[key]; ok {
				params[key] = value
			}
		}
		resp, err := node.Send(ctx, "use_figma", nil, params)
		return renderResponse(resp, err)
	})
}
