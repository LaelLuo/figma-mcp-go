package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type restBridgeSender interface {
	Send(ctx context.Context, tool string, nodeIDs []string, params map[string]interface{}) (BridgeResponse, error)
}

// FigmaRestShim exposes a small official-Figma-REST-compatible surface over the
// local Figma plugin bridge. It is intentionally narrow: enough for early
// DesignCompose CLI experiments without pretending to be the full cloud API.
type FigmaRestShim struct {
	sender restBridgeSender
}

func NewFigmaRestShim(sender restBridgeSender) *FigmaRestShim {
	return &FigmaRestShim{sender: sender}
}

func (s *FigmaRestShim) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/", s.handle)
}

func (s *FigmaRestShim) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	segments := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/"), "/")
	if len(segments) < 2 || segments[0] != "files" || segments[1] == "" {
		http.Error(w, "unsupported local Figma REST path", http.StatusNotFound)
		return
	}

	fileKey := segments[1]
	switch {
	case len(segments) == 2:
		s.handleFile(w, r, fileKey)
	case len(segments) == 3 && segments[2] == "images":
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"meta": map[string]interface{}{
				"err":    nil,
				"images": map[string]interface{}{},
			},
		})
	case len(segments) == 3 && segments[2] == "nodes":
		s.handleNodes(w, r, fileKey)
	case len(segments) == 4 && segments[2] == "variables" && segments[3] == "local":
		s.handleVariables(w, r, fileKey)
	default:
		http.Error(w, "unsupported local Figma REST path", http.StatusNotFound)
	}
}

func (s *FigmaRestShim) handleFile(w http.ResponseWriter, r *http.Request, fileKey string) {
	if r.URL.Query().Get("depth") == "1" {
		s.handleFileDepthOne(w, r, fileKey)
		return
	}

	resp, err := s.sender.Send(r.Context(), "get_document", nil, map[string]interface{}{"fileKey": fileKey})
	if err != nil {
		s.sendJSON(w, http.StatusBadGateway, map[string]interface{}{"err": err.Error()})
		return
	}
	if resp.Error != "" {
		s.sendJSON(w, http.StatusBadGateway, map[string]interface{}{"err": resp.Error})
		return
	}

	document, ok := mapValue(resp.Data)
	if !ok {
		s.sendJSON(w, http.StatusBadGateway, map[string]interface{}{"err": "get_document returned non-object data"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"components":    map[string]interface{}{},
		"componentSets": map[string]interface{}{},
		"document":      buildOfficialDocument(document),
		"version":       "local",
		"lastModified":  now,
		"name":          fileKey,
		"branches":      nil,
	})
}

func (s *FigmaRestShim) handleFileDepthOne(w http.ResponseWriter, r *http.Request, fileKey string) {
	resp, err := s.sender.Send(r.Context(), "get_pages", nil, map[string]interface{}{"fileKey": fileKey})
	if err != nil {
		s.sendJSON(w, http.StatusBadGateway, map[string]interface{}{"err": err.Error()})
		return
	}
	if resp.Error != "" {
		s.sendJSON(w, http.StatusBadGateway, map[string]interface{}{"err": resp.Error})
		return
	}

	data, ok := mapValue(resp.Data)
	if !ok {
		s.sendJSON(w, http.StatusBadGateway, map[string]interface{}{"err": "get_pages returned non-object data"})
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"components":    map[string]interface{}{},
		"componentSets": map[string]interface{}{},
		"document":      buildOfficialDocumentFromPages(data),
		"version":       "local",
		"lastModified":  now,
		"name":          fileKey,
		"branches":      nil,
	})
}

func (s *FigmaRestShim) handleNodes(w http.ResponseWriter, r *http.Request, fileKey string) {
	ids := splitCommaList(r.URL.Query().Get("ids"))
	if len(ids) == 0 {
		s.sendJSON(w, http.StatusBadRequest, map[string]interface{}{"err": "ids query parameter is required"})
		return
	}

	resp, err := s.sender.Send(r.Context(), "get_nodes_info", ids, map[string]interface{}{"detail": "full", "fileKey": fileKey})
	if err != nil {
		s.sendJSON(w, http.StatusBadGateway, map[string]interface{}{"err": err.Error()})
		return
	}
	if resp.Error != "" {
		s.sendJSON(w, http.StatusBadGateway, map[string]interface{}{"err": resp.Error})
		return
	}

	nodes := map[string]interface{}{}
	rawNodes, _ := sliceValue(resp.Data)
	for _, rawNode := range rawNodes {
		node, ok := mapValue(rawNode)
		if !ok {
			continue
		}
		nodeID := stringValue(node["id"])
		if nodeID == "" {
			continue
		}
		nodes[nodeID] = map[string]interface{}{
			"document":      convertLocalNode(node),
			"components":    map[string]interface{}{},
			"schemaVersion": 0,
			"styles":        map[string]interface{}{},
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"name":         fileKey,
		"lastModified": now,
		"thumbnailUrl": "",
		"version":      "local",
		"role":         "owner",
		"nodes":        nodes,
	})
}

