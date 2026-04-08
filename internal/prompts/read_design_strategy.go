package prompts

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func addReadDesignStrategy(s *server.MCPServer) {
	s.AddPrompt(mcp.NewPrompt("read_design_strategy",
		mcp.WithPromptDescription("Best practices for reading Figma designs with the official Figma MCP and figma-mcp-go fallback"),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return mcp.NewGetPromptResult(
			"Best practices for reading Figma designs",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent(`To effectively read a Figma design, treat the official Figma MCP as the preferred path and figma-mcp-go as the local fallback when the official server is rate-limited, unavailable, or unsuitable for desktop/plugin-local workflows.

1. Choose the server intentionally:
   - Use the official Figma MCP first for cloud-backed capabilities, design-system discovery, Code Connect, and other remote-only workflows.
   - Use figma-mcp-go fallback for local file inspection, local screenshots, local variable reads, and desktop/plugin execution.

2. Start with get_metadata to understand the current document or selection.
   - In fallback mode, get_metadata returns an official-style compatibility envelope with metadataText, pages, kind, and nodeId.
   - Treat metadataText as an XML-ish local summary, not as cloud-complete metadata.

3. Use get_pages to list pages without loading their full trees.

4. Use get_design_context for the main structured read path.
   - In fallback mode, get_design_context returns text content plus structuredContent, and can include an image when screenshots are enabled.
   - detail=minimal: id/name/type/bounds only (~5% tokens)
   - detail=compact: + fills/strokes/opacity (~30% tokens)
   - detail=full: everything, default (100% tokens)
   - dedupe_components=true: repeated INSTANCE nodes are collapsed to compact stubs and unique component structures are collected once.
   - If the selected node is too large, expect an honest degraded response that tells you to use get_metadata plus narrower child reads.

5. For screens with many repeated components, the recommended flow is:
   a. get_design_context(depth=2, detail=minimal, dedupe_components=true)
   b. Inspect componentDefs in the response
   c. Read componentProperties / overrides on instance stubs
   d. Drill into specific nodes with get_node or get_nodes_info only when necessary

6. Use search_nodes, scan_text_nodes, and scan_nodes_by_types to narrow the search before pulling more detail.

7. Call get_styles and get_variable_defs once per session to understand the local design system surface.
   - In fallback mode, get_variable_defs only exposes variables defined in the current file.
   - Local variable visibility is not the same thing as official remote library or design-system search.

8. Call get_figjam only for FigJam files.
   - In fallback mode, get_figjam rejects design files with a clear error and returns an official-style compatibility payload for FigJam nodes.

9. Use get_fonts, get_viewport, get_reactions, and get_screenshot as supporting tools.
   - Call get_screenshot last and only when visual confirmation is needed.

10. Node IDs use colon format: 4029:12345 — never use hyphens.`),
				),
			},
		), nil
	})
}
