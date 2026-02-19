package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDataFiles_YAML(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	yamlContent := `name: Alice
age: 30
hobbies:
  - reading
  - hiking
`
	if err := os.WriteFile(filepath.Join(dataDir, "resume.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadDataFiles(dataDir)
	if err != nil {
		t.Fatalf("LoadDataFiles() error = %v", err)
	}

	resume, ok := result["resume"]
	if !ok {
		t.Fatal("result[\"resume\"] not found")
	}

	m, ok := resume.(map[string]any)
	if !ok {
		t.Fatalf("resume is %T, want map[string]any", resume)
	}

	if got, ok := m["name"].(string); !ok || got != "Alice" {
		t.Errorf("name = %v, want %q", m["name"], "Alice")
	}
	if got, ok := m["age"].(int); !ok || got != 30 {
		t.Errorf("age = %v, want 30", m["age"])
	}

	hobbies, ok := m["hobbies"].([]any)
	if !ok {
		t.Fatalf("hobbies is %T, want []any", m["hobbies"])
	}
	if len(hobbies) != 2 {
		t.Errorf("len(hobbies) = %d, want 2", len(hobbies))
	}
	if hobbies[0] != "reading" {
		t.Errorf("hobbies[0] = %v, want %q", hobbies[0], "reading")
	}
}

func TestLoadDataFiles_YML(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ymlContent := `title: Test
value: 42
`
	if err := os.WriteFile(filepath.Join(dataDir, "info.yml"), []byte(ymlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadDataFiles(dataDir)
	if err != nil {
		t.Fatalf("LoadDataFiles() error = %v", err)
	}

	info, ok := result["info"]
	if !ok {
		t.Fatal("result[\"info\"] not found")
	}

	m, ok := info.(map[string]any)
	if !ok {
		t.Fatalf("info is %T, want map[string]any", info)
	}

	if got, ok := m["title"].(string); !ok || got != "Test" {
		t.Errorf("title = %v, want %q", m["title"], "Test")
	}
}

func TestLoadDataFiles_JSON(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	jsonContent := `{
  "members": [
    {"name": "Bob", "role": "engineer"},
    {"name": "Carol", "role": "designer"}
  ],
  "count": 2
}`
	if err := os.WriteFile(filepath.Join(dataDir, "team.json"), []byte(jsonContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadDataFiles(dataDir)
	if err != nil {
		t.Fatalf("LoadDataFiles() error = %v", err)
	}

	team, ok := result["team"]
	if !ok {
		t.Fatal("result[\"team\"] not found")
	}

	m, ok := team.(map[string]any)
	if !ok {
		t.Fatalf("team is %T, want map[string]any", team)
	}

	members, ok := m["members"].([]any)
	if !ok {
		t.Fatalf("members is %T, want []any", m["members"])
	}
	if len(members) != 2 {
		t.Errorf("len(members) = %d, want 2", len(members))
	}

	// JSON numbers are float64.
	if got, ok := m["count"].(float64); !ok || got != 2 {
		t.Errorf("count = %v, want 2", m["count"])
	}
}

func TestLoadDataFiles_TOML(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `title = "Site Config"
debug = true

[database]
host = "localhost"
port = 5432
`
	if err := os.WriteFile(filepath.Join(dataDir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadDataFiles(dataDir)
	if err != nil {
		t.Fatalf("LoadDataFiles() error = %v", err)
	}

	cfg, ok := result["config"]
	if !ok {
		t.Fatal("result[\"config\"] not found")
	}

	m, ok := cfg.(map[string]any)
	if !ok {
		t.Fatalf("config is %T, want map[string]any", cfg)
	}

	if got, ok := m["title"].(string); !ok || got != "Site Config" {
		t.Errorf("title = %v, want %q", m["title"], "Site Config")
	}
	if got, ok := m["debug"].(bool); !ok || got != true {
		t.Errorf("debug = %v, want true", m["debug"])
	}

	db, ok := m["database"].(map[string]any)
	if !ok {
		t.Fatalf("database is %T, want map[string]any", m["database"])
	}
	if got, ok := db["host"].(string); !ok || got != "localhost" {
		t.Errorf("database.host = %v, want %q", db["host"], "localhost")
	}
}

func TestLoadDataFiles_Nested(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	peopleDir := filepath.Join(dataDir, "people")
	if err := os.MkdirAll(peopleDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Root-level file.
	rootYAML := `site: Forge`
	if err := os.WriteFile(filepath.Join(dataDir, "site.yaml"), []byte(rootYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Nested file.
	teamJSON := `{"lead": "Dave", "size": 5}`
	if err := os.WriteFile(filepath.Join(peopleDir, "team.json"), []byte(teamJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadDataFiles(dataDir)
	if err != nil {
		t.Fatalf("LoadDataFiles() error = %v", err)
	}

	// Check root-level file.
	site, ok := result["site"]
	if !ok {
		t.Fatal("result[\"site\"] not found")
	}
	siteMap, ok := site.(map[string]any)
	if !ok {
		t.Fatalf("site is %T, want map[string]any", site)
	}
	if got := siteMap["site"]; got != "Forge" {
		t.Errorf("site.site = %v, want %q", got, "Forge")
	}

	// Check nested file: result["people"]["team"].
	people, ok := result["people"]
	if !ok {
		t.Fatal("result[\"people\"] not found")
	}
	peopleMap, ok := people.(map[string]any)
	if !ok {
		t.Fatalf("people is %T, want map[string]any", people)
	}

	team, ok := peopleMap["team"]
	if !ok {
		t.Fatal("people[\"team\"] not found")
	}
	teamMap, ok := team.(map[string]any)
	if !ok {
		t.Fatalf("team is %T, want map[string]any", team)
	}
	if got := teamMap["lead"]; got != "Dave" {
		t.Errorf("people.team.lead = %v, want %q", got, "Dave")
	}
}

func TestLoadDataFiles_NonexistentDir(t *testing.T) {
	dir := t.TempDir()
	nonexistent := filepath.Join(dir, "does-not-exist")

	result, err := LoadDataFiles(nonexistent)
	if err != nil {
		t.Fatalf("LoadDataFiles() error = %v, want nil", err)
	}
	if result == nil {
		t.Fatal("result is nil, want empty map")
	}
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestLoadDataFiles_MixedFormats(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	yamlContent := `format: yaml`
	jsonContent := `{"format": "json"}`
	tomlContent := `format = "toml"`

	if err := os.WriteFile(filepath.Join(dataDir, "a.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "b.json"), []byte(jsonContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "c.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadDataFiles(dataDir)
	if err != nil {
		t.Fatalf("LoadDataFiles() error = %v", err)
	}

	// Verify all three files were loaded.
	for _, key := range []string{"a", "b", "c"} {
		entry, ok := result[key]
		if !ok {
			t.Errorf("result[%q] not found", key)
			continue
		}
		m, ok := entry.(map[string]any)
		if !ok {
			t.Errorf("result[%q] is %T, want map[string]any", key, entry)
			continue
		}
		got, ok := m["format"].(string)
		if !ok {
			t.Errorf("result[%q][\"format\"] is %T, want string", key, m["format"])
			continue
		}
		// Each file's "format" value matches its extension.
		wantFormats := map[string]string{"a": "yaml", "b": "json", "c": "toml"}
		if got != wantFormats[key] {
			t.Errorf("result[%q][\"format\"] = %q, want %q", key, got, wantFormats[key])
		}
	}
}

func TestLoadDataFiles_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write invalid YAML content.
	invalidYAML := `
key: value
  bad_indent: oops
    another: broken
  : missing key
`
	badFile := filepath.Join(dataDir, "broken.yaml")
	if err := os.WriteFile(badFile, []byte(invalidYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadDataFiles(dataDir)
	if err == nil {
		t.Fatal("LoadDataFiles() expected error for invalid YAML, got nil")
	}
	if result != nil {
		t.Errorf("result should be nil on error, got %v", result)
	}

	// Error message should contain the file path.
	if !strings.Contains(err.Error(), "broken.yaml") {
		t.Errorf("error should mention file path, got: %v", err)
	}
}
