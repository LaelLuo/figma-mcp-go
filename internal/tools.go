package internal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/vkhanhqui/figma-mcp-go/internal/prompts"
)

// RegisterTools registers all MCP tools on the server.
func RegisterTools(s *server.MCPServer, node *Node) {
	registerReadTools(s, node)
	registerUseFigmaTool(s, node)
	registerWriteTools(s, node)
}

// RegisterPrompts registers MCP prompts on the server.
func RegisterPrompts(s *server.MCPServer) {
	prompts.RegisterAll(s)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// makeHandler creates a simple tool handler with no parameters.
func makeHandler(node *Node, command string, nodeIDs []string, params map[string]interface{}) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := node.Send(ctx, command, nodeIDs, params)
		return renderResponse(resp, err)
	}
}

// renderResponse converts a BridgeResponse into an MCP tool result.
func renderResponse(resp BridgeResponse, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if resp.Error != "" {
		return mcp.NewToolResultError(resp.Error), nil
	}
	text, err := json.Marshal(resp.Data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(text)), nil
}

func renderScreenshotResponse(resp BridgeResponse, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if resp.Error != "" {
		return mcp.NewToolResultError(resp.Error), nil
	}

	exports, err := extractScreenshotExports(resp.Data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("parse screenshot response: %v", err)), nil
	}

	content := make([]mcp.Content, 0, len(exports)+1)
	content = append(content, mcp.TextContent{
		Type: mcp.ContentTypeText,
		Text: screenshotSummary(exports),
	})
	for _, export := range exports {
		content = append(content, mcp.ImageContent{
			Type:     mcp.ContentTypeImage,
			Data:     export.Base64,
			MIMEType: screenshotMIMEType(export.Format),
		})
	}

	return &mcp.CallToolResult{
		Content:           content,
		StructuredContent: resp.Data,
	}, nil
}

func renderMetadataResponse(resp BridgeResponse, err error) (*mcp.CallToolResult, error) {
	return renderStructuredTextResponse(resp, err, func(data map[string]interface{}) string {
		if metadataText, _ := data["metadataText"].(string); strings.TrimSpace(metadataText) != "" {
			return metadataText
		}
		return ""
	})
}

func renderVariableDefsResponse(resp BridgeResponse, err error) (*mcp.CallToolResult, error) {
	return renderStructuredTextResponse(resp, err, func(data map[string]interface{}) string {
		message, _ := data["message"].(string)
		scope, _ := data["scope"].(string)
		source, _ := data["source"].(string)
		if message != "" {
			if scope != "" && source != "" {
				return fmt.Sprintf("%s\n\nsource=%s\nscope=%s", message, source, scope)
			}
			return message
		}
		return ""
	})
}

func renderDesignContextResponse(resp BridgeResponse, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if resp.Error != "" {
		return mcp.NewToolResultError(resp.Error), nil
	}

	data, err := asStringMap(resp.Data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("parse design context response: %v", err)), nil
	}

	content := []mcp.Content{
		mcp.TextContent{
			Type: mcp.ContentTypeText,
			Text: buildDesignContextText(data),
		},
	}

	if screenshot := extractNestedMap(data, "screenshot"); len(screenshot) > 0 {
		if imageData, _ := screenshot["base64"].(string); imageData != "" {
			mimeType, _ := screenshot["mimeType"].(string)
			if mimeType == "" {
				mimeType = "image/png"
			}
			content = append(content, mcp.ImageContent{
				Type:     mcp.ContentTypeImage,
				Data:     imageData,
				MIMEType: mimeType,
			})
		}
	}

	return &mcp.CallToolResult{
		Content:           content,
		StructuredContent: resp.Data,
	}, nil
}

func renderFigJamResponse(resp BridgeResponse, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if resp.Error != "" {
		return mcp.NewToolResultError(resp.Error), nil
	}

	data, err := asStringMap(resp.Data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("parse figjam response: %v", err)), nil
	}

	content := []mcp.Content{
		mcp.TextContent{
			Type: mcp.ContentTypeText,
			Text: firstNonEmptyString(
				stringFromMap(data, "metadataText"),
				stringFromMap(data, "message"),
				mustJSON(data),
			),
		},
	}

	if rawImages, ok := data["images"].([]interface{}); ok {
		for _, rawImage := range rawImages {
			imageMap, ok := rawImage.(map[string]interface{})
			if !ok {
				continue
			}
			imageData, _ := imageMap["base64"].(string)
			if imageData == "" {
				continue
			}
			mimeType, _ := imageMap["mimeType"].(string)
			if mimeType == "" {
				mimeType = "image/png"
			}
			content = append(content, mcp.ImageContent{
				Type:     mcp.ContentTypeImage,
				Data:     imageData,
				MIMEType: mimeType,
			})
		}
	}

	return &mcp.CallToolResult{
		Content:           content,
		StructuredContent: resp.Data,
	}, nil
}

