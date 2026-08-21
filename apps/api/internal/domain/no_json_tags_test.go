package domain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoJSONTags enforces the layering convention: domain types are
// transport-agnostic; wire-format json tags belong on the delivery DTOs.
func TestNoJSONTags(t *testing.T) {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, "`json:") {
				t.Errorf("%s:%d: domain struct carries a json tag; put it on the delivery DTO instead:\n\t%s", path, i+1, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
