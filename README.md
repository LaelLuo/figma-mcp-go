# figma-mcp-go

Local fallback MCP for the official Figma MCP

`figma-mcp-go` is a local Figma desktop/plugin bridge MCP that complements the official Figma MCP. Keep the official remote MCP for cloud-native and team-library workflows, and use `figma-mcp-go` as the fallback when the official server is rate-limited, unavailable, or unsuitable for local desktop/plugin workflows.

**Highlights**
- Keep the official `figma` MCP untouched and available
- No Figma REST API token required for local fallback workflows
- No official MCP rate-limit dependency for plugin-bridge reads and writes
- Live desktop/plugin read-write access for styles, variables, components, prototypes, and content
- Built-in prompts and compatibility work aimed at Codex Figma skill fallback usage

https://github.com/user-attachments/assets/17bda971-0e83-4f18-8758-8ac2b8dcba62

---

## Why this exists

The official Figma MCP is still the right first choice for cloud-native capabilities. The problem is that many users, especially on Starter / View / Collab plans, can hit tool-call limits very quickly.

When that happens, a local fallback is often more useful than trying to replace the official server entirely.

`figma-mcp-go` exists to cover the local side:

- read and write the currently open desktop file through the plugin bridge
- keep working when official MCP tool-call limits are exhausted
- support live local workflows that do not need cloud-only APIs

Typical trigger:

| Plan | Limit |
|------|-------|
| Starter / View / Collab | **6 tool calls/month** |
| Pro / Org (Dev seat) | 200 tool calls/day |
| Enterprise | 600 tool calls/day |

If you're experimenting with AI-driven design workflows, that can disappear in minutes.

This project does **not** proxy the official MCP. The recommended model is to keep both servers:

- `figma`: official remote MCP
- `figma-mcp-go`: local fallback MCP

---

## Installation & Setup

Install via `npx` — no build step required. You can keep the official MCP and `figma-mcp-go` side-by-side.

