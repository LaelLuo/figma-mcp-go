# figma-mcp-go Fallback Alignment Design

> Status: Approved by user on 2026-04-08
> Scope: Position `figma-mcp-go` as the local fallback MCP for the official Figma MCP, while aligning the most important local behaviors and interfaces with official Figma MCP expectations used by Codex Figma skills.

## Goal

Turn `figma-mcp-go` into a practical local fallback for the official Figma MCP server:

- Keep the official remote `figma` MCP untouched and available.
- Keep `figma-mcp-go` as a separate MCP server with its current name.
- Make `figma-mcp-go` clearly documented as the fallback when the official MCP is limited by plan quotas, remote-only capabilities, auth friction, or service instability.
- Align local tool names, parameters, and response shapes with official Figma MCP where that helps Codex Figma skills continue working after switching away from the official server.

This is not a full local reimplementation of every official cloud capability. It is a skills-first compatibility strategy.

## User Intent

The primary objective is not "clone the official Figma MCP" in isolation. The real target is:

1. Preserve the official Figma MCP for cloud-native capabilities.
2. Use `figma-mcp-go` as the fallback when official usage is blocked or impractical.
3. Make the fallback behave closely enough to the official server that Codex Figma skills can still execute meaningful workflows with minimal prompt or tool-path breakage.

## Product Positioning

`figma-mcp-go` should be described as:

> A local Figma desktop/plugin bridge MCP that complements the official Figma MCP. Use the official MCP for cloud-native capabilities and team-library workflows; use `figma-mcp-go` as the local fallback for live desktop/plugin read-write access when the official MCP is limited or unavailable.

## Naming Decision

- Keep the existing MCP server name: `figma-mcp-go`
- Do not rename it to `figma-local` or `figma-desktop`
- Clarify its role through documentation, installation examples, and compatibility tables instead of renaming

## Architectural Decision

Do not proxy or wrap the official Figma MCP inside `figma-mcp-go`.

The two MCP servers remain separate:

- `figma`: official remote MCP
- `figma-mcp-go`: local fallback MCP

Rationale:

- The user explicitly wants to preserve direct access to the official MCP.
- Cloud-only tools like `whoami`, `create_new_file`, `search_design_system`, Code Connect, and generated design capture should remain the responsibility of the official MCP.
- A proxy architecture would add complexity, hide capability boundaries, and make failures harder to reason about.

## Compatibility Strategy

`figma-mcp-go` should evolve using a layered interface model:

### Layer 1: Official-style fallback tools

These are the tools that should be added or reshaped first because Codex Figma skills depend on them heavily:

- `use_figma`
- `get_metadata`
- `get_design_context`
- `get_screenshot`
- `get_figjam`
- `get_variable_defs`

These tools should prefer official-style naming, parameter names, and response shapes.

### Layer 2: Legacy local tools

Keep the current 58 local primitive tools, including examples such as:

- `create_frame`
- `set_text`
- `set_fills`
- `scan_nodes_by_types`
- `create_variable`
- `bind_variable_to_node`

These remain supported, but they are no longer the main compatibility surface for Codex Figma skills.

### Layer 3: Explicit capability gaps

Some official Figma MCP tools should be documented as official-only rather than locally reimplemented poorly:

- `whoami`
- `create_new_file`
- `search_design_system`
- `generate_figma_design`
- `get_code_connect_map`
- `add_code_connect_map`
- `get_code_connect_suggestions`
- `send_code_connect_mappings`
- `create_design_system_rules`

The fallback server must not pretend to provide equivalent cloud capabilities when it cannot.

## README / Documentation Changes

The README should be reorganized around the fallback role.

### Required README changes

1. Add a top-level statement that the recommended setup is to keep both MCP servers:
   - Official `figma` MCP for cloud-native and team-library workflows
   - `figma-mcp-go` for local desktop/plugin fallback workflows
2. Add a "When to use which MCP" section.
3. Add a compatibility matrix that makes capability boundaries obvious.
4. Split tool documentation into:
   - Official-style tools
   - Legacy local tools
   - Official-only capabilities not replicated locally
5. Explain clearly that `figma-mcp-go` is especially useful when the official Figma MCP is blocked by Starter plan call limits.

## Recommended Compatibility Matrix

The README should include a matrix equivalent to the following:

| Tool / Capability | Official MCP | figma-mcp-go | Strategy |
| --- | --- | --- | --- |
| `use_figma` | Supported | Target strong support | Local fallback priority |
| `get_screenshot` | Supported | Supported | Local output should match official-style image result |
| `get_metadata` | Supported | Target compatible support | Reshape local output toward official expectations |
| `get_design_context` | Supported | Target partial-to-strong compatible support | First version may be degraded but stable |
| `get_figjam` | Supported | Target support | Local implementation acceptable |
| `get_variable_defs` | Supported | Target support | Clarify local-vs-library visibility |
| `whoami` | Supported | Not equivalent locally | Official-only |
| `create_new_file` | Supported | Not supported locally | Official-only |
| `search_design_system` | Supported | Not equivalent locally | Official-only |
| `generate_figma_design` | Supported | Not supported locally | Official-only |
| Code Connect tools | Supported | Not equivalent locally | Official-only |
| `create_design_system_rules` | Supported | Not equivalent locally | Official-only |

## Why a Skills-first Plan Is Better Than Full Tool Mirroring

Codex Figma skills do not primarily depend on the current local primitive tool set. They assume the presence of a smaller group of higher-level official-style tools.

The highest-value compatibility surface is therefore the subset of tools most used by:

- `figma-use`
- `figma-implement-design`
- `figma-generate-design`
- `figma-code-connect-components`
- `figma-create-design-system-rules`

