// Package config handles loading, validating, and managing site configuration
// for the Forge static site generator.
package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// SiteConfig is the top-level configuration for a Forge site.
type SiteConfig struct {
	BaseURL     string            `yaml:"baseURL"     mapstructure:"baseURL"`
	Title       string            `yaml:"title"       mapstructure:"title"`
	Description string            `yaml:"description" mapstructure:"description"`
	Language    string            `yaml:"language"    mapstructure:"language"`
	Theme       string            `yaml:"theme"       mapstructure:"theme"`
	Author      AuthorConfig      `yaml:"author"      mapstructure:"author"`
	Menu        MenuConfig        `yaml:"menu"        mapstructure:"menu"`
	Pagination  PaginationConfig  `yaml:"pagination"  mapstructure:"pagination"`
	Taxonomies  map[string]string `yaml:"taxonomies"  mapstructure:"taxonomies"`
	Highlight   HighlightConfig   `yaml:"highlight"   mapstructure:"highlight"`
	Search      SearchConfig      `yaml:"search"      mapstructure:"search"`
	Feeds       FeedsConfig       `yaml:"feeds"       mapstructure:"feeds"`
	SEO         SEOConfig         `yaml:"seo"         mapstructure:"seo"`
	Server      ServerConfig      `yaml:"server"      mapstructure:"server"`
	Build       BuildConfig       `yaml:"build"       mapstructure:"build"`
	Deploy      DeployConfig      `yaml:"deploy"      mapstructure:"deploy"`
	Images      ImageConfig       `yaml:"images"      mapstructure:"images"`
	Security    SecurityConfig    `yaml:"security"    mapstructure:"security"`
	Params      map[string]any    `yaml:"params"      mapstructure:"params"`
}

// AuthorConfig holds information about the site author.
type AuthorConfig struct {
	Name   string       `yaml:"name"   mapstructure:"name"`
	Email  string       `yaml:"email"  mapstructure:"email"`
	Bio    string       `yaml:"bio"    mapstructure:"bio"`
	Avatar string       `yaml:"avatar" mapstructure:"avatar"`
	Social SocialConfig `yaml:"social" mapstructure:"social"`
}

// SocialConfig holds social media handles for the author.
type SocialConfig struct {
	GitHub   string `yaml:"github"   mapstructure:"github"`
	LinkedIn string `yaml:"linkedin" mapstructure:"linkedin"`
	Twitter  string `yaml:"twitter"  mapstructure:"twitter"`
	Mastodon string `yaml:"mastodon" mapstructure:"mastodon"`
	Email    string `yaml:"email"    mapstructure:"email"`
}

// MenuItem represents a single navigation menu entry.
type MenuItem struct {
	Name   string `yaml:"name"   mapstructure:"name"`
	URL    string `yaml:"url"    mapstructure:"url"`
	Weight int    `yaml:"weight" mapstructure:"weight"`
}

// MenuConfig holds the navigation menus for the site.
type MenuConfig struct {
	Main []MenuItem `yaml:"main" mapstructure:"main"`
}

// PaginationConfig controls how content lists are paginated.
type PaginationConfig struct {
	PageSize int `yaml:"pageSize" mapstructure:"pageSize"`
}

// HighlightConfig controls syntax highlighting behaviour.
type HighlightConfig struct {
	Style       string `yaml:"style"       mapstructure:"style"`
	DarkStyle   string `yaml:"darkStyle"   mapstructure:"darkStyle"`
	LineNumbers bool   `yaml:"lineNumbers" mapstructure:"lineNumbers"`
	TabWidth    int    `yaml:"tabWidth"    mapstructure:"tabWidth"`
}

// SearchConfig controls the client-side search index.
type SearchConfig struct {
	Enabled       bool        `yaml:"enabled"       mapstructure:"enabled"`
	ContentLength int         `yaml:"contentLength" mapstructure:"contentLength"`
	Keys          []SearchKey `yaml:"keys"          mapstructure:"keys"`
}

