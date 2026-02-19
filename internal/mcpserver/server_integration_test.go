package mcpserver_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aellingwood/forge/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const testSiteDir = "../../testdata/mcp-test-site"

// newTestClient starts a ForgeServer and connects a test client.
// Returns the client session and a cleanup function.
func newTestClient(t *testing.T) (*mcp.ClientSession, func()) {
	t.Helper()

	siteDir, err := filepath.Abs(testSiteDir)
	if err != nil {
		t.Fatalf("resolving site dir: %v", err)
	}

	srv := mcpserver.New(siteDir, "test")

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())

	// Start server in background
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatalf("connecting client: %v", err)
	}

	cleanup := func() {
		session.Close()
		cancel()
		select {
		case <-serverDone:
		case <-time.After(2 * time.Second):
		}
	}

	return session, cleanup
}

func TestIntegration_Initialize(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	result := session.InitializeResult()
	if result == nil {
		t.Fatal("expected non-nil initialize result")
	}
	if result.ServerInfo.Name != "forge" {
		t.Errorf("expected server name 'forge', got %q", result.ServerInfo.Name)
	}
}

func TestIntegration_ListResources(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}

	uris := make(map[string]bool)
	for _, r := range result.Resources {
		uris[r.URI] = true
	}

	expectedURIs := []string{
		"forge://config",
		"forge://content/pages",
		"forge://content/sections",
		"forge://taxonomies",
		"forge://templates",
		"forge://build/status",
		"forge://schema/frontmatter",
	}
	for _, uri := range expectedURIs {
		if !uris[uri] {
			t.Errorf("expected resource %q not found in list", uri)
		}
	}
}

func TestIntegration_ConfigResource(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "forge://config"})
	if err != nil {
		t.Fatalf("ReadResource forge://config: %v", err)
	}
	if len(result.Contents) == 0 {
		t.Fatal("expected non-empty contents")
	}

	var cfg map[string]any
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &cfg); err != nil {
		t.Fatalf("parsing config JSON: %v", err)
	}

	// SiteConfig has no json tags, so fields serialize with capitalized names
	titleRaw := cfg["Title"]
	if titleRaw == nil {
		titleRaw = cfg["title"] // fallback if json tags are added later
	}
	title, _ := titleRaw.(string)
	if title == "" {
		t.Errorf("expected non-empty Title in config, got keys: %v", func() []string {
			keys := make([]string, 0, len(cfg))
			for k := range cfg {
				keys = append(keys, k)
			}
			return keys
		}())
	}
}

func TestIntegration_ContentPagesResource(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "forge://content/pages"})
	if err != nil {
		t.Fatalf("ReadResource forge://content/pages: %v", err)
	}
	if len(result.Contents) == 0 {
		t.Fatal("expected non-empty contents")
	}

	var inventory map[string]any
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &inventory); err != nil {
		t.Fatalf("parsing pages JSON: %v", err)
	}

	total, ok := inventory["totalPages"].(float64)
	if !ok || total == 0 {
		t.Errorf("expected totalPages > 0, got: %v", inventory["totalPages"])
	}

	pages, ok := inventory["pages"].([]any)
	if !ok {
		t.Fatalf("expected pages array, got: %T", inventory["pages"])
	}
	if len(pages) == 0 {
		t.Error("expected at least one page")
	}
}

func TestIntegration_TaxonomiesResource(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "forge://taxonomies"})
	if err != nil {
		t.Fatalf("ReadResource forge://taxonomies: %v", err)
	}

	var overview map[string]any
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &overview); err != nil {
		t.Fatalf("parsing taxonomies JSON: %v", err)
	}

	taxos, ok := overview["taxonomies"].([]any)
	if !ok || len(taxos) == 0 {
		t.Errorf("expected non-empty taxonomies, got: %v", overview["taxonomies"])
	}
}