This means local parity work should focus on the tools those skills assume first, rather than attempting a ground-up local clone of every official cloud capability.

## Phase 1 Scope

Phase 1 should focus on the official-style read + execution tools that make the fallback actually useful to Codex Figma skills.

### Phase 1 tools

1. `use_figma`
2. `get_metadata`
3. `get_design_context`
4. `get_figjam`
5. `get_variable_defs`

`get_screenshot` has already been improved and should be treated as part of this compatibility surface.

## Phase 1 Tool Design

### 1. `use_figma`

#### Purpose

Provide the official-style JavaScript execution entry point expected by `figma-use` and other skills.

#### Input compatibility target

Accept:

- `code`
- `description`
- optional `fileKey`
- optional `skillNames`

#### Local behavior

- Execute JavaScript against the currently open desktop file via the plugin bridge
- Treat `fileKey` as a compatibility parameter; in local mode it may be ignored or validated loosely
- Support top-level `await`
- Return the script `return` value as structured output
- Propagate execution errors clearly
- Preserve atomic execution semantics as much as the underlying bridge allows

#### Implementation note

This requires a new plugin-bridge execution path for arbitrary JavaScript, not just the existing catalog of predefined RPC actions.

### 2. `get_metadata`

#### Purpose

Provide the high-level node map used by skills for structure discovery and recovery when `get_design_context` is too large.

#### Input compatibility target

Accept official-style parameters:

- optional `fileKey`
- optional `nodeId`
- optional `clientFrameworks`
- optional `clientLanguages`

#### Local behavior

- Use current desktop file context
- Default to the current page or current selection when appropriate
- Return a lightweight scenegraph view
- Prefer official-style sparse structure over the current large local JSON tree

#### Implementation note

The local bridge should expose a dedicated lightweight traversal mode instead of reusing the current heavy metadata result directly.

### 3. `get_design_context`

#### Purpose

Support `figma-implement-design` and related skills by returning a stable design context envelope.

#### Input compatibility target

Accept official-style parameters:

- `fileKey`
- `nodeId`
- optional `clientFrameworks`
- optional `clientLanguages`
- optional `disableCodeConnect`
- optional `excludeScreenshot`
- optional `forceCode`

#### Local behavior

- Return a compatibility-oriented design context object
- Include:
  - structural node context
  - screenshot when available
  - local reference snippet or summarized implementation guidance when possible
- If the node is too large, return a clear degraded response that points the caller to `get_metadata` plus targeted child-node fetches

#### Implementation note

This should not attempt to perfectly duplicate the official remote code generation semantics in Phase 1. The goal is stable fallback behavior for skills, not fake parity.

### 4. `get_figjam`

#### Purpose

Provide a local equivalent for official FigJam node extraction workflows.

#### Input compatibility target

Accept:

- `fileKey`
- `nodeId`
- optional `includeImagesOfNodes`

#### Local behavior

- Only work when the current file is actually a FigJam file
- Return a structured representation of the targeted FigJam subtree
- Fail clearly on non-FigJam files

### 5. `get_variable_defs`

#### Purpose

Expose local variable collections, modes, and values in a more official-style shape.

#### Input compatibility target

Accept:

- `fileKey`
- `nodeId`
- optional `clientFrameworks`
- optional `clientLanguages`

#### Local behavior

- Return visible local variable definitions from the current file
- Make it explicit that library/team variables discovered by official `search_design_system` are not equivalent to local variable visibility
- Do not pretend to replace remote design system search

## Phase 1 Non-goals

Do not implement these in Phase 1:

- `whoami`
- `create_new_file`
- `search_design_system`
- `generate_figma_design`
- `get_code_connect_map`
- `add_code_connect_map`
- `get_code_connect_suggestions`
- `send_code_connect_mappings`
- `create_design_system_rules`

These remain official-MCP-first capabilities.

## Breaking-change Policy

Because the user chose a strong official-alignment direction, controlled breaking changes are acceptable in the official-style compatibility layer.

Rules:

- Existing legacy local tools remain available.
- New official-style tools should be treated as the preferred surface for future workflows.
- Existing same-name local tools may be adjusted if needed to move closer to official behavior, as long as README and tests are updated together.

## Testing Strategy

Phase 1 validation should include:

1. Unit tests for official-style parameter parsing
2. Unit tests for response-shape compatibility
3. Integration tests for local plugin-bridge execution where feasible
4. Smoke tests against a live Figma desktop session for:
   - `get_screenshot`
   - `get_metadata`
   - `use_figma`
5. Explicit tests for large-node degradation behavior in `get_design_context`

## Risks

### Risk 1: False parity

If the fallback claims equivalence for cloud-only tools, skills and users will trust behavior that is not real.

Mitigation:

- Document official-only tools explicitly
- Return unsupported errors instead of fake success

### Risk 2: `get_design_context` becomes too ambitious

Trying to exactly mirror the official remote output too early will slow delivery and produce brittle behavior.

Mitigation:

- Ship a stable compatible envelope first
- Prefer degraded-but-honest output over fragile fake parity

### Risk 3: Existing users rely on local primitive tools

Aggressive rewrites could break current workflows.

Mitigation:

- Keep legacy tools
- Add official-style tools incrementally
- Document the new preferred path

## Recommended Next Step

After this spec, create an implementation plan for Phase 1 only:

- README repositioning as official fallback
- compatibility matrix
- `use_figma`
- `get_metadata`
- `get_design_context`
- `get_figjam`
- `get_variable_defs`

That plan should treat the local fallback + official coexistence model as the baseline architecture.
