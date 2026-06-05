package internal

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerReadDocumentTools(s *server.MCPServer, node *Node) {
	s.AddTool(mcp.NewTool("get_document",
		mcp.WithDescription("Get the current Figma page document tree"),
	), makeHandler(node, "get_document", nil, nil))

	s.AddTool(mcp.NewTool("get_pages",
		mcp.WithDescription("List all pages in the document with their IDs and names. Lightweight alternative to get_document."),
	), makeHandler(node, "get_pages", nil, nil))

	s.AddTool(mcp.NewTool("get_metadata",
		mcp.WithDescription("Get metadata for a node or page in the current Figma desktop file. Local fallback output is lighter than the official cloud response but uses official-style parameters."),
		mcp.WithString("fileKey",
			mcp.Description("Compatibility parameter for the official Figma MCP. In local fallback mode this may be ignored."),
		),
		mcp.WithString("nodeId",
			mcp.Description("Optional node ID in colon format e.g. '4029:12345'. Defaults to the current page when omitted."),
		),
		mcp.WithString("clientFrameworks",
			mcp.Description("Optional framework context for compatibility with the official Figma MCP."),
		),
		mcp.WithString("clientLanguages",
			mcp.Description("Optional language context for compatibility with the official Figma MCP."),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := node.Send(ctx, "get_metadata", nil, collectArgs(req, "fileKey", "nodeId", "clientFrameworks", "clientLanguages"))
		return renderMetadataResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("get_selection",
		mcp.WithDescription("Get the currently selected nodes in Figma"),
	), makeHandler(node, "get_selection", nil, nil))

	s.AddTool(mcp.NewTool("get_node",
		mcp.WithDescription("Get a specific Figma node by ID. Must use colon format e.g. '4029:12345', never hyphens."),
		mcp.WithString("nodeId",
			mcp.Required(),
			mcp.Description("Node ID in colon format e.g. '4029:12345'"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nodeID, _ := req.GetArguments()["nodeId"].(string)
		resp, err := node.Send(ctx, "get_node", []string{nodeID}, nil)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("get_nodes_info",
		mcp.WithDescription("Get lightweight information about multiple Figma nodes by ID in a single call. Use detail=full only when you explicitly need recursive payloads."),
		mcp.WithArray("nodeIds",
			mcp.Required(),
			mcp.Description("List of node IDs in colon format e.g. ['4029:12345', '4029:67890']"),
			mcp.WithStringItems(),
		),
		mcp.WithString("detail",
			mcp.Description("Optional payload depth. Defaults to shallow for id/name/type/bounds summaries; use 'full' only when recursive node data is required."),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, _ := req.GetArguments()["nodeIds"].([]interface{})
		nodeIDs := toStringSlice(raw)
		params := map[string]interface{}{}
		if detail, ok := req.GetArguments()["detail"].(string); ok && detail != "" {
			params["detail"] = detail
		}
		resp, err := node.Send(ctx, "get_nodes_info", nodeIDs, params)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("get_design_context",
		mcp.WithDescription("Get design context for a Figma node. Local fallback accepts official-style parameters and also preserves legacy local options like depth/detail."),
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
		mcp.WithBoolean("disableCodeConnect",
			mcp.Description("Accepted for official compatibility. Local fallback may treat this as a no-op."),
		),
		mcp.WithBoolean("excludeScreenshot",
			mcp.Description("When true, omit screenshots from the response where supported."),
		),
		mcp.WithBoolean("forceCode",
			mcp.Description("Accepted for official compatibility."),
		),
		mcp.WithNumber("depth",
			mcp.Description("How many levels deep to traverse (default 2)"),
		),
		mcp.WithString("detail",
			mcp.Description("Property verbosity: minimal (id/name/type/bounds only), compact (+fills/strokes/opacity), full (everything, default)"),
		),
		mcp.WithBoolean("dedupe_components",
			mcp.Description("When true, INSTANCE nodes are serialized compactly (mainComponentId + componentProperties + overrides array of differing text/nested content) and unique component definitions are collected once in a top-level componentDefs map. Highly token-efficient for screens with many repeated component instances."),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := collectArgs(req,
			"fileKey",
			"nodeId",
			"clientFrameworks",
			"clientLanguages",
			"disableCodeConnect",
			"excludeScreenshot",
			"forceCode",
		)
		if d, ok := req.GetArguments()["depth"].(float64); ok && d > 0 {
			params["depth"] = d
		}
		if det, ok := req.GetArguments()["detail"].(string); ok && det != "" {
			params["detail"] = det
		}
		if dd, ok := req.GetArguments()["dedupe_components"].(bool); ok && dd {
			params["dedupeComponents"] = true
		}
		resp, err := node.Send(ctx, "get_design_context", nil, params)
		return renderDesignContextResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("get_figjam",
		mcp.WithDescription("Generate UI code for a FigJam node in the current desktop file. This is the local fallback counterpart to the official Figma MCP get_figjam tool."),
		mcp.WithString("fileKey",
			mcp.Description("Compatibility parameter for the official Figma MCP. In local fallback mode this may be ignored."),
		),
		mcp.WithString("nodeId",
			mcp.Description("Optional node ID in colon format e.g. '4029:12345'. Defaults to the root node when omitted."),
		),
		mcp.WithBoolean("includeImagesOfNodes",
			mcp.Description("When true, include images of nodes in the response where supported."),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := node.Send(ctx, "get_figjam", nil, collectArgs(req, "fileKey", "nodeId", "includeImagesOfNodes"))
		return renderFigJamResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("search_nodes",
		mcp.WithDescription("Search for nodes by name substring and/or type within a subtree. Avoids dumping the entire document tree."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Name substring to match (case-insensitive)"),
		),
		mcp.WithString("nodeId",
			mcp.Description("Scope search to this subtree (default: current page), colon format e.g. '4029:12345'"),
		),
		mcp.WithArray("types",
			mcp.Description("Filter by Figma node type e.g. ['TEXT', 'FRAME', 'COMPONENT']"),
			mcp.WithStringItems(),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results to return (default: 50)"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := map[string]interface{}{
			"query": req.GetArguments()["query"],
		}
		if id, ok := req.GetArguments()["nodeId"].(string); ok && id != "" {
			params["nodeId"] = id
		}
		if raw, ok := req.GetArguments()["types"].([]interface{}); ok && len(raw) > 0 {
			params["types"] = raw
		}
		if limit, ok := req.GetArguments()["limit"].(float64); ok && limit > 0 {
			params["limit"] = limit
		}
		resp, err := node.Send(ctx, "search_nodes", nil, params)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("scan_text_nodes",
		mcp.WithDescription("Scan all TEXT nodes in a subtree. Useful for extracting all copy from a component or frame."),
		mcp.WithString("nodeId",
			mcp.Required(),
			mcp.Description("Root node ID to scan from, colon format e.g. '4029:12345'"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nodeID, _ := req.GetArguments()["nodeId"].(string)
		resp, err := node.Send(ctx, "scan_text_nodes", nil, map[string]interface{}{"nodeId": nodeID})
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("scan_nodes_by_types",
		mcp.WithDescription("Find all nodes matching specific types (e.g. FRAME, COMPONENT, INSTANCE) within a subtree."),
		mcp.WithString("nodeId",
			mcp.Required(),
			mcp.Description("Root node ID to scan from, colon format e.g. '4029:12345'"),
		),
		mcp.WithArray("types",
			mcp.Required(),
			mcp.Description("Node types to find e.g. ['FRAME', 'COMPONENT', 'INSTANCE']"),
			mcp.WithStringItems(),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nodeID, _ := req.GetArguments()["nodeId"].(string)
		raw, _ := req.GetArguments()["types"].([]interface{})
		resp, err := node.Send(ctx, "scan_nodes_by_types", nil, map[string]interface{}{
			"nodeId": nodeID,
			"types":  raw,
		})
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("get_reactions",
		mcp.WithDescription("Get prototype/interaction reactions on a node. Useful for understanding interactive prototypes."),
		mcp.WithString("nodeId",
			mcp.Required(),
			mcp.Description("Node ID in colon format e.g. '4029:12345'"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nodeID, _ := req.GetArguments()["nodeId"].(string)
		resp, err := node.Send(ctx, "get_reactions", []string{nodeID}, nil)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("get_viewport",
		mcp.WithDescription("Get the current Figma viewport: scroll center, zoom level, and visible bounds."),
	), makeHandler(node, "get_viewport", nil, nil))

	s.AddTool(mcp.NewTool("get_fonts",
		mcp.WithDescription("List all fonts used in the current page, sorted by usage frequency. Useful for understanding typography without scanning all text nodes."),
	), makeHandler(node, "get_fonts", nil, nil))
}

func collectArgs(req mcp.CallToolRequest, keys ...string) map[string]interface{} {
	params := map[string]interface{}{}
	for _, key := range keys {
		if value, ok := req.GetArguments()[key]; ok {
			params[key] = value
		}
	}
	return params
}