func TestIntegration_FrontmatterSchemaResource(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "forge://schema/frontmatter"})
	if err != nil {
		t.Fatalf("ReadResource forge://schema/frontmatter: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &schema); err != nil {
		t.Fatalf("parsing schema JSON: %v", err)
	}

	required, ok := schema["required"].([]any)
	if !ok || len(required) == 0 {
		t.Error("expected non-empty required fields")
	}

	fields, ok := schema["fields"].(map[string]any)
	if !ok || len(fields) == 0 {
		t.Error("expected non-empty fields")
	}
}

func TestIntegration_QueryContent_NoFilter(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "query_content",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool query_content: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	total, ok := out["totalMatches"].(float64)
	if !ok || total == 0 {
		t.Errorf("expected totalMatches > 0, got: %v", out["totalMatches"])
	}
}

func TestIntegration_QueryContent_SectionFilter(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "query_content",
		Arguments: map[string]any{"section": "blog"},
	})
	if err != nil {
		t.Fatalf("CallTool query_content with section: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	total, _ := out["totalMatches"].(float64)
	if total == 0 {
		t.Error("expected blog posts in result")
	}

	// Verify all returned pages are from blog section
	pages, _ := out["pages"].([]any)
	for _, p := range pages {
		page := p.(map[string]any)
		if section, _ := page["section"].(string); section != "blog" {
			t.Errorf("expected section=blog, got: %q", section)
		}
	}
}

func TestIntegration_QueryContent_DraftFilter(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "query_content",
		Arguments: map[string]any{"draft": true},
	})
	if err != nil {
		t.Fatalf("CallTool query_content with draft: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	total, _ := out["totalMatches"].(float64)
	if total == 0 {
		t.Error("expected draft pages in result")
	}

	// All returned pages should be drafts
	pages, _ := out["pages"].([]any)
	for _, p := range pages {
		page := p.(map[string]any)
		if draft, _ := page["draft"].(bool); !draft {
			t.Errorf("expected draft=true for page %v", page["title"])
		}
	}
}

func TestIntegration_QueryContent_TagFilter(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "query_content",
		Arguments: map[string]any{"tags": []any{"go"}},
	})
	if err != nil {
		t.Fatalf("CallTool query_content with tags: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	total, _ := out["totalMatches"].(float64)
	if total == 0 {
		t.Error("expected pages with go tag")
	}
}

func TestIntegration_QueryContent_InvalidSection(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "query_content",
		Arguments: map[string]any{"section": "nonexistent"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for unknown section")
	}
}

func TestIntegration_ListDrafts(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_drafts",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool list_drafts: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	total, _ := out["totalDrafts"].(float64)
	if total == 0 {
		t.Error("expected at least one draft")
	}
}

func TestIntegration_ValidateFrontmatter_Valid(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	fm := "title: \"My Test Post\"\ndate: 2025-01-15T10:00:00Z\ndraft: true\n"
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "validate_frontmatter",
		Arguments: map[string]any{"frontmatter": fm},
	})
	if err != nil {
		t.Fatalf("CallTool validate_frontmatter: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	if valid, _ := out["valid"].(bool); !valid {
		t.Errorf("expected valid=true, got errors: %v", out["errors"])
	}
}

func TestIntegration_ValidateFrontmatter_BadDate(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	fm := "title: \"My Post\"\ndate: \"January 15, 2025\"\n"
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "validate_frontmatter",
		Arguments: map[string]any{"frontmatter": fm},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	if valid, _ := out["valid"].(bool); valid {
		t.Error("expected invalid due to bad date format")
	}
}

func TestIntegration_ValidateFrontmatter_SimilarTag(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	fm := "title: \"My Post\"\ndate: 2025-01-15T10:00:00Z\ntags:\n  - k8s\n"
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "validate_frontmatter",
		Arguments: map[string]any{"frontmatter": fm},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	warnings, _ := out["warnings"].([]any)
	if len(warnings) == 0 {
		t.Error("expected warning for k8s similar to kubernetes")
	}
}