[![Watch the video](https://img.youtube.com/vi/DjqyU0GKv9k/sddefault.jpg)](https://youtu.be/DjqyU0GKv9k)

### Recommended setup: keep both MCP servers

Use the official MCP when you need:

- cloud-native Figma capabilities
- team libraries and remote design-system search
- Code Connect workflows
- generated design capture and account-aware operations

Use `figma-mcp-go` when you need:

- local desktop/plugin read-write access
- a fallback after official MCP Starter-plan limits are exhausted
- direct manipulation of the file currently open in Figma Desktop
- local screenshots, exports, and automation that do not depend on REST APIs

### 1. Configure `figma-mcp-go` in your AI tool

**Claude Code CLI**
```bash
claude mcp add -s project figma-mcp-go -- npx -y @vkhanhqui/figma-mcp-go@latest
```

**.mcp.json** (Claude and other MCP-compatible tools)
```json
{
  "mcpServers": {
    "figma-mcp-go": {
      "command": "npx",
      "args": ["-y", "@vkhanhqui/figma-mcp-go"]
    }
  }
}
```

**.vscode/mcp.json** (Cursor / VS Code / GitHub Copilot)
```json
{
  "servers": {
    "figma-mcp-go": {
      "type": "stdio",
      "command": "npx",
      "args": [
        "-y",
        "@vkhanhqui/figma-mcp-go"
      ]
    }
  }
}
```

### 2. Install the Figma plugin

1. In Figma Desktop: **Plugins → Development → Import plugin from manifest**
2. Select `manifest.json` from the [plugin.zip](https://github.com/vkhanhqui/figma-mcp-go/releases)
3. Run the plugin inside any Figma file

---

## When to use which MCP

| Situation | Recommended MCP |
| --- | --- |
| Need `whoami`, `create_new_file`, `generate_figma_design`, Code Connect, or design-system/library search | Official `figma` MCP |
| Need local desktop/plugin read-write access to the file you already have open | `figma-mcp-go` |
| Hit Starter-plan tool-call limits on the official MCP | `figma-mcp-go` fallback |
| Need screenshot/export behavior without depending on remote quotas | `figma-mcp-go` |

## Compatibility Matrix

| Tool / Capability | Official MCP | figma-mcp-go | Strategy |
| --- | --- | --- | --- |
| `use_figma` | Supported | In progress | Local fallback priority |
| `get_screenshot` | Supported | Supported | Return MCP image content plus structured metadata |
| `get_metadata` | Supported | In progress | Align local output toward official-style expectations |
| `get_design_context` | Supported | In progress | Stable fallback first, full parity later |
| `get_figjam` | Supported | In progress | Local implementation when current file is FigJam |
| `get_variable_defs` | Supported | In progress | Current-file variables only |
| `whoami` | Supported | Not equivalent locally | Official-only |
| `create_new_file` | Supported | Not supported locally | Official-only |
| `search_design_system` | Supported | Not equivalent locally | Official-only |
| `generate_figma_design` | Supported | Not supported locally | Official-only |
| Code Connect tools | Supported | Not equivalent locally | Official-only |
| `create_design_system_rules` | Supported | Not equivalent locally | Official-only |

## Available Tools

### Official-style fallback tools

These tools are the primary compatibility surface for Codex Figma skills and for switching away from the official MCP when needed:

| Tool | Status | Notes |
|------|--------|-------|
| `use_figma` | In progress | Official-style local JS execution entry point |
| `get_metadata` | In progress | Moving toward official-style metadata recovery flow |
| `get_design_context` | In progress | Moving toward official-style design context envelope |
| `get_screenshot` | Supported | Returns MCP image content plus structured metadata |
| `get_figjam` | In progress | Local FigJam fallback |
| `get_variable_defs` | In progress | Official-style args; local variable visibility only |

### Legacy local tools

The tools below remain supported and are useful for direct local automation, but they are not the main compatibility surface for official Figma MCP workflows.

### Write — Create

| Tool | Description |
|------|-------------|
| `create_frame` | Create a frame with optional auto-layout, fill, and parent |
| `create_rectangle` | Create a rectangle with optional fill and corner radius |
| `create_ellipse` | Create an ellipse or circle |
| `create_text` | Create a text node (font loaded automatically) |
| `import_image` | Decode base64 image and place it as a rectangle fill |
| `create_component` | Convert an existing FRAME node into a reusable component |

### Write — Modify

| Tool | Description |
|------|-------------|
| `set_text` | Update text content of an existing TEXT node |
| `set_fills` | Set solid fill color (hex) on a node |
| `set_strokes` | Set solid stroke color and weight on a node |
| `set_opacity` | Set opacity of one or more nodes (0 = transparent, 1 = opaque) |
| `set_corner_radius` | Set corner radius — uniform or per-corner |
| `set_auto_layout` | Set or update auto-layout (flex) properties on a frame |
| `move_nodes` | Move nodes to an absolute x/y position |
| `resize_nodes` | Resize nodes by width and/or height |
| `rename_node` | Rename a node |
| `clone_node` | Clone a node, optionally repositioning or reparenting |

### Write — Delete

| Tool | Description |
|------|-------------|
| `delete_nodes` | Delete one or more nodes permanently |

### Write — Prototype

| Tool | Description |
|------|-------------|
| `set_reactions` | Set prototype reactions (triggers + actions) on a node; mode `replace` or `append` |
| `remove_reactions` | Remove all or specific reactions by zero-based index from a node |

### Write — Styles

| Tool | Description |
|------|-------------|
| `create_paint_style` | Create a named paint style with a solid color |
| `create_text_style` | Create a named text style with font, size, and spacing |
| `create_effect_style` | Create a named effect style (drop shadow, inner shadow, blur) |
| `create_grid_style` | Create a named layout grid style (columns, rows, or grid) |
| `update_paint_style` | Rename or recolor an existing paint style |
| `apply_style_to_node` | Apply an existing local style to a node, linking it to that style |
| `delete_style` | Delete any style (paint, text, effect, or grid) by ID |

### Write — Variables

| Tool | Description |
|------|-------------|
| `create_variable_collection` | Create a new local variable collection with an optional initial mode |
| `add_variable_mode` | Add a new mode to an existing collection (e.g. Light/Dark) |
| `create_variable` | Create a variable (COLOR/FLOAT/STRING/BOOLEAN) in a collection |
| `set_variable_value` | Set a variable's value for a specific mode |
| `bind_variable_to_node` | Bind a local variable to a node property |
| `delete_variable` | Delete a variable or an entire collection |

### Write — Components & Navigation

| Tool | Description |
|------|-------------|
| `navigate_to_page` | Switch the active Figma page by ID or name |
| `group_nodes` | Group two or more nodes into a GROUP |
| `ungroup_nodes` | Ungroup GROUP nodes, moving children to the parent |
| `swap_component` | Swap the main component of an INSTANCE node |
| `detach_instance` | Detach component instances, converting them to plain frames |

### Read — Document & Selection

| Tool | Description |
|------|-------------|
| `get_document` | Full current page tree |
| `get_metadata` | File name, pages, current page |
| `get_pages` | All pages (IDs + names) — lightweight, no tree loading |
| `get_selection` | Currently selected nodes |
| `get_node` | Single node by ID |
| `get_nodes_info` | Multiple nodes by ID |
| `get_design_context` | Depth-limited tree with `detail` level (`minimal`/`compact`/`full`) |
| `search_nodes` | Find nodes by name substring and/or type within a subtree |
| `scan_text_nodes` | All text nodes in a subtree |
| `scan_nodes_by_types` | Nodes matching given type list |
| `get_viewport` | Current viewport center, zoom, and visible bounds |

### Read — Styles & Variables

| Tool | Description |
|------|-------------|
| `get_styles` | Paint, text, effect, and grid styles |
| `get_variable_defs` | Variable collections and values |
| `get_local_components` | All components + component sets with variant properties |
| `get_annotations` | Dev-mode annotations |
| `get_fonts` | All fonts used on the current page, sorted by frequency |
| `get_reactions` | Prototype/interaction reactions on a node |

### Export

| Tool | Description |
|------|-------------|
| `get_screenshot` | Returns MCP image content plus structured metadata for exported nodes |
| `save_screenshots` | Export images to disk (server-side, no API call) |
| `export_frames_to_pdf` | Export multiple frames as a single multi-page PDF file saved to disk |
| `export_tokens` | Export design tokens (variables + paint styles) as JSON or CSS |

`save_screenshots` and `export_frames_to_pdf` accept relative paths inside the current working directory, or explicit absolute paths when you want to write elsewhere on disk.

### MCP Prompts

| Prompt | Description |
|--------|-------------|
| `read_design_strategy` | Best practices for reading Figma designs |
| `design_strategy` | Best practices for creating and modifying designs |
| `text_replacement_strategy` | Chunked approach for replacing text across a design |
| `annotation_conversion_strategy` | Convert manual annotations to native Figma annotations |
| `swap_overrides_instances` | Transfer overrides between component instances |
| `reaction_to_connector_strategy` | Map prototype reactions into interaction flow diagrams |

### Official-only capabilities not replicated locally

`figma-mcp-go` does not try to fake cloud-only capabilities. Keep using the official `figma` MCP for:

- `whoami`
- `create_new_file`
- `search_design_system`
- `generate_figma_design`
- `get_code_connect_map`
- `add_code_connect_map`
- `get_code_connect_suggestions`
- `send_code_connect_mappings`
- `create_design_system_rules`

---

## Related Projects

- [magic-spells/figma-mcp-bridge](https://github.com/magic-spells/figma-mcp-bridge)
- [grab/cursor-talk-to-figma-mcp](https://github.com/grab/cursor-talk-to-figma-mcp)
- [gethopp/figma-mcp-bridge](https://github.com/gethopp/figma-mcp-bridge)

---

## Contributing

Issues and PRs are welcome.

## Star History

<a href="https://www.star-history.com/?repos=vkhanhqui%2Ffigma-mcp-go&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=vkhanhqui/figma-mcp-go&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=vkhanhqui/figma-mcp-go&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=vkhanhqui/figma-mcp-go&type=date&legend=top-left" />
 </picture>
</a>
