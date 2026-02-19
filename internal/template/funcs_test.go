package template

import (
	"testing"
)

func TestJoin(t *testing.T) {
	tests := []struct {
		name  string
		sep   string
		items any
		want  string
	}{
		{
			name:  "string slice",
			sep:   ",",
			items: []string{"go", "rust", "python"},
			want:  "go,rust,python",
		},
		{
			name:  "empty slice",
			sep:   ",",
			items: []string{},
			want:  "",
		},
		{
			name:  "single element",
			sep:   ",",
			items: []string{"solo"},
			want:  "solo",
		},
		{
			name:  "different separator",
			sep:   " | ",
			items: []string{"a", "b", "c"},
			want:  "a | b | c",
		},
		{
			name:  "non-slice falls back to Sprint",
			sep:   ",",
			items: 42,
			want:  "42",
		},
		{
			name:  "non-slice string",
			sep:   ",",
			items: "hello",
			want:  "hello",
		},
		{
			name:  "nil input",
			sep:   ",",
			items: nil,
			want:  "",
		},
		{
			name:  "any slice from params",
			sep:   ",",
			items: []any{"Go", "TypeScript"},
			want:  "Go,TypeScript",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := join(tt.sep, tt.items)
			if got != tt.want {
				t.Errorf("join(%q, %v) = %q, want %q", tt.sep, tt.items, got, tt.want)
			}
		})
	}
}
