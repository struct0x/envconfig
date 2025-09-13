package envconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvFileReader(t *testing.T) {
	tempDir, err := os.MkdirTemp(t.TempDir(), "envconfig-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	tests := []struct {
		name        string
		fileContent string
		lookupKey   string
		expected    string
		exists      bool
	}{
		{
			name:        "basic_key_value",
			fileContent: "KEY1=value1\nKEY2=value2",
			lookupKey:   "KEY1",
			expected:    "value1",
			exists:      true,
		},
		{
			name:        "export",
			fileContent: "export KEY1=value1\nKEY2=value2",
			lookupKey:   "KEY1",
			expected:    "value1",
			exists:      true,
		},
		{
			name:        "with_spaces",
			fileContent: "SPACED_KEY = spaced value ",
			lookupKey:   "SPACED_KEY",
			expected:    "spaced value",
			exists:      true,
		},
		{
			name:        "with_quotes",
			fileContent: `QUOTED_KEY="quoted value"`,
			lookupKey:   "QUOTED_KEY",
			expected:    "quoted value",
			exists:      true,
		},
		{
			name:        "with_single_quotes",
			fileContent: "SINGLE_QUOTED='single quoted'",
			lookupKey:   "SINGLE_QUOTED",
			expected:    "single quoted",
			exists:      true,
		},
		{
			name:        "with_quotes_and_comments",
			fileContent: "SINGLE_QUOTED='single quoted' # This is comment",
			lookupKey:   "SINGLE_QUOTED",
			expected:    "single quoted",
			exists:      true,
		},
		{
			name:        "with_quotes_and_comments_inside",
			fileContent: "SINGLE_QUOTED='single quoted # This is comment'",
			lookupKey:   "SINGLE_QUOTED",
			expected:    "single quoted # This is comment",
			exists:      true,
		},
		{
			name:        "comments_inside",
			fileContent: "SINGLE_QUOTED=singlequoted#Thisis comment",
			lookupKey:   "SINGLE_QUOTED",
			expected:    "singlequoted#Thisis comment",
			exists:      true,
		},
		{
			name:        "comments_removed",
			fileContent: "SINGLE_QUOTED=singlequoted #Thisis comment",
			lookupKey:   "SINGLE_QUOTED",
			expected:    "singlequoted",
			exists:      true,
		},
		{
			name:        "with_comments",
			fileContent: "# This is a comment\nCOMMENTED=after comment\n# Another comment",
			lookupKey:   "COMMENTED",
			expected:    "after comment",
			exists:      true,
		},
		{
			name:        "with_empty_lines",
			fileContent: "\n\nEMPTY_LINES=value\n\n",
			lookupKey:   "EMPTY_LINES",
			expected:    "value",
			exists:      true,
		},
		{
			name:        "invalid_line_format",
			fileContent: "INVALID_LINE\nVALID_LINE=value",
			lookupKey:   "VALID_LINE",
			expected:    "value",
			exists:      true,
		},
		{
			name:        "missing_key",
			fileContent: "OTHER_KEY=value",
			lookupKey:   "MISSING_KEY",
			expected:    "",
			exists:      false,
		},
		{
			name:        "empty_file",
			fileContent: "",
			lookupKey:   "ANY_KEY",
			expected:    "",
			exists:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			envFile := filepath.Join(tempDir, tc.name+".env")
			if err := os.WriteFile(envFile, []byte(tc.fileContent), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			lookupFn := EnvFileLookup(envFile)

			value, exists := lookupFn(tc.lookupKey)
			if exists != tc.exists {
				t.Errorf("Expected exists=%v, got %v", tc.exists, exists)
			}
			if exists && value != tc.expected {
				t.Errorf("Expected value=%q, got %q", tc.expected, value)
			}
		})
	}
}

func TestEnvFileReaderUnknownFile(t *testing.T) {
	defer func() {
		_ = recover()
	}()

	EnvFileLookup("non_existent.env")
	t.Fatalf("should not be called")
}
