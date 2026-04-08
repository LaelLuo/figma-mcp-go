# Figma MCP Fallback Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship Phase 1 of the official-Figma-MCP fallback strategy so `figma-mcp-go` can coexist with the official MCP and provide usable local fallback coverage for the highest-value Codex Figma skill workflows.

**Architecture:** Keep the official remote MCP untouched and evolve `figma-mcp-go` as a separate local fallback. Rework the Go and plugin bridge layers so the public compatibility surface moves toward official tool names, parameters, and response envelopes, while legacy local tools remain available where they still add value. Prefer honest degraded responses over fake cloud parity.

**Tech Stack:** Go 1.26, `mark3labs/mcp-go`, Figma Plugin API, TypeScript, Bun, Vite, Svelte

---

## File Structure

### Public docs and package metadata

- Modify: `README.md`
  - Reposition the project as the official Figma MCP fallback instead of a generic “58 tools” pitch.
  - Add the dual-MCP recommendation, compatibility matrix, and official-only gaps.
- Modify: `server.json`
  - Update the MCP registry description so it explicitly says this server is the local fallback when official MCP is unavailable or rate-limited.
- Modify: `npm/package.json`
  - Align the npm package description with the fallback positioning.

### Go-side MCP contract and transport

- Modify: `internal/tools.go`
  - Register any new official-style tool entry points and keep the public registration order coherent.
- Modify: `internal/tools_read_document.go`
  - Reshape `get_metadata` and `get_design_context`, and add `get_figjam`.
- Modify: `internal/tools_read_styles.go`
  - Extend `get_variable_defs` to accept official-style arguments and return the compatibility envelope.
- Create: `internal/tools_use_figma.go`
  - Add the official-style `use_figma` tool without overloading existing write primitives.
- Modify: `internal/tools_test.go`
  - Add response-rendering tests for official-style content and structured payloads.
- Modify: `internal/tools_handler_test.go`
  - Add MCP registration and smoke-path coverage for the new/reshaped tool contracts.
- Modify: `internal/schema.go`
  - Validate official-style arguments for `use_figma`, `get_metadata`, `get_design_context`, `get_figjam`, and `get_variable_defs`.
- Modify: `internal/schema_test.go`
  - Add positive and negative validation cases for the new arguments.
- Modify: `internal/bridge.go`
  - Introduce a timeout policy table so long-running fallback reads and `use_figma` do not fail prematurely.
- Modify: `internal/types.go`
  - Extend bridge payload shapes only if a new request/response field is needed for progress, screenshots, or `use_figma` execution results.

### Plugin-side official compatibility handlers

- Modify: `plugin/src/read-handlers.ts`
  - Route official-style read requests to the right handler path.
- Modify: `plugin/src/read-document.ts`
  - Implement official-style `get_metadata`, `get_design_context`, and `get_figjam` semantics.
- Modify: `plugin/src/read-styles.ts`
  - Implement official-style `get_variable_defs` semantics.
- Create: `plugin/src/use-figma.ts`
  - Execute arbitrary JavaScript for `use_figma`, including top-level `await`, explicit return values, and useful error propagation.
- Modify: `plugin/src/main.ts`
  - Wire the new `use_figma` handler into the request dispatcher.
- Create: `plugin/src/use-figma.test.ts`
  - Unit-test the execution wrapper and error handling.
- Modify: `plugin/src/read-styles.test.ts`
  - Add `get_variable_defs` compatibility coverage.
- Modify: `plugin/src/serializers.test.ts`
  - Add any helper coverage needed by the new design-context / metadata envelope.

### Prompt and guidance refresh

- Modify: `internal/prompts/read_design_strategy.go`
  - Stop teaching the old local-only `get_metadata` / `get_design_context` contract as the preferred path.
- Modify: `internal/prompts/design_strategy.go`
  - Update guidance to match the coexistence model and official-style fallback semantics.
- Modify: `internal/prompts/design_token_generation_strategy.go`
  - Clarify that local `get_variable_defs` sees current-file variables, not remote library search.
- Modify: `internal/prompts/style_audit_strategy.go`
  - Update variable-reading instructions to the new contract.
- Modify: `internal/prompts/reaction_to_connector_strategy.go`
  - Remove stale assumptions about the old `get_design_context` payload if needed.