// SearchKey defines a field and its relevance weight for search indexing.
type SearchKey struct {
	Name   string  `yaml:"name"   mapstructure:"name"`
	Weight float64 `yaml:"weight" mapstructure:"weight"`
}

// FeedsConfig controls RSS/Atom feed generation.
type FeedsConfig struct {
	RSS         bool     `yaml:"rss"         mapstructure:"rss"`
	Atom        bool     `yaml:"atom"        mapstructure:"atom"`
	Limit       int      `yaml:"limit"       mapstructure:"limit"`
	FullContent bool     `yaml:"fullContent" mapstructure:"fullContent"`
	Sections    []string `yaml:"sections"    mapstructure:"sections"`
}

// SEOConfig holds search-engine optimisation settings.
type SEOConfig struct {
	TitleTemplate string `yaml:"titleTemplate" mapstructure:"titleTemplate"`
	DefaultImage  string `yaml:"defaultImage"  mapstructure:"defaultImage"`
	TwitterHandle string `yaml:"twitterHandle" mapstructure:"twitterHandle"`
	JSONLD        bool   `yaml:"jsonLD"        mapstructure:"jsonLD"`
}

// ServerConfig controls the local development server.
type ServerConfig struct {
	Port       int    `yaml:"port"       mapstructure:"port"`
	Host       string `yaml:"host"       mapstructure:"host"`
	LiveReload bool   `yaml:"livereload" mapstructure:"livereload"`
}

// BuildConfig controls the site build process.
type BuildConfig struct {
	Minify    bool `yaml:"minify"    mapstructure:"minify"`
	CleanURLs bool `yaml:"cleanUrls" mapstructure:"cleanUrls"`
}

// DeployConfig holds deployment target configuration.
type DeployConfig struct {
	Endpoint   string           `yaml:"endpoint"   mapstructure:"endpoint"`
	Profile    string           `yaml:"profile"    mapstructure:"profile"`
	S3         S3Config         `yaml:"s3"         mapstructure:"s3"`
	CloudFront CloudFrontConfig `yaml:"cloudfront" mapstructure:"cloudfront"`
}

// S3Config holds AWS S3 deployment settings.
type S3Config struct {
	Bucket string `yaml:"bucket" mapstructure:"bucket"`
	Region string `yaml:"region" mapstructure:"region"`
}

// CloudFrontConfig holds AWS CloudFront invalidation settings.
type CloudFrontConfig struct {
	DistributionID  string   `yaml:"distributionId"  mapstructure:"distributionId"`
	InvalidatePaths []string `yaml:"invalidatePaths" mapstructure:"invalidatePaths"`
	URLRewrite      bool     `yaml:"urlRewrite"      mapstructure:"urlRewrite"`
	SecurityHeaders bool     `yaml:"securityHeaders" mapstructure:"securityHeaders"`
}

// ImageConfig controls responsive image generation and format conversion.
type ImageConfig struct {
	Enabled bool     `yaml:"enabled" mapstructure:"enabled"`
	Quality int      `yaml:"quality" mapstructure:"quality"`
	Sizes   []int    `yaml:"sizes"   mapstructure:"sizes"`
	Formats []string `yaml:"formats" mapstructure:"formats"`
}

// SecurityConfig controls security header generation.
type SecurityConfig struct {
	Enabled bool       `yaml:"enabled" mapstructure:"enabled"`
	CSP     CSPConfig  `yaml:"csp"     mapstructure:"csp"`
	HSTS    HSTSConfig `yaml:"hsts"    mapstructure:"hsts"`
}

// CSPConfig holds Content Security Policy directive sources.
type CSPConfig struct {
	ScriptSrc  []string `yaml:"scriptSrc"  mapstructure:"scriptSrc"`
	StyleSrc   []string `yaml:"styleSrc"   mapstructure:"styleSrc"`
	ImgSrc     []string `yaml:"imgSrc"     mapstructure:"imgSrc"`
	ConnectSrc []string `yaml:"connectSrc" mapstructure:"connectSrc"`
	FontSrc    []string `yaml:"fontSrc"    mapstructure:"fontSrc"`
}

