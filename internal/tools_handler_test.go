package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// newTestServer returns an MCPServer with all tools + prompts registered
// against an Unknown-role Node (no real Figma connection).
func newTestServer(t *testing.T) (*server.MCPServer, *Node) {
	t.Helper()
	s := server.NewMCPServer("test", "0.0.1")
	node := NewNode(19940, "test")
	RegisterTools(s, node)
	RegisterPrompts(s)
	return s, node
}

// callTool dispatches a tool call through the server's full HandleMessage path.
// With an Unknown node, every call succeeds at the MCP level but returns
// an IsError=true tool result (no Figma connection).
func callTool(t *testing.T, s *server.MCPServer, name string, args map[string]any) {
	t.Helper()
	argsJSON, _ := json.Marshal(args)
	msg := fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`,
		name, argsJSON,
	)
	resp := s.HandleMessage(context.Background(), []byte(msg))
	if resp == nil {
		t.Errorf("HandleMessage returned nil for tool %q", name)
	}
}

// ── Registration smoke tests ──────────────────────────────────────────────────

func TestRegisterTools_Smoke(t *testing.T) {
	s := server.NewMCPServer("test", "0.0.1")
	RegisterTools(s, NewNode(19940, "test"))
}

func TestRegisterPrompts_Smoke(t *testing.T) {
	s := server.NewMCPServer("test", "0.0.1")
	RegisterPrompts(s)
}

// ── makeHandler ───────────────────────────────────────────────────────────────

func TestMakeHandler_UnknownNode(t *testing.T) {
	node := NewNode(19940, "test")
	handler := makeHandler(node, "get_document", nil, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when node has no Figma connection")
	}
}

// ── Read – no-param tools (all use makeHandler) ───────────────────────────────

func TestHandlers_NoParamReadTools(t *testing.T) {
	s, _ := newTestServer(t)
	noParamTools := []string{
		"get_document", "get_pages", "get_metadata", "get_selection",
		"get_viewport", "get_fonts", "get_styles", "get_variable_defs",
		"get_local_components", "get_annotations",
	}
	for _, name := range noParamTools {
		callTool(t, s, name, nil)
	}
}

// ── Read – param tools ────────────────────────────────────────────────────────

func TestHandlers_GetNode(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "get_node", map[string]any{"nodeId": "1:1"})
}

func TestHandlers_GetNodesInfo(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "get_nodes_info", map[string]any{"nodeIds": []string{"1:1", "2:2"}})
}

func TestHandlers_GetDesignContext(t *testing.T) {
	s, _ := newTestServer(t)
	// with all optional params
	callTool(t, s, "get_design_context", map[string]any{
		"depth": float64(2), "detail": "compact", "dedupe_components": true,
	})
	// with no params (defaults)
	callTool(t, s, "get_design_context", nil)
	// depth = 0 should be ignored (not passed through)
	callTool(t, s, "get_design_context", map[string]any{"depth": float64(0)})
	// official-style compatibility args
	callTool(t, s, "get_design_context", map[string]any{
		"fileKey":            "dummy",
		"nodeId":             "1:1",
		"clientFrameworks":   "react",
		"clientLanguages":    "typescript",
		"disableCodeConnect": true,
		"excludeScreenshot":  true,
		"forceCode":          true,
	})
}

func TestHandlers_GetMetadata_OfficialArgs(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "get_metadata", map[string]any{
		"fileKey":          "dummy",
		"nodeId":           "1:1",
		"clientFrameworks": "react",
		"clientLanguages":  "typescript",
	})
}

func TestHandlers_GetFigjam(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "get_figjam", map[string]any{
		"fileKey":              "dummy",
		"nodeId":               "0:1",
		"includeImagesOfNodes": true,
	})
}

func TestHandlers_UseFigma(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "use_figma", map[string]any{
		"code":        "return figma.root.name",
		"description": "Read file name",
		"fileKey":     "dummy",
		"skillNames":  "figma-use",
	})
}

func TestHandlers_GetVariableDefs_OfficialArgs(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "get_variable_defs", map[string]any{
		"fileKey":          "dummy",
		"nodeId":           "1:1",
		"clientFrameworks": "react",
		"clientLanguages":  "typescript",
	})
}

func TestHandlers_SearchNodes(t *testing.T) {
	s, _ := newTestServer(t)
	// all optional params present
	callTool(t, s, "search_nodes", map[string]any{
		"query":  "button",
		"nodeId": "1:1",
		"types":  []any{"TEXT", "FRAME"},
		"limit":  float64(25),
	})
	// minimal (query only)
	callTool(t, s, "search_nodes", map[string]any{"query": "icon"})
}

func TestHandlers_ScanTextNodes(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "scan_text_nodes", map[string]any{"nodeId": "1:1"})
}

func TestHandlers_ScanNodesByTypes(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "scan_nodes_by_types", map[string]any{
		"nodeId": "1:1",
		"types":  []any{"FRAME", "COMPONENT"},
	})
}

func TestHandlers_GetReactions(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "get_reactions", map[string]any{"nodeId": "1:1"})
}

// ── Read – export tools ───────────────────────────────────────────────────────

func TestHandlers_GetScreenshot(t *testing.T) {
	s, _ := newTestServer(t)
	// with format + scale
	callTool(t, s, "get_screenshot", map[string]any{
		"nodeIds": []any{"1:1"},
		"format":  "PNG",
		"scale":   float64(2),
	})
	// no params (exports current selection)
	callTool(t, s, "get_screenshot", nil)
}

// TestHandlers_SaveScreenshots exercises executeSaveScreenshots + saveScreenshotItem.
// With Unknown node, node.Send fails → error captured in result JSON (no panic).
func TestHandlers_SaveScreenshots(t *testing.T) {
	s, _ := newTestServer(t)

	// single item – reaches saveScreenshotItem → node.Send fails → error result
	callTool(t, s, "save_screenshots", map[string]any{
		"items": []any{
			map[string]any{"nodeId": "1:1", "outputPath": "out/screen.png"},
		},
	})

	// multiple items with default format + scale
	callTool(t, s, "save_screenshots", map[string]any{
		"format": "SVG",
		"scale":  float64(1),
		"items": []any{
			map[string]any{"nodeId": "1:1", "outputPath": "out/a.svg"},
			map[string]any{"nodeId": "2:2", "outputPath": "out/b.svg", "format": "PNG"},
		},
	})

	// item with explicit per-item format + scale
	callTool(t, s, "save_screenshots", map[string]any{
		"items": []any{
			map[string]any{"nodeId": "3:3", "outputPath": "out/c.jpg", "format": "JPG", "scale": float64(2)},
		},
	})
}

// ── Write – create tools ──────────────────────────────────────────────────────

func TestHandlers_WriteCreateTools(t *testing.T) {
	s, _ := newTestServer(t)

	callTool(t, s, "create_frame", map[string]any{
		"width": float64(100), "height": float64(100), "name": "Card",
		"layoutMode": "VERTICAL", "parentId": "1:1",
	})
	callTool(t, s, "create_frame", map[string]any{}) // minimal

	callTool(t, s, "create_rectangle", map[string]any{"fillColor": "#FF5733", "cornerRadius": float64(8)})
	callTool(t, s, "create_rectangle", map[string]any{})

	callTool(t, s, "create_ellipse", map[string]any{"width": float64(50), "height": float64(50)})
	callTool(t, s, "create_ellipse", map[string]any{})

	callTool(t, s, "create_text", map[string]any{
		"text": "Hello", "fontSize": float64(16), "fontFamily": "Inter", "fontStyle": "Bold",
		"fillColor": "#000000", "name": "Label",
	})

	// import_image with optional params
	callTool(t, s, "import_image", map[string]any{
		"imageData": "abc123", "x": float64(10), "y": float64(20),
		"width": float64(200), "height": float64(150),
		"name": "Hero", "scaleMode": "FILL", "parentId": "1:1",
	})
	// import_image minimal
	callTool(t, s, "import_image", map[string]any{"imageData": "abc123"})
}

// ── Write – modify tools ──────────────────────────────────────────────────────

func TestHandlers_WriteModifyTools(t *testing.T) {
	s, _ := newTestServer(t)

	callTool(t, s, "set_text", map[string]any{"nodeId": "1:1", "text": "Updated"})

	callTool(t, s, "set_fills", map[string]any{
		"nodeId": "1:1", "color": "#FF0000", "opacity": float64(0.8), "mode": "replace",
	})
	callTool(t, s, "set_fills", map[string]any{"nodeId": "1:1", "color": "#00FF00"}) // minimal

	callTool(t, s, "set_strokes", map[string]any{
		"nodeId": "1:1", "color": "#000000", "strokeWeight": float64(2), "mode": "append",
	})
	callTool(t, s, "set_strokes", map[string]any{"nodeId": "1:1", "color": "#000000"}) // minimal

	callTool(t, s, "move_nodes", map[string]any{"nodeIds": []any{"1:1"}, "x": float64(10), "y": float64(20)})
	callTool(t, s, "move_nodes", map[string]any{"nodeIds": []any{"1:1"}, "x": float64(5)}) // y omitted

	callTool(t, s, "resize_nodes", map[string]any{"nodeIds": []any{"1:1"}, "width": float64(300), "height": float64(200)})
	callTool(t, s, "resize_nodes", map[string]any{"nodeIds": []any{"1:1"}, "height": float64(100)}) // width omitted

	callTool(t, s, "rename_node", map[string]any{"nodeId": "1:1", "name": "New Name"})

	callTool(t, s, "clone_node", map[string]any{"nodeId": "1:1", "x": float64(50), "y": float64(50), "parentId": "2:2"})
	callTool(t, s, "clone_node", map[string]any{"nodeId": "1:1"}) // minimal

	callTool(t, s, "set_auto_layout", map[string]any{"nodeId": "1:1", "layoutMode": "HORIZONTAL"})

	callTool(t, s, "delete_nodes", map[string]any{"nodeIds": []any{"1:1", "2:2"}})
}

// ── Write – style tools ───────────────────────────────────────────────────────

func TestHandlers_WriteStyleTools(t *testing.T) {
	s, _ := newTestServer(t)

	callTool(t, s, "create_paint_style", map[string]any{"name": "Brand/Primary", "color": "#FF5733", "description": "Main brand color"})
	callTool(t, s, "create_text_style", map[string]any{"name": "Heading/H1"})
	callTool(t, s, "create_effect_style", map[string]any{"name": "Elevation/1", "type": "DROP_SHADOW"})
	callTool(t, s, "create_grid_style", map[string]any{"name": "Layout/12col", "pattern": "COLUMNS", "alignment": "STRETCH"})

	callTool(t, s, "update_paint_style", map[string]any{"styleId": "S:abc", "color": "#00FF00"})
	callTool(t, s, "update_paint_style", map[string]any{"styleId": "S:abc", "name": "Renamed"})

	callTool(t, s, "delete_style", map[string]any{"styleId": "S:abc"})
}

// ── Write – variable tools ────────────────────────────────────────────────────

func TestHandlers_WriteVariableTools(t *testing.T) {
	s, _ := newTestServer(t)

	callTool(t, s, "create_variable_collection", map[string]any{"name": "Brand", "initialModeName": "Light"})
	callTool(t, s, "add_variable_mode", map[string]any{"collectionId": "c1", "modeName": "Dark"})
	callTool(t, s, "create_variable", map[string]any{"name": "primary", "collectionId": "c1", "type": "COLOR"})
	callTool(t, s, "set_variable_value", map[string]any{"variableId": "v1", "modeId": "m1", "value": "#fff"})
	callTool(t, s, "delete_variable", map[string]any{"variableId": "v1"})
	callTool(t, s, "delete_variable", map[string]any{"collectionId": "c1"})
}

// ── Write – component tools ───────────────────────────────────────────────────

func TestHandlers_WriteComponentTools(t *testing.T) {
	s, _ := newTestServer(t)

	callTool(t, s, "swap_component", map[string]any{"nodeId": "1:1", "componentId": "2:2"})
	callTool(t, s, "detach_instance", map[string]any{"nodeIds": []any{"1:1", "2:2"}})
}

// ── Write – linked tools (apply_style_to_node, bind_variable_to_node) ─────────

func TestHandlers_LinkedTools(t *testing.T) {
	s, _ := newTestServer(t)

	callTool(t, s, "apply_style_to_node", map[string]any{"nodeId": "1:1", "styleId": "S:abc", "target": "fill"})
	callTool(t, s, "apply_style_to_node", map[string]any{"nodeId": "1:1", "styleId": "S:abc"}) // no target

	callTool(t, s, "bind_variable_to_node", map[string]any{"nodeId": "1:1", "variableId": "v1", "field": "fills"})
}