- Modify: `internal/prompts/prompts_test.go`
  - Update prompt snapshot/substring assertions if they encode old wording.

## Chunk 1: Public Positioning and Contract Baseline

### Task 1: Reposition the public docs around the fallback model

**Files:**
- Modify: `README.md`
- Modify: `server.json`
- Modify: `npm/package.json`
- Reference: `docs/superpowers/specs/2026-04-08-figma-fallback-alignment-design.md`

- [ ] **Step 1: Capture the current public wording before editing**

Run:

```powershell
rg -n "No Rate Limits|58 tools|Installation & Setup|Available Tools|description" README.md server.json npm/package.json
```

Expected: current copy still presents the project primarily as an unlimited generic Figma MCP server.

- [ ] **Step 2: Rewrite the README introduction and setup guidance**

Make these changes in `README.md`:

```md
- Add a top-level “Use both MCP servers” recommendation.
- Add a “When to use which MCP” section.
- Add a compatibility matrix matching the spec.
- Split tool docs into:
  - Official-style fallback tools
  - Legacy local tools
  - Official-only capabilities
- Explicitly call out Starter plan tool-call limits as the main fallback trigger.
```

- [ ] **Step 3: Align registry/package descriptions**

Update `server.json` and `npm/package.json` descriptions so they describe `figma-mcp-go` as:

```text
Local fallback MCP for the official Figma MCP, providing desktop/plugin read-write access when the official server is rate-limited, unavailable, or unsuitable for local workflows.
```

- [ ] **Step 4: Verify the public docs now reflect the fallback architecture**

Run:

```powershell
rg -n "When to use which MCP|Official-style fallback tools|Official-only|fallback|official Figma MCP" README.md server.json npm/package.json
```

Expected: the new dual-MCP language and fallback wording appear in all three files.

- [ ] **Step 5: Commit the docs-only slice**

Run:

```powershell
git add -- README.md server.json npm/package.json
git diff --cached
git commit -m ':memo: docs(readme): 明确 figma-mcp-go 作为官方 Figma MCP fallback' -m 'WHAT: 重写 README 与包元数据，明确官方 MCP 与本地 fallback 并存的推荐模型。' -m 'WHY: 没有清晰的公开定位，后续实现和用户接入都容易误把本地桥接能力当成官方云能力替代。' -m 'HOW: 新增兼容矩阵、适用场景、官方专属能力说明，并同步 server/npm 描述。'
```

Expected: one docs-only commit with no code changes.

## Chunk 2: Go-side Official Compatibility Surface

### Task 2: Add the official-style tool contracts and timeout policy in Go

**Files:**
- Create: `internal/tools_use_figma.go`
- Modify: `internal/tools.go`
- Modify: `internal/schema.go`
- Modify: `internal/schema_test.go`
- Modify: `internal/bridge.go`
- Modify: `internal/tools_handler_test.go`

- [ ] **Step 1: Write validation tests for the new contracts**

Add failing tests in `internal/schema_test.go` for:

```go
func TestValidateRPC_UseFigma(t *testing.T)
func TestValidateRPC_GetMetadata_OfficialArgs(t *testing.T)
func TestValidateRPC_GetDesignContext_OfficialArgs(t *testing.T)
func TestValidateRPC_GetFigjam(t *testing.T)
func TestValidateRPC_GetVariableDefs_OfficialArgs(t *testing.T)
```

Cover:

```go
- use_figma requires params["code"] as non-empty string
- get_metadata/get_design_context/get_figjam/get_variable_defs accept fileKey/nodeId/clientFrameworks/clientLanguages
- get_design_context validates excludeScreenshot/forceCode/disableCodeConnect as booleans when present
- get_figjam validates includeImagesOfNodes as boolean when present
- nodeId stays colon-format only
```

- [ ] **Step 2: Run the targeted Go tests and confirm they fail for the missing validation paths**

Run:

```powershell
go test ./internal -run 'TestValidateRPC_(UseFigma|GetMetadata_OfficialArgs|GetDesignContext_OfficialArgs|GetFigjam|GetVariableDefs_OfficialArgs)$'
```

Expected: FAIL because the new validation logic does not exist yet.

- [ ] **Step 3: Implement schema validation and register `use_figma`**

Implement:

