package practical

import (
	"os"
	"strings"
	"testing"
)

func TestRegexReplaceFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		pattern  string
		repl     string
		expected string
	}{
		{
			name: "No image tags to replace",
			input: `name: nginx
		version: 1.0.0
		description: no image tags here
		`,
			pattern: `image: (.*)`,
			repl:    "image: abc.com/$1",
			expected: `name: nginx
		version: 1.0.0
		description: no image tags here
		`,
		},
		{
			name: "Replace with complex image names",
			input: `image: complex-app_v2.1.3-rc1:20230915
		image: another-app:2.0
		`,
			pattern: `image: (.*)$`,
			repl:    "image: docker.io/$1",
			expected: `image: docker.io/complex-app_v2.1.3-rc1:20230915
		image: docker.io/another-app:2.0
		`,
		},
		{
			name: "Real case",
			input: `ncdPostgres:
  	image: registry.i.ncmps.com/images/postgres:latest
	initContainers:
    - name: init-postgres-check
      image: registry.i.ncmps.com/golang/ncd_golang:cluster
		`,
			pattern: `image: (.*)$`,
			repl:    "image: reg.ncmps.com:14000/$1",
			expected: `ncdPostgres:
  	image: reg.ncmps.com:14000/registry.i.ncmps.com/images/postgres:latest
	initContainers:
    - name: init-postgres-check
      image: reg.ncmps.com:14000/registry.i.ncmps.com/golang/ncd_golang:cluster
		`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile, err := os.CreateTemp("", "testfile-*.txt")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())
			defer tempFile.Close()

			if _, err := tempFile.WriteString(tt.input); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}

			_, err = RegexReplaceFile(tempFile.Name(), tt.pattern, tt.repl)
			if err != nil {
				t.Errorf("editLines failed: %v", err)
			}

			modifiedContent, err := os.ReadFile(tempFile.Name())
			if err != nil {
				t.Fatalf("Failed to read modified content: %v", err)
			}

			if strings.TrimSpace(string(modifiedContent)) != strings.TrimSpace(tt.expected) {
				t.Errorf("Test %s failed.\nExpected:\n%s\nGot:\n%s", tt.name, tt.expected, string(modifiedContent))
			}
		})
	}
}