func renderStructuredTextResponse(resp BridgeResponse, err error, textBuilder func(map[string]interface{}) string) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if resp.Error != "" {
		return mcp.NewToolResultError(resp.Error), nil
	}

	data, err := asStringMap(resp.Data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("parse response: %v", err)), nil
	}

	text := textBuilder(data)
	if strings.TrimSpace(text) == "" {
		text = mustJSON(data)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: mcp.ContentTypeText,
				Text: text,
			},
		},
		StructuredContent: resp.Data,
	}, nil
}

func asStringMap(data interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func buildDesignContextText(data map[string]interface{}) string {
	code := stringFromMap(data, "code")
	metadata := extractNestedMap(data, "metadata")
	message := stringFromMap(metadata, "message")
	name := stringFromMap(data, "name")
	nodeID := stringFromMap(data, "nodeId")

	parts := make([]string, 0, 3)
	if name != "" || nodeID != "" {
		parts = append(parts, strings.TrimSpace(fmt.Sprintf("Design context for %s (%s)", name, nodeID)))
	}
	if message != "" {
		parts = append(parts, message)
	}
	if strings.TrimSpace(code) != "" {
		parts = append(parts, code)
	}
	return strings.Join(parts, "\n\n")
}

func extractNestedMap(data map[string]interface{}, key string) map[string]interface{} {
	raw, ok := data[key]
	if !ok {
		return nil
	}
	nested, ok := raw.(map[string]interface{})
	if ok {
		return nested
	}
	return nil
}

func stringFromMap(data map[string]interface{}, key string) string {
	if data == nil {
		return ""
	}
	value, _ := data[key].(string)
	return value
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mustJSON(data interface{}) string {
	b, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// toStringSlice converts []interface{} to []string.
func toStringSlice(raw []interface{}) []string {
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// ── save_screenshots ─────────────────────────────────────────────────────────

type saveItem struct {
	NodeID     string  `json:"nodeId"`
	OutputPath string  `json:"outputPath"`
	Format     string  `json:"format,omitempty"`
	Scale      float64 `json:"scale,omitempty"`
}

type saveResult struct {
	Index        int     `json:"index"`
	NodeID       string  `json:"nodeId"`
	NodeName     string  `json:"nodeName,omitempty"`
	OutputPath   string  `json:"outputPath"`
	Format       string  `json:"format,omitempty"`
	Width        float64 `json:"width,omitempty"`
	Height       float64 `json:"height,omitempty"`
	BytesWritten int     `json:"bytesWritten,omitempty"`
	Success      bool    `json:"success"`
	Error        string  `json:"error,omitempty"`
}

func executeSaveScreenshots(ctx context.Context, node *Node, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawItems, _ := req.GetArguments()["items"].([]interface{})
	defaultFormat, _ := req.GetArguments()["format"].(string)
	defaultScale, _ := req.GetArguments()["scale"].(float64)

	workDir, err := os.Getwd()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getwd: %v", err)), nil
	}

	results := make([]saveResult, 0, len(rawItems))
	succeeded, failed := 0, 0

	for i, rawItem := range rawItems {
		item, err := parseSaveItem(rawItem)
		if err != nil {
			results = append(results, saveResult{Index: i, Error: err.Error()})
			failed++
			continue
		}

		r := saveScreenshotItem(ctx, node, item, i, workDir, defaultFormat, defaultScale)
		results = append(results, r)
		if r.Success {
			succeeded++
		} else {
			failed++
		}
	}

	out, err := json.Marshal(map[string]interface{}{
		"total":     len(results),
		"succeeded": succeeded,
		"failed":    failed,
		"hasErrors": failed > 0,
		"results":   results,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal results: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

func saveScreenshotItem(ctx context.Context, node *Node, item saveItem, index int, workDir, defaultFormat string, defaultScale float64) saveResult {
	resolvedPath, err := resolveOutputPath(item.OutputPath, workDir)
	if err != nil {
		return saveResult{Index: index, NodeID: item.NodeID, OutputPath: item.OutputPath, Error: err.Error()}
	}

	format := coalesce(item.Format, defaultFormat)
	inferredFormat := inferFormat(resolvedPath)
	if format == "" {
		format = inferredFormat
	}
	if format == "" {
		format = "PNG"
	}
	if inferredFormat != "" && format != inferredFormat {
		return saveResult{Index: index, NodeID: item.NodeID, OutputPath: resolvedPath,
			Error: fmt.Sprintf("format %s conflicts with file extension %s", format, inferredFormat)}
	}

	scale := item.Scale
	if scale <= 0 {
		scale = defaultScale
	}

	params := map[string]interface{}{"format": format}
	if scale > 0 {
		params["scale"] = scale
	}

	resp, err := node.Send(ctx, "get_screenshot", []string{item.NodeID}, params)
	if err != nil {
		return saveResult{Index: index, NodeID: item.NodeID, OutputPath: resolvedPath, Error: err.Error()}
	}
	if resp.Error != "" {
		return saveResult{Index: index, NodeID: item.NodeID, OutputPath: resolvedPath, Error: resp.Error}
	}

	export, err := extractScreenshotExport(resp.Data)
	if err != nil {
		return saveResult{Index: index, NodeID: item.NodeID, OutputPath: resolvedPath, Error: err.Error()}
	}

	bytes, err := writeBase64(export.Base64, resolvedPath)
	if err != nil {
		return saveResult{Index: index, NodeID: item.NodeID, OutputPath: resolvedPath, Error: err.Error()}
	}

	return saveResult{
		Index:        index,
		NodeID:       export.NodeID,
		NodeName:     export.NodeName,
		OutputPath:   resolvedPath,
		Format:       format,
		Width:        export.Width,
		Height:       export.Height,
		BytesWritten: bytes,
		Success:      true,
	}
}

type screenshotExport struct {
	NodeID   string  `json:"nodeId"`
	NodeName string  `json:"nodeName"`
	Format   string  `json:"format"`
	Base64   string  `json:"base64"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
}

func extractScreenshotExport(data interface{}) (screenshotExport, error) {
	exports, err := extractScreenshotExports(data)
	if err != nil {
		return screenshotExport{}, err
	}
	return exports[0], nil
}

func extractScreenshotExports(data interface{}) ([]screenshotExport, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Exports []screenshotExport `json:"exports"`
	}
	if err := json.Unmarshal(b, &wrapper); err != nil {
		return nil, err
	}
	if len(wrapper.Exports) == 0 {
		return nil, errors.New("no screenshot export returned by plugin")
	}
	return wrapper.Exports, nil
}

func writeBase64(b64, outputPath string) (int, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return 0, fmt.Errorf("base64 decode: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return 0, fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return 0, fmt.Errorf("file already exists at outputPath: %s", outputPath)
		}
		return 0, err
	}
	defer f.Close()
	n, err := f.Write(data)
	return n, err
}

func resolveOutputPath(outputPath, workDir string) (string, error) {
	if filepath.IsAbs(outputPath) {
		return filepath.Clean(outputPath), nil
	}
	return mustBeInsideDir(filepath.Join(workDir, outputPath), workDir)
}

func mustBeInsideDir(resolved, workDir string) (string, error) {
	rel, err := filepath.Rel(workDir, resolved)
	if err != nil {
		return "", fmt.Errorf("outputPath must be inside the working directory: %s", workDir)
	}
	// Convert to forward slashes before prefix check so Windows paths like
	// "C:\.." don't bypass the ".." detection.
	if strings.HasPrefix(filepath.ToSlash(rel), "..") {
		return "", fmt.Errorf("outputPath must be inside the working directory: %s", workDir)
	}
	return resolved, nil
}

func inferFormat(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "PNG"
	case ".svg":
		return "SVG"
	case ".jpg", ".jpeg":
		return "JPG"
	case ".pdf":
		return "PDF"
	}
	return ""
}

func parseSaveItem(raw interface{}) (saveItem, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return saveItem{}, err
	}
	var item saveItem
	if err := json.Unmarshal(b, &item); err != nil {
		return saveItem{}, err
	}
	return item, nil
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func screenshotSummary(exports []screenshotExport) string {
	if len(exports) == 1 {
		return fmt.Sprintf("Exported screenshot for %s.", screenshotLabel(exports[0]))
	}
	return fmt.Sprintf("Exported %d screenshots.", len(exports))
}

func screenshotLabel(export screenshotExport) string {
	if export.NodeName != "" {
		return export.NodeName
	}
	if export.NodeID != "" {
		return export.NodeID
	}
	return "selection"
}

func screenshotMIMEType(format string) string {
	switch strings.ToUpper(format) {
	case "PNG", "":
		return "image/png"
	case "JPG", "JPEG":
		return "image/jpeg"
	case "SVG":
		return "image/svg+xml"
	case "PDF":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}
