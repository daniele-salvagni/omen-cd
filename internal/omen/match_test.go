package omen

import "testing"

func TestMatchAny(t *testing.T) {
	cases := []struct {
		name  string
		globs []string
		files []string
		want  bool
	}{
		{"exact hit", []string{"a/b.yml"}, []string{"a/b.yml"}, true},
		{"doublestar", []string{"stacks/web/**"}, []string{"stacks/web/docker-compose.yml"}, true},
		{"no match", []string{"a/**"}, []string{"b/x"}, false},
		{"empty files", []string{"**"}, nil, false},
		{"empty globs", nil, []string{"a"}, false},
		{"any of many", []string{"x/*", "stacks/*/compose.yml"}, []string{"unrelated.md", "stacks/web/compose.yml"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchAny(tc.globs, tc.files); got != tc.want {
				t.Fatalf("matchAny=%v want %v", got, tc.want)
			}
		})
	}
}