```go
// internal/tools_use_figma.go
s.AddTool(mcp.NewTool("use_figma", ...), handler)

// internal/schema.go
case "use_figma":
    code, _ := params["code"].(string)
    if strings.TrimSpace(code) == "" { return "code is required" }
```

Also accept the official compatibility parameters for:

```go
get_metadata
get_design_context
get_figjam
get_variable_defs
```

- [ ] **Step 4: Add a request-timeout policy instead of hard-coded special cases**

Replace the current `get_document`-only timeout branch in `internal/bridge.go` with a helper such as:

```go
func timeoutForRequest(requestType string) time.Duration {
	switch requestType {
	case "get_document", "get_design_context", "get_figjam", "use_figma":
		return 90 * time.Second
	default:
		return 30 * time.Second
	}
}
```

This keeps the “Figma may hang for a while” cases explicit and testable.

- [ ] **Step 5: Add MCP smoke-path coverage for the new registration**

Extend `internal/tools_handler_test.go` so it exercises:

```go
callTool(t, s, "use_figma", map[string]any{
    "code": "return figma.root.name",
    "description": "Read file name",
    "fileKey": "dummy",
})

callTool(t, s, "get_figjam", map[string]any{
    "fileKey": "dummy",
    "nodeId": "0:1",
    "includeImagesOfNodes": true,
})
```

- [ ] **Step 6: Run the affected Go tests and confirm they pass**

Run:

```powershell
go test ./internal -run 'Test(ValidateRPC|Handlers_)'
```

Expected: PASS.

- [ ] **Step 7: Commit the Go contract slice**

Run:

```powershell
git add -- internal/tools_use_figma.go internal/tools.go internal/schema.go internal/schema_test.go internal/bridge.go internal/tools_handler_test.go
git diff --cached
git commit -m ':sparkles: feat(mcp): 增加官方风格 fallback 工具契约' -m 'WHAT: 增加 use_figma 工具入口，并为 Phase 1 官方风格工具补齐参数校验、注册与超时策略。' -m 'WHY: 没有稳定的 MCP 契约和超时策略，后续插件实现即使完成也会被参数不兼容或超时打断。' -m 'HOW: 新增 use_figma 注册，扩展 schema 校验，并把长耗时请求迁移到显式 timeout policy。'
```

### Task 3: Reshape Go response rendering for official-style read tools

**Files:**
- Modify: `internal/tools_read_document.go`
- Modify: `internal/tools_read_styles.go`
- Modify: `internal/tools_test.go`
- Modify: `internal/tools_handler_test.go`

- [ ] **Step 1: Write failing response-shape tests**

Add tests in `internal/tools_test.go` that assert:

```go
func TestRenderMetadataResponse_ReturnsTextEnvelope(t *testing.T)
func TestRenderDesignContextResponse_IncludesStructuredContentAndOptionalImage(t *testing.T)
func TestRenderVariableDefsResponse_PreservesStructuredContent(t *testing.T)
```

Expect:

```go
- get_metadata returns text content, not just JSON-marshalled legacy output
- get_design_context can include text + image + structured content
- get_variable_defs returns structured content in a stable compatibility envelope
```

- [ ] **Step 2: Run the targeted test set and confirm it fails**

Run:

```powershell
go test ./internal -run 'TestRender(Metadata|DesignContext|VariableDefs)Response'
```

Expected: FAIL because the custom renderers do not exist yet.

- [ ] **Step 3: Implement official-style handlers for the reshaped tools**

In `internal/tools_read_document.go` and `internal/tools_read_styles.go`:

```go
- get_metadata: pass through official args and render the returned metadata text/envelope directly
- get_design_context: pass through official args and use a specialized renderer
- get_figjam: register the new tool and render a structured official-style result
- get_variable_defs: accept official args even if fileKey is local-compat only
```

Do not keep the old “depth/detail/dedupe only” path as the sole public contract. If legacy local options remain, document them as compatibility extensions rather than the primary surface.

- [ ] **Step 4: Make `get_design_context` screenshot-capable on the Go side**

Add a renderer that can translate plugin output like:

```json
{
  "metadata": {...},
  "code": "...",
  "screenshot": {
    "mimeType": "image/png",
    "base64": "..."
  }
}
```

into MCP content blocks:

```text
1. Text summary / code block / guidance
2. Optional image content
3. structuredContent containing the full compatibility payload
```

