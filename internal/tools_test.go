package internal

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// ── renderResponse ────────────────────────────────────────────────────────────

func TestRenderResponse_TransportError(t *testing.T) {
	result, err := renderResponse(BridgeResponse{}, fmt.Errorf("connection failed"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for transport error")
	}
}

func TestRenderResponse_PluginError(t *testing.T) {
	result, err := renderResponse(BridgeResponse{Error: "node not found"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for plugin error")
	}
}

func TestRenderResponse_Success(t *testing.T) {
	result, err := renderResponse(BridgeResponse{Data: map[string]any{"id": "1:1"}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected IsError=false for successful response")
	}
	if len(result.Content) == 0 {
		t.Error("expected non-empty content")
	}
}

func TestRenderResponse_NilData(t *testing.T) {
	result, err := renderResponse(BridgeResponse{Data: nil}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected IsError=false for nil data")
	}
}

func TestRenderScreenshotResponse_SuccessIncludesImageContent(t *testing.T) {
	first := base64.StdEncoding.EncodeToString([]byte("first-image"))
	second := base64.StdEncoding.EncodeToString([]byte("second-image"))

	result, err := renderScreenshotResponse(BridgeResponse{
		Data: map[string]any{
			"exports": []any{
				map[string]any{
					"nodeId":   "1:1",
					"nodeName": "Frame 1",
					"format":   "PNG",
					"base64":   first,
					"width":    float64(100),
					"height":   float64(80),
				},
				map[string]any{
					"nodeId":   "1:2",
					"nodeName": "Frame 2",
					"format":   "PNG",
					"base64":   second,
					"width":    float64(120),
					"height":   float64(90),
				},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected successful screenshot response")
	}
	if result.StructuredContent == nil {
		t.Fatal("expected structured content for screenshot metadata")
	}
	if len(result.Content) != 3 {
		t.Fatalf("expected 3 content blocks, got %d", len(result.Content))
	}

	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected first content block to be text, got %T", result.Content[0])
	}
	if text.Text == "" {
		t.Fatal("expected non-empty text summary")
	}

	image1, ok := result.Content[1].(mcp.ImageContent)
	if !ok {
		t.Fatalf("expected second content block to be image, got %T", result.Content[1])
	}
	if image1.Data != first {
		t.Fatalf("unexpected first image payload")
	}
	if image1.MIMEType != "image/png" {
		t.Fatalf("unexpected first image MIME type: %s", image1.MIMEType)
	}

	image2, ok := result.Content[2].(mcp.ImageContent)
	if !ok {
		t.Fatalf("expected third content block to be image, got %T", result.Content[2])
	}
	if image2.Data != second {
		t.Fatalf("unexpected second image payload")
	}
	if image2.MIMEType != "image/png" {
		t.Fatalf("unexpected second image MIME type: %s", image2.MIMEType)
	}
}

func TestRenderMetadataResponse_ReturnsTextEnvelope(t *testing.T) {
	result, err := renderMetadataResponse(BridgeResponse{
		Data: map[string]any{
			"nodeId":       "1:1",
			"metadataText": "<metadata source=\"figma-mcp-go\" />",
			"source":       "figma-mcp-go",
		},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected successful metadata response")
	}
	if result.StructuredContent == nil {
		t.Fatal("expected structured content for metadata response")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}

	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	if text.Text != "<metadata source=\"figma-mcp-go\" />" {
		t.Fatalf("unexpected metadata text: %q", text.Text)
	}
}

func TestRenderDesignContextResponse_IncludesStructuredContentAndOptionalImage(t *testing.T) {
	image := base64.StdEncoding.EncodeToString([]byte("context-image"))

	result, err := renderDesignContextResponse(BridgeResponse{
		Data: map[string]any{
			"nodeId": "1:1",
			"name":   "Card",
			"code":   "{\n  \"notice\": \"fallback\"\n}",
			"metadata": map[string]any{
				"source":   "figma-mcp-go",
				"degraded": true,
				"message":  "Local fallback response",
			},
			"screenshot": map[string]any{
				"mimeType": "image/png",
				"base64":   image,
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected successful design context response")
	}
	if result.StructuredContent == nil {
		t.Fatal("expected structured content for design context response")
	}
	if len(result.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(result.Content))
	}

	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected first content block to be text, got %T", result.Content[0])
	}
	if text.Text == "" {
		t.Fatal("expected non-empty design context text")
	}

	renderedImage, ok := result.Content[1].(mcp.ImageContent)
	if !ok {
		t.Fatalf("expected second content block to be image, got %T", result.Content[1])
	}
	if renderedImage.Data != image {
		t.Fatalf("unexpected image payload")
	}
	if renderedImage.MIMEType != "image/png" {
		t.Fatalf("unexpected image MIME type: %s", renderedImage.MIMEType)
	}
}

func TestRenderVariableDefsResponse_PreservesStructuredContent(t *testing.T) {
	result, err := renderVariableDefsResponse(BridgeResponse{
		Data: map[string]any{
			"source":  "figma-mcp-go",
			"scope":   "local-file",
			"message": "Local fallback exposes variables defined in this file.",
			"collections": []any{
				map[string]any{"id": "col:1", "name": "Brand"},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected successful variable defs response")
	}
	if result.StructuredContent == nil {
		t.Fatal("expected structured content for variable defs response")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}

	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	if text.Text == "" {
		t.Fatal("expected non-empty variable defs text")
	}
}

// ── toStringSlice ─────────────────────────────────────────────────────────────

func TestToStringSlice(t *testing.T) {
	raw := []any{"a", "b", 42, nil, "c"}
	got := toStringSlice(raw)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, s := range want {
		if got[i] != s {
			t.Errorf("[%d] = %q, want %q", i, got[i], s)
		}
	}
}

func TestToStringSlice_Empty(t *testing.T) {
	if got := toStringSlice(nil); len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

// ── coalesce ─────────────────────────────────────────────────────────────────

func TestCoalesce(t *testing.T) {
	if got := coalesce("a", "b"); got != "a" {
		t.Errorf("coalesce(a,b) = %q, want a", got)
	}
	if got := coalesce("", "b"); got != "b" {
		t.Errorf("coalesce('',b) = %q, want b", got)
	}
	if got := coalesce("", ""); got != "" {
		t.Errorf("coalesce('','') = %q, want empty", got)
	}
}

// ── inferFormat ───────────────────────────────────────────────────────────────

func TestInferFormat(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"image.png", "PNG"},
		{"image.PNG", "PNG"},
		{"icon.svg", "SVG"},
		{"photo.jpg", "JPG"},
		{"photo.jpeg", "JPG"},
		{"doc.pdf", "PDF"},
		{"noext", ""},
		{"file.txt", ""},
		{"file.gif", ""},
	}
	for _, c := range cases {
		if got := inferFormat(c.path); got != c.want {
			t.Errorf("inferFormat(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

// ── resolveOutputPath ─────────────────────────────────────────────────────────

func TestResolveOutputPath_Relative(t *testing.T) {
	dir := t.TempDir()
	got, err := resolveOutputPath("output/img.png", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "output", "img.png")
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestResolveOutputPath_AbsoluteInsideDir(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "sub", "img.png")
	got, err := resolveOutputPath(abs, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != abs {
		t.Errorf("got %s, want %s", got, abs)
	}
}

func TestResolveOutputPath_Traversal_Blocked(t *testing.T) {
	dir := t.TempDir()
	_, err := resolveOutputPath("../outside.png", dir)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestResolveOutputPath_AbsoluteOutsideDir_Blocked(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(filepath.Dir(dir), "outside.png")
	got, err := resolveOutputPath(outside, dir)
	if err != nil {
		t.Fatalf("unexpected error for absolute output path: %v", err)
	}
	if got != outside {
		t.Errorf("got %s, want %s", got, outside)
	}
}

// ── mustBeInsideDir ───────────────────────────────────────────────────────────

func TestMustBeInsideDir_Allowed(t *testing.T) {
	dir := t.TempDir()
	inner := filepath.Join(dir, "a", "b", "c.txt")
	got, err := mustBeInsideDir(inner, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != inner {
		t.Errorf("got %s, want %s", got, inner)
	}
}

func TestMustBeInsideDir_Blocked(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Dir(dir)
	_, err := mustBeInsideDir(filepath.Join(parent, "evil.txt"), dir)
	if err == nil {
		t.Error("expected error for path outside directory")
	}
}

func TestMustBeInsideDir_SameDir(t *testing.T) {
	dir := t.TempDir()
	_, err := mustBeInsideDir(filepath.Join(dir, "file.txt"), dir)
	if err != nil {
		t.Fatalf("file directly in workDir should be allowed: %v", err)
	}
}

// ── parseSaveItem ─────────────────────────────────────────────────────────────

func TestParseSaveItem_Valid(t *testing.T) {
	raw := map[string]any{
		"nodeId":     "1:1",
		"outputPath": "out/img.png",
		"format":     "PNG",
		"scale":      float64(2),
	}
	item, err := parseSaveItem(raw)
	if err != nil {
		t.Fatalf("parseSaveItem: %v", err)
	}
	if item.NodeID != "1:1" {
		t.Errorf("NodeID = %q, want 1:1", item.NodeID)
	}
	if item.OutputPath != "out/img.png" {
		t.Errorf("OutputPath = %q, want out/img.png", item.OutputPath)
	}
	if item.Scale != 2 {
		t.Errorf("Scale = %v, want 2", item.Scale)
	}
}

func TestParseSaveItem_UnmarshalError(t *testing.T) {
	// A channel cannot be marshaled to JSON.
	_, err := parseSaveItem(make(chan int))
	if err == nil {
		t.Error("expected marshal error for non-JSON-serialisable value")
	}
}

// ── extractScreenshotExport ───────────────────────────────────────────────────

func TestExtractScreenshotExport_Valid(t *testing.T) {
	data := map[string]any{
		"exports": []any{
			map[string]any{
				"nodeId":   "1:1",
				"nodeName": "Frame",
				"base64":   "abc123",
				"width":    float64(100),
				"height":   float64(200),
			},
		},
	}
	export, err := extractScreenshotExport(data)
	if err != nil {
		t.Fatalf("extractScreenshotExport: %v", err)
	}
	if export.NodeID != "1:1" {
		t.Errorf("NodeID = %q, want 1:1", export.NodeID)
	}
	if export.Width != 100 || export.Height != 200 {
		t.Errorf("dimensions = %vx%v, want 100x200", export.Width, export.Height)
	}
}

func TestExtractScreenshotExport_EmptyExports(t *testing.T) {
	data := map[string]any{"exports": []any{}}
	_, err := extractScreenshotExport(data)
	if err == nil {
		t.Error("expected error for empty exports array")
	}
}

func TestExtractScreenshotExport_MissingExports(t *testing.T) {
	_, err := extractScreenshotExport(map[string]any{})
	if err == nil {
		t.Error("expected error when exports key is missing")
	}
}

// ── writeBase64 ───────────────────────────────────────────────────────────────

func TestWriteBase64_WritesFile(t *testing.T) {
	dir := t.TempDir()
	data := []byte("hello figma")
	b64 := base64.StdEncoding.EncodeToString(data)

	path := filepath.Join(dir, "out.png")
	n, err := writeBase64(b64, path)
	if err != nil {
		t.Fatalf("writeBase64: %v", err)
	}
	if n != len(data) {
		t.Errorf("bytes written = %d, want %d", n, len(data))
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(data) {
		t.Error("file content does not match original data")
	}
}

func TestWriteBase64_CreatesIntermediateDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "out.png")
	b64 := base64.StdEncoding.EncodeToString([]byte("x"))

	if _, err := writeBase64(b64, path); err != nil {
		t.Fatalf("writeBase64 should create dirs: %v", err)
	}
}

func TestWriteBase64_RejectsExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.png")
	b64 := base64.StdEncoding.EncodeToString([]byte("data"))

	// Create the file first.
	if _, err := writeBase64(b64, path); err != nil {
		t.Fatalf("first write: %v", err)
	}

	// Second write must fail.
	_, err := writeBase64(b64, path)
	if err == nil {
		t.Error("expected error when file already exists")
	}
}

func TestWriteBase64_InvalidBase64(t *testing.T) {
	dir := t.TempDir()
	_, err := writeBase64("not-valid-base64!!!", filepath.Join(dir, "out.png"))
	if err == nil {
		t.Error("expected error for invalid base64 input")
	}
}
