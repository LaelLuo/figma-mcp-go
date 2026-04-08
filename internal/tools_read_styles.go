package internal

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerReadStyleTools(s *server.MCPServer, node *Node) {
	s.AddTool(mcp.NewTool("get_styles",
		mcp.WithDescription("Get all local styles in the document: paint, text, effect, and grid styles"),
	), makeHandler(node, "get_styles", nil, nil))

	s.AddTool(mcp.NewTool("get_variable_defs",
		mcp.WithDescription("Get all local variable definitions: collections, modes, and values. Accepts official-style compatibility parameters in local fallback mode."),
		mcp.WithString("fileKey",
			mcp.Description("Compatibility parameter for the official Figma MCP. In local fallback mode this may be ignored."),
		),
		mcp.WithString("nodeId",
			mcp.Description("Optional node ID in colon format e.g. '4029:12345'."),
		),
		mcp.WithString("clientFrameworks",
			mcp.Description("Optional framework context for compatibility with the official Figma MCP."),
		),
		mcp.WithString("clientLanguages",
			mcp.Description("Optional language context for compatibility with the official Figma MCP."),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := node.Send(ctx, "get_variable_defs", nil, collectArgs(req, "fileKey", "nodeId", "clientFrameworks", "clientLanguages"))
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("get_local_components",
		mcp.WithDescription("Get all components defined in the current Figma file."),
	), makeHandler(node, "get_local_components", nil, nil))

	s.AddTool(mcp.NewTool("get_annotations",
		mcp.WithDescription("Get all dev-mode annotations in the current document or on a specific node."),
		mcp.WithString("nodeId",
			mcp.Description("Optional node ID to filter annotations, colon format e.g. '4029:12345'"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := map[string]interface{}{}
		if id, ok := req.GetArguments()["nodeId"].(string); ok && id != "" {
			params["nodeId"] = id
		}
		resp, err := node.Send(ctx, "get_annotations", nil, params)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("export_tokens",
		mcp.WithDescription("Export all design tokens (variables and paint styles) as JSON or CSS custom properties. Ideal for bridging Figma variables into your codebase."),
		mcp.WithString("format", mcp.Description("Output format: json (default) or css")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := map[string]interface{}{}
		if f, ok := req.GetArguments()["format"].(string); ok && f != "" {
			params["format"] = f
		}
		resp, err := node.Send(ctx, "export_tokens", nil, params)
		return renderResponse(resp, err)
	})
}