- [ ] **Step 5: Re-run the focused tests**

Run:

```powershell
go test ./internal -run 'Test(Render|Handlers_).*'
```

Expected: PASS.

- [ ] **Step 6: Commit the Go response-layer slice**

Run:

```powershell
git add -- internal/tools_read_document.go internal/tools_read_styles.go internal/tools_test.go internal/tools_handler_test.go
git diff --cached
git commit -m ':recycle: refactor(read): 调整官方风格工具返回结构' -m 'WHAT: 调整 get_metadata、get_design_context、get_figjam、get_variable_defs 的 Go 侧参数透传与返回渲染。' -m 'WHY: 仅有参数兼容还不够，Codex Figma skills 还依赖更接近官方 MCP 的文本、图片和 structured content 组合。' -m 'HOW: 为官方风格工具增加专用 renderer，并把截图与结构化数据封装成 MCP 友好的内容块。'
```

## Chunk 3: Plugin Bridge Compatibility Implementation

### Task 4: Implement `use_figma` execution in the plugin

**Files:**
- Create: `plugin/src/use-figma.ts`
- Create: `plugin/src/use-figma.test.ts`
- Modify: `plugin/src/main.ts`

- [ ] **Step 1: Write failing tests for the execution wrapper**

Create `plugin/src/use-figma.test.ts` with cases for:

```ts
it("runs async code and returns the result")
it("surfaces thrown errors with message text")
it("supports top-level await semantics via async wrapper")
it("returns undefined cleanly when code does not explicitly return")
```

- [ ] **Step 2: Run the plugin test subset and confirm it fails**

Run:

```powershell
bun test plugin/src/use-figma.test.ts
```

Expected: FAIL because the executor file does not exist yet.

- [ ] **Step 3: Implement the execution wrapper**

In `plugin/src/use-figma.ts`, use an async function wrapper similar to:

```ts
export async function runUseFigma(code: string, request: any) {
  const runner = new Function(
    "figma",
    "request",
    `"use strict"; return (async () => { ${code} })();`,
  );
  return await runner(figma, request);
}
```

Requirements:

```ts
- Support top-level await
- Return plain JSON-serializable values when possible
- Surface thrown Error.message values
- Do not silently swallow exceptions
```

- [ ] **Step 4: Wire `use_figma` into `plugin/src/main.ts`**

Extend the dispatcher so `use_figma` is handled before falling through to legacy read/write handlers.

- [ ] **Step 5: Re-run the plugin tests**

Run:

```powershell
bun test plugin/src/use-figma.test.ts
```

Expected: PASS.

- [ ] **Step 6: Commit the plugin execution slice**

Run:

```powershell
git add -- plugin/src/use-figma.ts plugin/src/use-figma.test.ts plugin/src/main.ts
git diff --cached
git commit -m ':sparkles: feat(plugin): 支持 use_figma 任意脚本执行' -m 'WHAT: 在插件桥接层增加 use_figma 脚本执行能力，支持 async/await 和显式返回值。' -m 'WHY: Codex 的 figma-use skill 依赖官方 use_figma 入口，没有这层执行能力，本地 fallback 仍无法承接关键工作流。' -m 'HOW: 新增执行包装器并在主调度入口接线，补齐错误传播与返回值测试。'
```

### Task 5: Implement official-style read handlers in the plugin

**Files:**
- Modify: `plugin/src/read-handlers.ts`
- Modify: `plugin/src/read-document.ts`
- Modify: `plugin/src/read-styles.ts`
- Modify: `plugin/src/read-styles.test.ts`
- Modify: `plugin/src/serializers.test.ts`

- [ ] **Step 1: Add failing tests for the new read envelopes**

Add tests that cover:

```ts
- get_metadata returns an official-style lightweight metadata overview
- get_design_context returns a compatibility envelope with context + optional screenshot payload
- get_figjam rejects non-FigJam files
- get_variable_defs returns collections/modes/variables in the new envelope
```

- [ ] **Step 2: Run the focused plugin test set and confirm it fails**

Run:

```powershell
bun test plugin/src/read-styles.test.ts plugin/src/serializers.test.ts
```

Expected: FAIL because the compatibility envelopes are not implemented yet.

- [ ] **Step 3: Rework `get_metadata` away from the current file-summary-only contract**

