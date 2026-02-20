// Package security provides Content Security Policy (CSP) generation,
// nonce-based script authorization, and related security header helpers.
package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aellingwood/forge/internal/config"
)

// GenerateNonce produces a 16-byte cryptographically random nonce,
// returned as a base64-encoded string.
func GenerateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// CSPPolicy holds the directives for a Content-Security-Policy header.
type CSPPolicy struct {
	DefaultSrc  []string
	ScriptSrc   []string
	StyleSrc    []string
	ImgSrc      []string
	FontSrc     []string
	ConnectSrc  []string
	ManifestSrc []string
	BaseURI     []string
	FormAction  []string
	FrameAnc    []string
}

// String serializes the policy to a CSP header value.
func (p *CSPPolicy) String() string {
	// Build directive strings, skipping empty directives.
	var directives []string
	add := func(name string, values []string) {
		if len(values) > 0 {
			directives = append(directives, name+" "+strings.Join(values, " "))
		}
	}
	add("default-src", p.DefaultSrc)
	add("script-src", p.ScriptSrc)
	add("style-src", p.StyleSrc)
	add("img-src", p.ImgSrc)
	add("font-src", p.FontSrc)
	add("connect-src", p.ConnectSrc)
	add("manifest-src", p.ManifestSrc)
	add("base-uri", p.BaseURI)
	add("form-action", p.FormAction)
	add("frame-ancestors", p.FrameAnc)
	return strings.Join(directives, "; ")
}

// DevPolicy returns a CSP suitable for the development server. The nonce
// secures inline scripts (live reload). WebSocket connect-src is allowed for
// the given port.
func DevPolicy(nonce string, port int) *CSPPolicy {
	return &CSPPolicy{
		DefaultSrc:  []string{"'none'"},
		ScriptSrc:   []string{"'self'", fmt.Sprintf("'nonce-%s'", nonce)},
		StyleSrc:    []string{"'self'", "'unsafe-inline'"},
		ImgSrc:      []string{"'self'", "data:"},
		FontSrc:     []string{"'self'"},
		ConnectSrc:  []string{"'self'", fmt.Sprintf("ws://localhost:%d", port)},
		ManifestSrc: []string{"'self'"},
		BaseURI:     []string{"'self'"},
		FormAction:  []string{"'self'"},
		FrameAnc:    []string{"'none'"},
	}
}

// ProdPolicy returns a CSP suitable for production. No nonces -- inline
// scripts should be externalized. Extra directives from config are appended.
func ProdPolicy(extra *config.CSPConfig) *CSPPolicy {
	p := &CSPPolicy{
		DefaultSrc:  []string{"'none'"},
		ScriptSrc:   []string{"'self'"},
		StyleSrc:    []string{"'self'", "'unsafe-inline'"},
		ImgSrc:      []string{"'self'", "data:"},
		FontSrc:     []string{"'self'"},
		ConnectSrc:  []string{"'self'"},
		ManifestSrc: []string{"'self'"},
		BaseURI:     []string{"'self'"},
		FormAction:  []string{"'self'"},
		FrameAnc:    []string{"'none'"},
	}
	if extra != nil {
		p.ScriptSrc = append(p.ScriptSrc, extra.ScriptSrc...)
		p.StyleSrc = append(p.StyleSrc, extra.StyleSrc...)
		p.ImgSrc = append(p.ImgSrc, extra.ImgSrc...)
		p.FontSrc = append(p.FontSrc, extra.FontSrc...)
		p.ConnectSrc = append(p.ConnectSrc, extra.ConnectSrc...)
	}
	return p
}