func (s *FigmaRestShim) handleVariables(w http.ResponseWriter, r *http.Request, fileKey string) {
	resp, err := s.sender.Send(r.Context(), "get_variable_defs", nil, map[string]interface{}{"fileKey": fileKey})
	if err != nil {
		s.sendJSON(w, http.StatusBadGateway, map[string]interface{}{"err": err.Error()})
		return
	}
	if resp.Error != "" {
		s.sendJSON(w, http.StatusBadGateway, map[string]interface{}{"err": resp.Error})
		return
	}

	data, _ := mapValue(resp.Data)
	s.sendJSON(w, http.StatusOK, convertVariablesResponse(data))
}

func (s *FigmaRestShim) sendJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		leaderLogger.Printf("encode local REST shim response error: %v", err)
	}
}

func buildOfficialDocument(localPage map[string]interface{}) map[string]interface{} {
	if stringValue(localPage["type"]) == "DOCUMENT" {
		return convertLocalNode(localPage)
	}
	return map[string]interface{}{
		"id":       "local-document",
		"name":     "Local Figma Document",
		"type":     "DOCUMENT",
		"children": []interface{}{convertLocalNode(localPage)},
	}
}

func buildOfficialDocumentFromPages(data map[string]interface{}) map[string]interface{} {
	rawPages, _ := sliceValue(data["pages"])
	pages := make([]interface{}, 0, len(rawPages))
	for _, rawPage := range rawPages {
		page, ok := mapValue(rawPage)
		if !ok {
			continue
		}
		pages = append(pages, map[string]interface{}{
			"id":       stringValueOr(page["id"], "local-page"),
			"name":     stringValueOr(page["name"], "Untitled"),
			"type":     "CANVAS",
			"children": []interface{}{},
		})
	}
	return map[string]interface{}{
		"id":       "local-document",
		"name":     "Local Figma Document",
		"type":     "DOCUMENT",
		"children": pages,
	}
}

func convertLocalNode(local map[string]interface{}) map[string]interface{} {
	return convertLocalNodeWithOffset(local, 0, 0)
}

func convertLocalNodeWithOffset(local map[string]interface{}, parentX, parentY float64) map[string]interface{} {
	nodeType := stringValue(local["type"])
	bounds, hasBounds := boundsValue(local["bounds"])
	childParentX := parentX
	childParentY := parentY
	if hasBounds {
		bounds["x"] = parentX + numberValue(bounds["x"], 0)
		bounds["y"] = parentY + numberValue(bounds["y"], 0)
		childParentX = numberValue(bounds["x"], parentX)
		childParentY = numberValue(bounds["y"], parentY)
	}
	node := map[string]interface{}{
		"id":       stringValueOr(local["id"], "local-node"),
		"name":     stringValueOr(local["name"], "Untitled"),
		"type":     nodeType,
		"visible":  true,
		"opacity":  1.0,
		"children": convertChildren(local["children"], childParentX, childParentY),
	}

	if hasBounds {
		node["absoluteBoundingBox"] = bounds
		node["absoluteRenderBounds"] = bounds
	}

	styles, _ := mapValue(local["styles"])
	fills := paintsFromStyles(styles)
	if len(fills) > 0 {
		node["fills"] = fills
	}

	switch nodeType {
	case "CANVAS":
		// No additional required fields.
	case "TEXT":
		node["constraints"] = defaultConstraints()
		node["characters"] = stringValue(local["characters"])
		node["style"] = defaultTextStyle(styles, fills)
		node["characterStyleOverrides"] = []interface{}{}
		node["styleOverrideTable"] = map[string]interface{}{}
	case "FRAME", "GROUP", "COMPONENT", "COMPONENT_SET", "INSTANCE":
		addDefaultFrameFields(node, styles)
		if nodeType == "INSTANCE" {
			node["componentId"] = stringValueOr(local["componentId"], "local-component")
		}
	case "RECTANGLE", "VECTOR", "LINE", "ELLIPSE", "STAR", "REGULAR_POLYGON", "BOOLEAN_OPERATION":
		node["constraints"] = defaultConstraints()
		if nodeType == "ELLIPSE" {
			node["arcData"] = map[string]interface{}{"startingAngle": 0.0, "endingAngle": 0.0, "innerRadius": 0.0}
		}
		if nodeType == "BOOLEAN_OPERATION" {
			node["booleanOperation"] = "UNION"
		}
	default:
		node["type"] = "FRAME"
		addDefaultFrameFields(node, styles)
	}

	return node
}

