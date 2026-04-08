package prompts

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func addDesignTokenGenerationStrategy(s *server.MCPServer) {
	s.AddPrompt(mcp.NewPrompt("design_token_generation_strategy",
		mcp.WithPromptDescription("Extract raw values from an existing design and build a structured variable + style token system"),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return mcp.NewGetPromptResult(
			"Extract raw values from an existing design and build a structured variable + style token system",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent(`# Design Token Generation Strategy

Use the official Figma MCP when you need remote design-system discovery. Use figma-mcp-go fallback when you need to extract raw values from the current local file and build a structured token system from what is already in that file.

Scan an existing design to discover all unique colors, font sizes, spacing values, and radii,
then create a structured variable collection and named styles, and finally link nodes to them.

## Steps

### Phase 1 — Discovery

1. Call get_styles() to check what styles already exist (avoid duplicating them).
2. Call get_variable_defs() to check existing variables.
   - In fallback mode, these are current-file variables only.
   - This is not the same thing as official library or search_design_system coverage.
3. Call get_design_context(detail="compact") to scan the full node tree.
4. Collect unique values:
   - **Colors**: all unique hex fills and stroke colors across nodes.
   - **Font sizes**: all unique fontSize values on TEXT nodes.
   - **Spacing**: all unique itemSpacing, paddingTop/Right/Bottom/Left values on FRAME nodes.
   - **Radii**: all unique cornerRadius values.

### Phase 2 — Token naming

Map discovered values to semantic token names. Use this hierarchy:

**Colors** (variable collection "Primitives"):
- Sort colors by hue/lightness.
- Assign names like "Blue/100", "Blue/200", … "Blue/900", "Neutral/50", "Neutral/900", etc.
- Also create a "Semantic" collection with aliases: "Color/Primary", "Color/Background", "Color/Text", etc.

**Spacing** (variable collection "Spacing"):
- Name by scale: "Spacing/0" (0), "Spacing/1" (4px), "Spacing/2" (8px), "Spacing/3" (12px), …

**Radius** (variable collection "Radius"):
- Name: "Radius/None" (0), "Radius/SM" (4), "Radius/MD" (8), "Radius/LG" (16), "Radius/Full" (9999)

**Typography** (local text styles):
- Name: "Heading/H1", "Heading/H2", "Body/Regular", "Body/Small", "Label/Medium", etc.

Present the full token plan to the user for approval before creating anything.

If the user explicitly needs remote design-system alignment or library-backed token discovery, stop and prefer the official Figma MCP for that discovery step before using local fallback tools.

### Phase 3 — Creation

For each approved token:
- COLOR variables: create_variable_collection() → create_variable(type="COLOR") → set_variable_value()
- FLOAT variables: create_variable_collection() → create_variable(type="FLOAT") → set_variable_value()
- Text styles: create_text_style() with name, fontFamily, fontSize, lineHeight, letterSpacing
- Paint styles: create_paint_style() with name, color

### Phase 4 — Linking (optional, ask user)

After creating tokens, offer to link existing nodes:
- For nodes with raw fill colors that match a new variable → bind_variable_to_node(field="fillColor")
- For TEXT nodes with matching font styles → apply_style_to_node(styleId)

## Rules
- Never delete or overwrite existing styles/variables — only add new ones.
- Prefer variables over paint styles for colors that will need dark-mode variants.
- Always get user approval on the token plan (Phase 2) before executing Phase 3.
- Process in batches of 20 during linking phase.
`),
				),
			},
		), nil
	})
}
