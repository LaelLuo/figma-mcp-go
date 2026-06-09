package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeRESTSender struct {
	calls []restCall
	data  map[string]interface{}
}

type restCall struct {
	tool    string
	nodeIDs []string
	params  map[string]interface{}
}

func (f *fakeRESTSender) Send(_ context.Context, tool string, nodeIDs []string, params map[string]interface{}) (BridgeResponse, error) {
	f.calls = append(f.calls, restCall{tool: tool, nodeIDs: nodeIDs, params: params})
	return BridgeResponse{Type: tool, Data: f.data[tool]}, nil
}

func TestFigmaRestShimFileReturnsOfficialFileEnvelope(t *testing.T) {
	sender := &fakeRESTSender{
		data: map[string]interface{}{
			"get_document": map[string]interface{}{
				"id":   "1:3",
				"name": "Page 2",
				"type": "CANVAS",
				"children": []interface{}{
					map[string]interface{}{
						"id":   "2:4",
						"name": "Hero",
						"type": "FRAME",
						"bounds": map[string]interface{}{
							"x": 10.0, "y": 20.0, "width": 320.0, "height": 180.0,
						},
						"styles": map[string]interface{}{
							"fills": []interface{}{"#336699"},
						},
					},
				},
			},
		},
	}
	mux := http.NewServeMux()
	NewFigmaRestShim(sender).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/files/local-file?depth=2", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if len(sender.calls) != 1 || sender.calls[0].tool != "get_document" {
		t.Fatalf("calls = %#v, want one get_document call", sender.calls)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["name"] != "local-file" {
		t.Fatalf("name = %#v, want local-file", body["name"])
	}
	document := body["document"].(map[string]interface{})
	if document["type"] != "DOCUMENT" {
		t.Fatalf("document.type = %#v, want DOCUMENT", document["type"])
	}
	pages := document["children"].([]interface{})
	page := pages[0].(map[string]interface{})
	if page["type"] != "CANVAS" || page["name"] != "Page 2" {
		t.Fatalf("page = %#v, want current page canvas", page)
	}
	frame := page["children"].([]interface{})[0].(map[string]interface{})
	if frame["absoluteBoundingBox"] == nil {
		t.Fatalf("frame lacks absoluteBoundingBox: %#v", frame)
	}
	fills := frame["fills"].([]interface{})
	paint := fills[0].(map[string]interface{})
	if paint["type"] != "SOLID" {
		t.Fatalf("paint.type = %#v, want SOLID", paint["type"])
	}
}

