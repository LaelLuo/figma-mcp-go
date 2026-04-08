package prompts

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestRegisterAll_NoPanic(t *testing.T) {
	s := server.NewMCPServer("test", "0.0.1")
	RegisterAll(s) // must register all prompts without panicking
}

func TestPromptText_ReflectsOfficialFallbackModel(t *testing.T) {
	s := server.NewMCPServer("test", "0.0.1")
	RegisterAll(s)

	c, err := client.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}

	ctx := context.Background()
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "prompt-test",
		Version: "1.0.0",
	}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}

	if _, err := c.Initialize(ctx, initReq); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	names := []string{
		"read_design_strategy",
		"design_strategy",
		"design_token_generation_strategy",
		"style_audit_strategy",
		"reaction_to_connector_strategy",
	}

	var combined strings.Builder
	for _, name := range names {
		req := mcp.GetPromptRequest{}
		req.Params.Name = name

		result, err := c.GetPrompt(ctx, req)
		if err != nil {
			t.Fatalf("GetPrompt(%s): %v", name, err)
		}
		for _, message := range result.Messages {
			text, ok := message.Content.(mcp.TextContent)
			if !ok {
				continue
			}
			combined.WriteString(text.Text)
			combined.WriteString("\n")
		}
	}

	text := combined.String()
	for _, needle := range []string{
		"official Figma MCP",
		"fallback",
		"get_figjam",
		"get_metadata",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("combined prompt text missing %q", needle)
		}
	}
}