Implement a lightweight metadata result that can support official-style structure recovery. A minimal acceptable Phase 1 shape is:

```json
{
  "fileKey": null,
  "nodeId": "0:1",
  "kind": "page-overview",
  "metadataText": "<page id=\"0:1\" name=\"Page\">...</page>",
  "pages": [{ "id": "0:1", "name": "Page" }]
}
```

This keeps the official “XML-ish metadata” expectation alive without pretending to be cloud-complete.

- [ ] **Step 4: Implement a degraded-but-honest `get_design_context` envelope**

Return a stable shape such as:

```json
{
  "fileKey": null,
  "nodeId": "123:456",
  "name": "Selected Frame",
  "code": "Local fallback summary or implementation guidance",
  "metadata": {
    "source": "figma-mcp-go",
    "degraded": false
  },
  "context": {...},
  "screenshot": {
    "mimeType": "image/png",
    "base64": "..."
  }
}
```

Rules:

```text
- If the node is too large, set degraded=true and include a message directing callers to get_metadata + targeted child fetches.
- Respect excludeScreenshot when present.
- Treat disableCodeConnect as accepted but no-op metadata in local fallback mode.
```

- [ ] **Step 5: Implement `get_figjam` and official-style `get_variable_defs`**

Requirements:

```ts
- get_figjam: require figma.editorType === "figjam"; otherwise throw a clear error
- get_figjam: default nodeId to "0:1" if omitted
- get_variable_defs: keep local variable scanning logic, but wrap it in a compatibility payload that states these are current-file variables only
```

- [ ] **Step 6: Re-run plugin verification**

Run:

```powershell
bun test
bun run build
```

Expected: PASS for both tests and build.

- [ ] **Step 7: Commit the plugin read slice**

Run:

```powershell
git add -- plugin/src/read-handlers.ts plugin/src/read-document.ts plugin/src/read-styles.ts plugin/src/read-styles.test.ts plugin/src/serializers.test.ts
git diff --cached
git commit -m ':sparkles: feat(plugin): 对齐官方风格读取工具语义' -m 'WHAT: 在插件侧实现官方风格的 get_metadata、get_design_context、get_figjam 与 get_variable_defs 兼容返回。' -m 'WHY: 没有插件侧真实语义支撑，Go 层即使改了参数和渲染也只能输出旧数据，无法承接官方 skills。' -m 'HOW: 重构读取处理器，增加 FigJam 判断、设计上下文降级策略和本地变量可见性说明，并跑通 bun test/build。'
```

## Chunk 4: Prompt Alignment and End-to-End Verification

### Task 6: Refresh built-in prompts so the repo teaches the new model

**Files:**
- Modify: `internal/prompts/read_design_strategy.go`
- Modify: `internal/prompts/design_strategy.go`
- Modify: `internal/prompts/design_token_generation_strategy.go`
- Modify: `internal/prompts/style_audit_strategy.go`
- Modify: `internal/prompts/reaction_to_connector_strategy.go`
- Modify: `internal/prompts/prompts_test.go`

- [ ] **Step 1: Update prompt text to match the coexistence model**

Make the prompts teach:

```text
- Official MCP remains preferred for cloud-only capabilities
- figma-mcp-go is the local fallback
- get_metadata/get_design_context/get_variable_defs now follow the official-style compatibility path
- local variable visibility is not the same thing as remote library/design-system search
```

- [ ] **Step 2: Add or update prompt assertions**

In `internal/prompts/prompts_test.go`, assert that key prompt strings reference:

```text
"official Figma MCP"
"fallback"
"get_figjam"
"get_metadata"
```

- [ ] **Step 3: Run the prompt-specific tests**

Run:

```powershell
go test ./internal/prompts
```

Expected: PASS.

- [ ] **Step 4: Commit the prompt refresh**

Run:

```powershell
git add -- internal/prompts/read_design_strategy.go internal/prompts/design_strategy.go internal/prompts/design_token_generation_strategy.go internal/prompts/style_audit_strategy.go internal/prompts/reaction_to_connector_strategy.go internal/prompts/prompts_test.go
git diff --cached
git commit -m ':memo: docs(prompts): 更新官方 MCP 与 fallback 协作指引' -m 'WHAT: 更新内置 prompts，使其教学路径与官方 MCP + figma-mcp-go fallback 的新模型一致。' -m 'WHY: 如果 prompts 继续教授旧语义，仓库内置策略会主动把调用方带回错误的工具认知。' -m 'HOW: 重写关键提示词并补齐测试断言，强调官方优先、fallback 兜底和本地能力边界。'
```

