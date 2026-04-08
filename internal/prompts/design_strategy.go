package prompts

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func addDesignStrategy(s *server.MCPServer) {
	s.AddPrompt(mcp.NewPrompt("design_strategy",
		mcp.WithPromptDescription("Best practices for working with Figma designs"),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return mcp.NewGetPromptResult(
			"Best practices for working with Figma designs",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent(`When working with Figma designs, use a dual-MCP model:

1. Choose the right server before acting.
   - Prefer the official Figma MCP for cloud-only workflows such as Code Connect, design-system/library search, file creation, and other remote capabilities.
   - Use figma-mcp-go as the fallback for local desktop/plugin reads and writes when the official Figma MCP is unavailable, rate-limited, or not suitable for local workflows.

2. Start with document understanding.
   - Call get_metadata to understand the current document, page, or selection.
   - Use get_pages to list available pages.
   - If you are in a FigJam file, use get_figjam instead of treating it like a design file.

3. Read before you write.
   - Use get_design_context as the primary structured read path.
   - In fallback mode, get_design_context returns official-style text plus structuredContent and may include screenshots.
   - Expect honest degraded responses for very large nodes rather than fake cloud parity.

4. Build with explicit hierarchy.
   - Create parent frames first, then add child nodes.
   - Use descriptive names and group related elements in frames.
   - Verify important creations with get_node or get_nodes_info.

5. Reuse local styles and variables where possible.
   - Call get_styles and get_variable_defs before introducing new local paint/text patterns.
   - Remember that fallback get_variable_defs only sees current-file variables; it does not replace official design-system search.

6. Use write tools deliberately.
   - Use create_frame, create_text, create_rectangle, and related primitives for local construction.
   - Use use_figma when you need official-style plugin execution semantics for targeted JavaScript in the current file.
   - Use set_text, set_fills, set_strokes, move_nodes, and resize_nodes for adjustments.

7. Keep the visual hierarchy intentional.
   - Position elements in logical reading order.
   - Maintain consistent spacing and alignment.
   - Use typography and grouping to communicate structure clearly.

8. Treat fallback limitations as design constraints, not hidden implementation details.
   - If a workflow depends on remote libraries or cloud metadata, say so and prefer the official Figma MCP.
   - If a workflow is local-only, lean on figma-mcp-go and be explicit about what data is local versus remote.`),
				),
			},
		), nil
	})
}