func convertChildren(value interface{}, parentX, parentY float64) []interface{} {
	rawChildren, ok := sliceValue(value)
	if !ok {
		return []interface{}{}
	}
	children := make([]interface{}, 0, len(rawChildren))
	for _, child := range rawChildren {
		if childMap, ok := mapValue(child); ok {
			children = append(children, convertLocalNodeWithOffset(childMap, parentX, parentY))
		}
	}
	return children
}

func addDefaultFrameFields(node map[string]interface{}, styles map[string]interface{}) {
	node["constraints"] = defaultConstraints()
	node["layoutMode"] = layoutModeFromStyles(styles)
	node["primaryAxisSizingMode"] = "AUTO"
	node["counterAxisSizingMode"] = "AUTO"
	node["primaryAxisAlignItems"] = "MIN"
	node["counterAxisAlignItems"] = "MIN"
	node["layoutWrap"] = "NO_WRAP"
}

func defaultConstraints() map[string]interface{} {
	return map[string]interface{}{
		"vertical":   "TOP",
		"horizontal": "LEFT",
	}
}

func layoutModeFromStyles(styles map[string]interface{}) string {
	if mode := strings.ToUpper(stringValue(styles["layoutMode"])); mode == "HORIZONTAL" || mode == "VERTICAL" {
		return mode
	}
	return "NONE"
}

func defaultTextStyle(styles map[string]interface{}, fills []interface{}) map[string]interface{} {
	fontSize := numberValue(styles["fontSize"], 16)
	lineHeight := fontSize
	if rawLineHeight, ok := mapValue(styles["lineHeight"]); ok {
		lineHeight = numberValue(rawLineHeight["value"], fontSize)
	}
	return map[string]interface{}{
		"fontFamily":         stringOrNil(styles["fontFamily"]),
		"fontPostScriptName": nil,
		"fontWeight":         numberValue(styles["fontWeight"], 400),
		"fontSize":           fontSize,
		"letterSpacing":      numberValue(styles["letterSpacing"], 0),
		"fills":              fills,
		"hyperlink":          nil,
		"lineHeightPx":       lineHeight,
		"lineHeightUnit":     "PIXELS",
	}
}

func paintsFromStyles(styles map[string]interface{}) []interface{} {
	rawFills := styles["fills"]
	values, ok := sliceValue(rawFills)
	if !ok {
		if fill, ok := rawFills.(string); ok {
			values = []interface{}{fill}
		}
	}

	paints := make([]interface{}, 0, len(values))
	for _, value := range values {
		hex := stringValue(value)
		color, ok := colorFromHex(hex)
		if !ok {
			continue
		}
		paints = append(paints, map[string]interface{}{
			"type":    "SOLID",
			"visible": true,
			"opacity": 1.0,
			"color":   color,
		})
	}
	return paints
}

func colorFromHex(hex string) (map[string]interface{}, bool) {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) != 6 && len(hex) != 8 {
		return nil, false
	}
	value, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return nil, false
	}

	r := float64((value >> uint(8*(len(hex)/2-1))) & 0xff)
	g := float64((value >> uint(8*(len(hex)/2-2))) & 0xff)
	b := float64((value >> uint(8*(len(hex)/2-3))) & 0xff)
	a := 255.0
	if len(hex) == 8 {
		a = float64(value & 0xff)
	}
	return map[string]interface{}{
		"r": r / 255.0,
		"g": g / 255.0,
		"b": b / 255.0,
		"a": a / 255.0,
	}, true
}