func TestFigmaRestShimDepthOneUsesPagesSummary(t *testing.T) {
	sender := &fakeRESTSender{
		data: map[string]interface{}{
			"get_pages": map[string]interface{}{
				"currentPageId": "1:3",
				"pages": []interface{}{
					map[string]interface{}{"id": "0:1", "name": "移动端"},
					map[string]interface{}{"id": "1:3", "name": "Page 2"},
				},
			},
		},
	}
	mux := http.NewServeMux()
	NewFigmaRestShim(sender).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/files/local-file?depth=1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if len(sender.calls) != 1 || sender.calls[0].tool != "get_pages" {
		t.Fatalf("calls = %#v, want one get_pages call", sender.calls)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	document := body["document"].(map[string]interface{})
	pages := document["children"].([]interface{})
	if len(pages) != 2 {
		t.Fatalf("page count = %d, want 2", len(pages))
	}
	page := pages[1].(map[string]interface{})
	if page["id"] != "1:3" || page["type"] != "CANVAS" {
		t.Fatalf("page = %#v, want official canvas summary", page)
	}
}

func TestFigmaRestShimNodesReturnsOfficialNodesEnvelope(t *testing.T) {
	sender := &fakeRESTSender{
		data: map[string]interface{}{
			"get_nodes_info": []interface{}{
				map[string]interface{}{
					"id":   "2:4",
					"name": "Hero",
					"type": "FRAME",
					"bounds": map[string]interface{}{
						"x": 10.0, "y": 20.0, "width": 320.0, "height": 180.0,
					},
					"children": []interface{}{},
				},
			},
		},
	}
	mux := http.NewServeMux()
	NewFigmaRestShim(sender).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/files/local-file/nodes?ids=2:4&geometry=paths", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if len(sender.calls) != 1 || sender.calls[0].tool != "get_nodes_info" {
		t.Fatalf("calls = %#v, want one get_nodes_info call", sender.calls)
	}
	if got := strings.Join(sender.calls[0].nodeIDs, ","); got != "2:4" {
		t.Fatalf("nodeIDs = %q, want 2:4", got)
	}
	if sender.calls[0].params["detail"] != "full" {
		t.Fatalf("params = %#v, want detail=full", sender.calls[0].params)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	nodes := body["nodes"].(map[string]interface{})
	nodeWrapper := nodes["2:4"].(map[string]interface{})
	document := nodeWrapper["document"].(map[string]interface{})
	if document["name"] != "Hero" || document["type"] != "FRAME" {
		t.Fatalf("document = %#v, want converted Hero frame", document)
	}
	if nodeWrapper["schemaVersion"] != float64(0) {
		t.Fatalf("schemaVersion = %#v, want 0", nodeWrapper["schemaVersion"])
	}
}

func TestFigmaRestShimImagesReturnsEmptyOfficialEnvelope(t *testing.T) {
	sender := &fakeRESTSender{data: map[string]interface{}{}}
	mux := http.NewServeMux()
	NewFigmaRestShim(sender).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/files/local-file/images", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if len(sender.calls) != 0 {
		t.Fatalf("image shim should not call plugin yet, got %#v", sender.calls)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	meta := body["meta"].(map[string]interface{})
	images := meta["images"].(map[string]interface{})
	if len(images) != 0 {
		t.Fatalf("images = %#v, want empty map", images)
	}
}

func TestFigmaRestShimVariablesConvertsLocalCollections(t *testing.T) {
	sender := &fakeRESTSender{
		data: map[string]interface{}{
			"get_variable_defs": map[string]interface{}{
				"collections": []interface{}{
					map[string]interface{}{
						"id":   "VariableCollectionId:1:2",
						"name": "Tokens",
						"modes": []interface{}{
							map[string]interface{}{"modeId": "1:0", "name": "Mode 1"},
						},
						"variables": []interface{}{
							map[string]interface{}{
								"id":           "VariableID:1:3",
								"name":         "Color/Brand",
								"resolvedType": "COLOR",
								"valuesByMode": map[string]interface{}{
									"1:0": map[string]interface{}{"r": 0.2, "g": 0.4, "b": 0.6, "a": 1.0},
								},
							},
						},
					},
				},
			},
		},
	}
	mux := http.NewServeMux()
	NewFigmaRestShim(sender).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/files/local-file/variables/local", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if len(sender.calls) != 1 || sender.calls[0].tool != "get_variable_defs" {
		t.Fatalf("calls = %#v, want one get_variable_defs call", sender.calls)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != false || body["status"] != float64(200) {
		t.Fatalf("body status = %#v", body)
	}
	meta := body["meta"].(map[string]interface{})
	collections := meta["variableCollections"].(map[string]interface{})
	if _, ok := collections["VariableCollectionId:1:2"]; !ok {
		t.Fatalf("variableCollections missing collection: %#v", collections)
	}
	variables := meta["variables"].(map[string]interface{})
	variable := variables["VariableID:1:3"].(map[string]interface{})
	if variable["variableCollectionId"] != "VariableCollectionId:1:2" {
		t.Fatalf("variableCollectionId = %#v", variable["variableCollectionId"])
	}
}

func TestFigmaRestShimRejectsUnsupportedPaths(t *testing.T) {
	sender := &fakeRESTSender{data: map[string]interface{}{}}
	mux := http.NewServeMux()
	NewFigmaRestShim(sender).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/123/files", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unsupported") {
		t.Fatalf("body = %q, want unsupported message", w.Body.String())
	}
}