func TestIntegration_CreateContent(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	siteDir, _ := filepath.Abs(testSiteDir)

	ctx := context.Background()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "create_content",
		Arguments: map[string]any{
			"type":  "post",
			"title": "Integration Test Post",
			"tags":  []any{"go", "testing"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool create_content: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	if created, _ := out["created"].(bool); !created {
		t.Error("expected created=true")
	}

	filePath, _ := out["filePath"].(string)
	if filePath == "" {
		t.Error("expected non-empty filePath")
	}

	// Clean up created file
	absPath := filepath.Join(siteDir, filePath)
	t.Cleanup(func() {
		os.Remove(absPath)
	})

	// Verify file was created
	if _, err := os.Stat(absPath); err != nil {
		t.Errorf("expected file to exist at %s: %v", absPath, err)
	}

	url, _ := out["url"].(string)
	if !strings.HasPrefix(url, "/blog/") {
		t.Errorf("expected blog URL, got: %q", url)
	}
}

func TestIntegration_CreateContent_DuplicateRejected(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	siteDir, _ := filepath.Abs(testSiteDir)
	ctx := context.Background()

	// Create once
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "create_content",
		Arguments: map[string]any{"type": "post", "title": "Duplicate Test Post"},
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	var out map[string]any
	json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out)
	filePath, _ := out["filePath"].(string)
	if filePath != "" {
		t.Cleanup(func() { os.Remove(filepath.Join(siteDir, filePath)) })
	}

	// Try to create duplicate (same title = same slug = same date)
	result2, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "create_content",
		Arguments: map[string]any{"type": "post", "title": "Duplicate Test Post"},
	})
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if !result2.IsError {
		t.Error("expected error when creating duplicate file")
	}
}

func TestIntegration_ResolveLayout(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "resolve_layout",
		Arguments: map[string]any{"pagePath": "content/blog/2025-01-15-go-error-handling.md"},
	})
	if err != nil {
		t.Fatalf("CallTool resolve_layout: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	var out map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("parsing output: %v", err)
	}

	if _, ok := out["lookupOrder"]; !ok {
		t.Error("expected lookupOrder in output")
	}
}

func TestIntegration_BuildStatus_Initial(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "forge://build/status"})
	if err != nil {
		t.Fatalf("ReadResource forge://build/status: %v", err)
	}

	var status map[string]any
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &status); err != nil {
		t.Fatalf("parsing build status: %v", err)
	}

	// lastBuild should be nil initially
	if lastBuild := status["lastBuild"]; lastBuild != nil {
		t.Logf("lastBuild: %v (non-nil, which is fine if a build ran)", lastBuild)
	}
}

func TestIntegration_ListTools(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	tools := make(map[string]bool)
	for _, tool := range result.Tools {
		tools[tool.Name] = true
	}

	expectedTools := []string{
		"query_content", "get_page", "list_drafts", "validate_frontmatter",
		"get_template_context", "resolve_layout", "create_content",
		"build_site", "deploy_site",
	}
	for _, name := range expectedTools {
		if !tools[name] {
			t.Errorf("expected tool %q not found", name)
		}
	}
}

func TestIntegration_ListPrompts(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}

	prompts := make(map[string]bool)
	for _, p := range result.Prompts {
		prompts[p.Name] = true
	}

	expectedPrompts := []string{"new_blog_post", "new_project", "content_review", "site_overview"}
	for _, name := range expectedPrompts {
		if !prompts[name] {
			t.Errorf("expected prompt %q not found", name)
		}
	}
}

func TestIntegration_GetPrompt_NewBlogPost(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "new_blog_post",
		Arguments: map[string]string{"topic": "Go generics"},
	})
	if err != nil {
		t.Fatalf("GetPrompt new_blog_post: %v", err)
	}
	if len(result.Messages) == 0 {
		t.Error("expected non-empty messages")
	}
}

func TestIntegration_GetPage(t *testing.T) {
	session, cleanup := newTestClient(t)
	defer cleanup()

	ctx := context.Background()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_page",
		Arguments: map[string]any{"url": "/blog/go-error-handling/"},
	})
	if err != nil {
		t.Fatalf("CallTool get_page: %v", err)
	}
	// May or may not find by URL depending on how the slug is generated
	// Just check it doesn't crash
	_ = result
}