// HSTSConfig holds HTTP Strict Transport Security settings.
type HSTSConfig struct {
	MaxAge            int  `yaml:"maxAge"            mapstructure:"maxAge"`
	IncludeSubDomains bool `yaml:"includeSubDomains" mapstructure:"includeSubDomains"`
	Preload           bool `yaml:"preload"           mapstructure:"preload"`
}

// Default returns a SiteConfig populated with sensible default values.
func Default() *SiteConfig {
	return &SiteConfig{
		Language: "en",
		Theme:    "default",
		Pagination: PaginationConfig{
			PageSize: 10,
		},
		Taxonomies: map[string]string{
			"tag":      "tags",
			"category": "categories",
		},
		Highlight: HighlightConfig{
			Style:     "github",
			DarkStyle: "github-dark",
			TabWidth:  4,
		},
		Search: SearchConfig{
			Enabled:       true,
			ContentLength: 5000,
			Keys: []SearchKey{
				{Name: "title", Weight: 2.0},
				{Name: "tags", Weight: 1.5},
				{Name: "summary", Weight: 1.0},
				{Name: "content", Weight: 0.5},
			},
		},
		Feeds: FeedsConfig{
			RSS:   true,
			Atom:  true,
			Limit: 20,
		},
		SEO: SEOConfig{
			JSONLD: true,
		},
		Server: ServerConfig{
			Port:       1313,
			Host:       "localhost",
			LiveReload: true,
		},
		Build: BuildConfig{},
		Images: ImageConfig{
			Enabled: true,
			Quality: 75,
			Sizes:   []int{320, 640, 960, 1280, 1920},
			Formats: []string{"webp", "original"},
		},
		Security: SecurityConfig{
			Enabled: false,
			HSTS: HSTSConfig{
				MaxAge:            31536000,
				IncludeSubDomains: true,
			},
		},
		Params: map[string]any{},
	}
}

// Load reads a configuration file from configPath (YAML or TOML) and returns
// a SiteConfig with defaults applied first and file values overlaid on top.
func Load(configPath string) (*SiteConfig, error) {
	cfg := Default()

	v := viper.New()

	// Determine format from extension.
	ext := strings.TrimPrefix(filepath.Ext(configPath), ".")
	switch ext {
	case "yaml", "yml":
		v.SetConfigType("yaml")
	case "toml":
		v.SetConfigType("toml")
	default:
		// Default to yaml if unrecognised.
		v.SetConfigType("yaml")
	}

	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// Validate checks the SiteConfig for common errors.
// It returns a descriptive error if:
//   - Title is empty
//   - BaseURL has a trailing slash
func (c *SiteConfig) Validate() error {
	if strings.TrimSpace(c.Title) == "" {
		return fmt.Errorf("config: title is required")
	}

	if c.BaseURL != "" && strings.HasSuffix(c.BaseURL, "/") {
		return fmt.Errorf("config: baseURL must not have a trailing slash (got %q)", c.BaseURL)
	}

	return nil
}

// WithOverrides applies CLI flag overrides to the config. Known keys are
// mapped to their corresponding struct fields. The modified config is returned
// for convenient chaining.
func (c *SiteConfig) WithOverrides(overrides map[string]any) *SiteConfig {
	for key, val := range overrides {
		switch key {
		case "baseURL":
			if s, ok := val.(string); ok {
				c.BaseURL = s
			}
		case "title":
			if s, ok := val.(string); ok {
				c.Title = s
			}
		case "theme":
			if s, ok := val.(string); ok {
				c.Theme = s
			}
		case "language":
			if s, ok := val.(string); ok {
				c.Language = s
			}
		case "port":
			if n, ok := val.(int); ok {
				c.Server.Port = n
			}
		case "host":
			if s, ok := val.(string); ok {
				c.Server.Host = s
			}
		case "minify":
			if b, ok := val.(bool); ok {
				c.Build.Minify = b
			}
		case "cleanUrls":
			if b, ok := val.(bool); ok {
				c.Build.CleanURLs = b
			}
		case "livereload":
			if b, ok := val.(bool); ok {
				c.Server.LiveReload = b
			}
		}
	}
	return c
}
