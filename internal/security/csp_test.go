package security

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/aellingwood/forge/internal/config"
)

func TestGenerateNonce_Length(t *testing.T) {
	nonce, err := GenerateNonce()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 16 bytes base64-encoded = 24 characters.
	decoded, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		t.Fatalf("nonce is not valid base64: %v", err)
	}
	if len(decoded) != 16 {
		t.Errorf("expected 16 decoded bytes, got %d", len(decoded))
	}
}

func TestGenerateNonce_Unique(t *testing.T) {
	n1, err := GenerateNonce()
	if err != nil {
		t.Fatal(err)
	}
	n2, err := GenerateNonce()
	if err != nil {
		t.Fatal(err)
	}
	if n1 == n2 {
		t.Error("two consecutive nonces should not be equal")
	}
}

func TestCSPPolicy_String(t *testing.T) {
	p := &CSPPolicy{
		DefaultSrc: []string{"'none'"},
		ScriptSrc:  []string{"'self'"},
		ImgSrc:     []string{"'self'", "data:"},
	}
	s := p.String()
	if !strings.Contains(s, "default-src 'none'") {
		t.Error("expected default-src 'none' in policy")
	}
	if !strings.Contains(s, "script-src 'self'") {
		t.Error("expected script-src 'self' in policy")
	}
	if !strings.Contains(s, "img-src 'self' data:") {
		t.Error("expected img-src 'self' data: in policy")
	}
	// Empty directives should be skipped.
	if strings.Contains(s, "style-src") {
		t.Error("empty style-src should be omitted")
	}
	if strings.Contains(s, "font-src") {
		t.Error("empty font-src should be omitted")
	}
}

func TestCSPPolicy_String_Empty(t *testing.T) {
	p := &CSPPolicy{}
	if p.String() != "" {
		t.Errorf("expected empty string for empty policy, got %q", p.String())
	}
}

func TestDevPolicy(t *testing.T) {
	nonce := "testNonce123"
	p := DevPolicy(nonce, 1313)
	s := p.String()

	if !strings.Contains(s, "'nonce-testNonce123'") {
		t.Error("expected nonce in dev policy script-src")
	}
	if !strings.Contains(s, "ws://localhost:1313") {
		t.Error("expected WebSocket connect-src for port 1313")
	}
	if !strings.Contains(s, "default-src 'none'") {
		t.Error("expected restrictive default-src")
	}
	if !strings.Contains(s, "frame-ancestors 'none'") {
		t.Error("expected frame-ancestors 'none'")
	}
}

func TestDevPolicy_DifferentPort(t *testing.T) {
	p := DevPolicy("abc", 8080)
	s := p.String()
	if !strings.Contains(s, "ws://localhost:8080") {
		t.Error("expected WebSocket connect-src for port 8080")
	}
}

func TestProdPolicy_NoNonce(t *testing.T) {
	p := ProdPolicy(nil)
	s := p.String()

	if strings.Contains(s, "nonce") {
		t.Error("production policy should not contain nonces")
	}
	if !strings.Contains(s, "script-src 'self'") {
		t.Error("expected script-src 'self' in prod policy")
	}
	if strings.Contains(s, "ws://") {
		t.Error("production policy should not contain WebSocket connect-src")
	}
}

func TestProdPolicy_WithExtras(t *testing.T) {
	extra := &config.CSPConfig{
		ScriptSrc:  []string{"https://cdn.example.com"},
		StyleSrc:   []string{"https://fonts.googleapis.com"},
		ImgSrc:     []string{"https://images.example.com"},
		FontSrc:    []string{"https://fonts.gstatic.com"},
		ConnectSrc: []string{"https://api.example.com"},
	}
	p := ProdPolicy(extra)
	s := p.String()

	if !strings.Contains(s, "https://cdn.example.com") {
		t.Error("expected extra script-src in prod policy")
	}
	if !strings.Contains(s, "https://fonts.googleapis.com") {
		t.Error("expected extra style-src in prod policy")
	}
	if !strings.Contains(s, "https://images.example.com") {
		t.Error("expected extra img-src in prod policy")
	}
	if !strings.Contains(s, "https://fonts.gstatic.com") {
		t.Error("expected extra font-src in prod policy")
	}
	if !strings.Contains(s, "https://api.example.com") {
		t.Error("expected extra connect-src in prod policy")
	}
}

func TestProdPolicy_NilExtra(t *testing.T) {
	// Should not panic with nil extras.
	p := ProdPolicy(nil)
	if p == nil {
		t.Fatal("expected non-nil policy")
	}
	s := p.String()
	if s == "" {
		t.Error("expected non-empty policy string")
	}
}

func TestCSPPolicy_DirectiveOrder(t *testing.T) {
	p := DevPolicy("test", 1313)
	s := p.String()

	// Verify directives are separated by "; ".
	parts := strings.Split(s, "; ")
	if len(parts) < 5 {
		t.Errorf("expected at least 5 directive parts, got %d", len(parts))
	}

	// default-src should come first.
	if !strings.HasPrefix(parts[0], "default-src") {
		t.Errorf("expected default-src as first directive, got %q", parts[0])
	}
}
