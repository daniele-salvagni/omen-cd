package omen

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadEnvFile reads KEY=VALUE lines from path and sets them in the process
// environment. Format follows systemd's EnvironmentFile: `#` comments and
// blank lines are ignored, surrounding single/double quotes are stripped,
// no shell expansion.
//
// If required is true, a missing file is an error. Otherwise, missing is
// silently OK. Malformed lines are always an error.
func LoadEnvFile(path string, required bool) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) && !required {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		i := strings.IndexByte(text, '=')
		if i < 0 {
			return fmt.Errorf("%s:%d: missing '='", path, line)
		}
		key := strings.TrimSpace(text[:i])
		val := unquote(strings.TrimSpace(text[i+1:]))
		if err := os.Setenv(key, val); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func unquote(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