func convertVariablesResponse(data map[string]interface{}) map[string]interface{} {
	collections := map[string]interface{}{}
	variables := map[string]interface{}{}

	rawCollections, _ := sliceValue(data["collections"])
	for _, rawCollection := range rawCollections {
		collection, ok := mapValue(rawCollection)
		if !ok {
			continue
		}
		collectionID := stringValue(collection["id"])
		if collectionID == "" {
			continue
		}

		modes := convertVariableModes(collection["modes"])
		defaultModeID := ""
		if len(modes) > 0 {
			if first, ok := modes[0].(map[string]interface{}); ok {
				defaultModeID = stringValue(first["modeId"])
			}
		}
		collections[collectionID] = map[string]interface{}{
			"defaultModeId": defaultModeID,
			"id":            collectionID,
			"name":          stringValueOr(collection["name"], collectionID),
			"remote":        false,
			"modes":         modes,
			"key":           collectionID,
		}

		rawVariables, _ := sliceValue(collection["variables"])
		for _, rawVariable := range rawVariables {
			variable, ok := mapValue(rawVariable)
			if !ok {
				continue
			}
			variableID := stringValue(variable["id"])
			if variableID == "" {
				continue
			}
			variables[variableID] = map[string]interface{}{
				"id":                   variableID,
				"name":                 stringValueOr(variable["name"], variableID),
				"remote":               false,
				"key":                  variableID,
				"variableCollectionId": collectionID,
				"resolvedType":         stringValueOr(variable["resolvedType"], "STRING"),
				"valuesByMode":         normalizeVariableValues(variable["valuesByMode"]),
			}
		}
	}

	return map[string]interface{}{
		"error":  false,
		"status": 200,
		"meta": map[string]interface{}{
			"variableCollections": collections,
			"variables":           variables,
		},
	}
}

func convertVariableModes(value interface{}) []interface{} {
	rawModes, ok := sliceValue(value)
	if !ok {
		return []interface{}{}
	}
	modes := make([]interface{}, 0, len(rawModes))
	for _, rawMode := range rawModes {
		mode, ok := mapValue(rawMode)
		if !ok {
			continue
		}
		modes = append(modes, map[string]interface{}{
			"modeId": stringValue(mode["modeId"]),
			"name":   stringValue(mode["name"]),
		})
	}
	return modes
}

func normalizeVariableValues(value interface{}) map[string]interface{} {
	values, ok := mapValue(value)
	if !ok {
		return map[string]interface{}{}
	}
	normalized := make(map[string]interface{}, len(values))
	for modeID, rawValue := range values {
		normalized[modeID] = normalizeVariableValue(rawValue)
	}
	return normalized
}

func normalizeVariableValue(value interface{}) interface{} {
	if color, ok := mapValue(value); ok {
		if _, hasR := color["r"]; hasR {
			return map[string]interface{}{
				"r": numberValue(color["r"], 0),
				"g": numberValue(color["g"], 0),
				"b": numberValue(color["b"], 0),
				"a": numberValue(color["a"], 1),
			}
		}
	}
	return value
}

func boundsValue(value interface{}) (map[string]interface{}, bool) {
	bounds, ok := mapValue(value)
	if !ok {
		return nil, false
	}
	return map[string]interface{}{
		"x":      numberValue(bounds["x"], 0),
		"y":      numberValue(bounds["y"], 0),
		"width":  numberValue(bounds["width"], 0),
		"height": numberValue(bounds["height"], 0),
	}, true
}

func mapValue(value interface{}) (map[string]interface{}, bool) {
	if typed, ok := value.(map[string]interface{}); ok {
		return typed, true
	}
	return nil, false
}

func sliceValue(value interface{}) ([]interface{}, bool) {
	if typed, ok := value.([]interface{}); ok {
		return typed, true
	}
	return nil, false
}

func stringValue(value interface{}) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func stringValueOr(value interface{}, fallback string) string {
	if typed := stringValue(value); typed != "" {
		return typed
	}
	return fallback
}

func stringOrNil(value interface{}) interface{} {
	if typed := stringValue(value); typed != "" {
		return typed
	}
	return nil
}

func numberValue(value interface{}, fallback float64) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		if parsed, err := typed.Float64(); err == nil {
			return parsed
		}
	case string:
		if parsed, err := strconv.ParseFloat(typed, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func splitCommaList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