### Task 7: Run end-to-end verification and record the manual smoke matrix

**Files:**
- Modify: `README.md` (only if manual verification reveals doc gaps)
- Reference: `docs/superpowers/specs/2026-04-08-figma-fallback-alignment-design.md`
- Reference: `docs/superpowers/plans/2026-04-08-figma-fallback-phase-1-implementation.md`

- [ ] **Step 1: Run the full automated verification suite**

Run:

```powershell
go test ./...
```

Run:

```powershell
bun test
```

Run:

```powershell
bun run build
```

Expected: PASS on all three commands.

- [ ] **Step 2: Run live fallback smoke tests against an open Figma desktop session**

Manual checks:

```text
1. get_screenshot still returns MCP image content and structuredContent
2. use_figma can read figma.root.name and mutate a harmless node property
3. get_metadata returns the new metadata overview contract
4. get_design_context works on a medium frame and degrades honestly on a very large node
5. get_figjam rejects design files and succeeds in a FigJam file
6. get_variable_defs returns local collections with the compatibility warning/metadata
```

- [ ] **Step 3: Rebuild and refresh the local binary if verification passes**

Run:

```powershell
go build -o D:\Applications\Scoop\shims\figma-mcp-go.exe ./cmd/figma-mcp-go
```

Expected: fresh binary is produced with the Phase 1 behavior.

- [ ] **Step 4: Create the final integration commit**

Run:

```powershell
git add -- README.md server.json npm/package.json internal/tools.go internal/tools_read_document.go internal/tools_read_styles.go internal/tools_use_figma.go internal/tools_test.go internal/tools_handler_test.go internal/schema.go internal/schema_test.go internal/bridge.go internal/types.go plugin/src/main.ts plugin/src/read-handlers.ts plugin/src/read-document.ts plugin/src/read-styles.ts plugin/src/use-figma.ts plugin/src/use-figma.test.ts plugin/src/read-styles.test.ts plugin/src/serializers.test.ts internal/prompts/read_design_strategy.go internal/prompts/design_strategy.go internal/prompts/design_token_generation_strategy.go internal/prompts/style_audit_strategy.go internal/prompts/reaction_to_connector_strategy.go internal/prompts/prompts_test.go
git diff --cached
git commit -m ':rocket: feat(fallback): 完成 Phase 1 官方 MCP 对齐' -m 'WHAT: 完成 figma-mcp-go Phase 1 fallback 对齐，实现官方风格核心工具、文档定位、插件执行和提示词迁移。' -m 'WHY: 这一步把仓库从“本地 58 工具集合”推进到“官方 MCP 不可用时可接手高价值 workflow 的 fallback”。' -m 'HOW: 同步改造 README、Go MCP 契约、插件桥接、提示词与测试，并通过自动化与人工 smoke test 双重验证。'
```

## Acceptance Checklist

- [ ] README 明确推荐双 MCP 并存，而不是替代官方 MCP
- [ ] `use_figma` 可执行任意 JS，并支持 top-level `await`
- [ ] `get_metadata`、`get_design_context`、`get_figjam`、`get_variable_defs` 接受官方风格参数
- [ ] `get_design_context` 可返回文本 + 可选截图 + structured content
- [ ] 长耗时工具不会因为 30 秒默认超时而误失败
- [ ] 内置 prompts 不再教授旧的单机语义作为唯一推荐路径
- [ ] `go test ./...`、`bun test`、`bun run build` 通过
- [ ] Live Figma smoke tests 通过

## Notes for the Implementer

- Keep the official `figma` MCP untouched in user docs and examples. This project is the fallback, not the proxy.
- Do not fake cloud-only tools. Unsupported should stay unsupported.
- Prefer compatibility envelopes that are explicit about local limitations over responses that look official but hide capability gaps.
- If a same-name tool must change semantics, update README, tests, and built-in prompts in the same slice so the repo stays self-consistent.

Plan complete and saved to `docs/superpowers/plans/2026-04-08-figma-fallback-phase-1-implementation.md`. Ready to execute?
