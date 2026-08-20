package omen

import "github.com/bmatcuk/doublestar/v4"

// matchAny reports whether any file matches any glob. Globs use doublestar
// syntax so `**` expands across directory boundaries.
func matchAny(globs, files []string) bool {
	for _, g := range globs {
		for _, f := range files {
			if ok, _ := doublestar.Match(g, f); ok {
				return true
			}
		}
	}
	return false
}
