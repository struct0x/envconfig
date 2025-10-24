package envconfig_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/struct0x/envconfig"
)

func TestAssert(t *testing.T) {
	t.Run("no_errors", func(t *testing.T) {
		err := envconfig.Assert(
			envconfig.NotEmpty("value", "FIELD1"),
			envconfig.Range(5, 1, 10, "FIELD2"),
			envconfig.Positive(1, "FIELD3"),
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("errors", func(t *testing.T) {
		err := envconfig.Assert(
			envconfig.NotEmpty("", "FIELD1"),
			envconfig.Range(100, 1, 10, "FIELD2"),
			envconfig.Positive(-1, "FIELD3"),
		)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		var errVal envconfig.ErrValidation
		if !errors.As(err, &errVal) {
			t.Fatalf("Expected ErrValidation, got %T", err)
		}
		if len(errVal) != 3 {
			t.Errorf("Expected 3 errors, got %d", len(errVal))
		}

		errStr := err.Error()
		if !strings.Contains(errStr, "FIELD1") {
			t.Errorf("Expected error to contain 'FIELD1', got %v", errStr)
		}
		if !strings.Contains(errStr, "FIELD2") {
			t.Errorf("Expected error to contain 'FIELD2', got %v", errStr)
		}
		if !strings.Contains(errStr, "FIELD3") {
			t.Errorf("Expected error to contain 'FIELD3', got %v", errStr)
		}
	})

	t.Run("empty_opts", func(t *testing.T) {
		err := envconfig.Assert()
		if err != nil {
			t.Errorf("Expected no error for empty opts, got %v", err)
		}
	})
}

func TestNotEmpty(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		field     string
		wantError bool
	}{
		{"valid", "value", "FIELD", false},
		{"empty", "", "FIELD", true},
		{"whitespace", "   ", "FIELD", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.NotEmpty(tt.value, tt.field)()
			if (err != nil) != tt.wantError {
				t.Errorf("NotEmpty() error = %v, wantError %v", err, tt.wantError)
			}

			if err != nil && !strings.Contains(err.Error(), tt.field) {
				t.Errorf("Error should contain field name %q, got %v", tt.field, err)
			}
		})
	}
}

func TestRange(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		min       int
		max       int
		field     string
		wantError bool
	}{
		{"within_range", 5, 1, 10, "PORT", false},
		{"at_min", 1, 1, 10, "PORT", false},
		{"at_max", 10, 1, 10, "PORT", false},
		{"below_min", 0, 1, 10, "PORT", true},
		{"above_max", 11, 1, 10, "PORT", true},
		{"negative_range", -5, -10, -1, "FIELD", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.Range(tt.value, tt.min, tt.max, tt.field)()
			if (err != nil) != tt.wantError {
				t.Errorf("Range() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil {
				if !strings.Contains(err.Error(), tt.field) {
					t.Errorf("Error should contain field name %q, got %v", tt.field, err)
				}
				if !strings.Contains(err.Error(), fmt.Sprintf("%d", tt.value)) {
					t.Errorf("Error should contain value %d, got %v", tt.value, err)
				}
			}
		})
	}
}

func TestPositive(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		field     string
		wantError bool
	}{
		{"positive", 1, "COUNT", false},
		{"large_positive", 1000, "COUNT", false},
		{"zero", 0, "COUNT", true},
		{"negative", -1, "COUNT", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.Positive(tt.value, tt.field)()
			if (err != nil) != tt.wantError {
				t.Errorf("Positive() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), tt.field) {
				t.Errorf("Error should contain field name %q, got %v", tt.field, err)
			}
		})
	}
}

func TestNonNegative(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		field     string
		wantError bool
	}{
		{"positive", 1, "RETRIES", false},
		{"zero", 0, "RETRIES", false},
		{"negative", -1, "RETRIES", true},
		{"large_negative", -100, "RETRIES", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.NonNegative(tt.value, tt.field)()
			if (err != nil) != tt.wantError {
				t.Errorf("NonNegative() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), tt.field) {
				t.Errorf("Error should contain field name %q, got %v", tt.field, err)
			}
		})
	}
}

func TestOneOf(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		field     string
		allowed   []string
		wantError bool
	}{
		{"valid_first", "dev", "ENV", []string{"dev", "staging", "prod"}, false},
		{"valid_middle", "staging", "ENV", []string{"dev", "staging", "prod"}, false},
		{"valid_last", "prod", "ENV", []string{"dev", "staging", "prod"}, false},
		{"invalid", "test", "ENV", []string{"dev", "staging", "prod"}, true},
		{"case_sensitive", "Dev", "ENV", []string{"dev", "staging", "prod"}, true},
		{"empty_not_allowed", "", "ENV", []string{"dev", "staging", "prod"}, true},
		{"single_allowed", "only", "ENV", []string{"only"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.OneOf(tt.value, tt.field, tt.allowed...)()
			if (err != nil) != tt.wantError {
				t.Errorf("OneOf() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil {
				if !strings.Contains(err.Error(), tt.field) {
					t.Errorf("Error should contain field name %q, got %v", tt.field, err)
				}
				if !strings.Contains(err.Error(), tt.value) {
					t.Errorf("Error should contain value %q, got %v", tt.value, err)
				}
			}
		})
	}
}

func TestCustom(t *testing.T) {
	tests := []struct {
		name      string
		condition bool
		field     string
		message   string
		wantError bool
	}{
		{"true_condition", true, "FIELD", "custom message", false},
		{"false_condition", false, "FIELD", "custom message", true},
		{"complex_condition", 5 > 3, "FIELD", "should be greater", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.Custom(tt.condition, tt.field, tt.message)()
			if (err != nil) != tt.wantError {
				t.Errorf("Custom() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil {
				if !strings.Contains(err.Error(), tt.field) {
					t.Errorf("Error should contain field name %q, got %v", tt.field, err)
				}
				if !strings.Contains(err.Error(), tt.message) {
					t.Errorf("Error should contain message %q, got %v", tt.message, err)
				}
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		min       int
		field     string
		wantError bool
	}{
		{"exact_min", "abc", 3, "PASSWORD", false},
		{"above_min", "abcde", 3, "PASSWORD", false},
		{"below_min", "ab", 3, "PASSWORD", true},
		{"empty_string", "", 1, "PASSWORD", true},
		{"zero_min", "", 0, "PASSWORD", false},
		{"unicode", "日本語", 3, "TEXT", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.MinLength(tt.value, tt.min, tt.field)()
			if (err != nil) != tt.wantError {
				t.Errorf("MinLength() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), tt.field) {
				t.Errorf("Error should contain field name %q, got %v", tt.field, err)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		max       int
		field     string
		wantError bool
	}{
		{"exact_max", "abc", 3, "USERNAME", false},
		{"below_max", "ab", 3, "USERNAME", false},
		{"above_max", "abcd", 3, "USERNAME", true},
		{"empty_string", "", 10, "USERNAME", false},
		{"zero_max", "a", 0, "USERNAME", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.MaxLength(tt.value, tt.max, tt.field)()
			if (err != nil) != tt.wantError {
				t.Errorf("MaxLength() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), tt.field) {
				t.Errorf("Error should contain field name %q, got %v", tt.field, err)
			}
		})
	}
}

func TestPattern(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		field     string
		pattern   string
		wantError bool
	}{
		{"valid_email", "test@example.com", "EMAIL", `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, false},
		{"invalid_email", "invalid.email", "EMAIL", `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, true},
		{"digits_only", "12345", "CODE", `^\d+$`, false},
		{"letters_in_digits", "123a5", "CODE", `^\d+$`, true},
		{"invalid_pattern", "value", "FIELD", `[`, true},
		{"empty_value", "", "FIELD", `^.+$`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.Pattern(tt.value, tt.field, tt.pattern)()
			if (err != nil) != tt.wantError {
				t.Errorf("Pattern() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), tt.field) {
				t.Errorf("Error should contain field name %q, got %v", tt.field, err)
			}
		})
	}
}

func TestURL(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		field     string
		wantError bool
	}{
		{"valid_http", "http://example.com", "API_URL", false},
		{"valid_https", "https://example.com", "API_URL", false},
		{"valid_with_path", "https://example.com/api/v1", "API_URL", false},
		{"valid_with_query", "https://example.com?key=value", "API_URL", false},
		{"valid_with_port", "http://localhost:8080", "API_URL", false},
		{"relative_url", "/api/v1", "API_URL", false}, // url.Parse accepts relative URLs
		{"empty_string", "", "API_URL", true},
		{"invalid_url", "ht4tp://invalid", "API_URL", false}, // url.Parse is lenient
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.URL(tt.value, tt.field)()
			if (err != nil) != tt.wantError {
				t.Errorf("URL() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), tt.field) {
				t.Errorf("Error should contain field name %q, got %v", tt.field, err)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Remove(existingFile); err != nil {
			t.Errorf("Failed to remove test file: %v", err)
		}
	})

	tests := []struct {
		name      string
		path      string
		field     string
		wantError bool
	}{
		{"existing_file", existingFile, "CONFIG_FILE", false},
		{"existing_dir", tempDir, "CONFIG_DIR", false},
		{"non_existent", filepath.Join(tempDir, "missing.txt"), "CONFIG_FILE", true},
		{"empty_path", "", "CONFIG_FILE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.FileExists(tt.path, tt.field)()
			if (err != nil) != tt.wantError {
				t.Errorf("FileExists() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), tt.field) {
				t.Errorf("Error should contain field name %q, got %v", tt.field, err)
			}
		})
	}
}

func TestMinSliceLen(t *testing.T) {
	tests := []struct {
		name      string
		length    int
		min       int
		field     string
		wantError bool
	}{
		{"exact_min", 3, 3, "SERVERS", false},
		{"above_min", 5, 3, "SERVERS", false},
		{"below_min", 2, 3, "SERVERS", true},
		{"zero_length", 0, 1, "SERVERS", true},
		{"zero_min", 0, 0, "SERVERS", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.MinSliceLen(tt.length, tt.min, tt.field)()
			if (err != nil) != tt.wantError {
				t.Errorf("MinSliceLen() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), tt.field) {
				t.Errorf("Error should contain field name %q, got %v", tt.field, err)
			}
		})
	}
}

func TestMaxSliceLen(t *testing.T) {
	tests := []struct {
		name      string
		length    int
		max       int
		field     string
		wantError bool
	}{
		{"exact_max", 3, 3, "TAGS", false},
		{"below_max", 2, 3, "TAGS", false},
		{"above_max", 4, 3, "TAGS", true},
		{"zero_length", 0, 10, "TAGS", false},
		{"zero_max", 1, 0, "TAGS", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := envconfig.MaxSliceLen(tt.length, tt.max, tt.field)()
			if (err != nil) != tt.wantError {
				t.Errorf("MaxSliceLen() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), tt.field) {
				t.Errorf("Error should contain field name %q, got %v", tt.field, err)
			}
		})
	}
}

func TestComposableValidators(t *testing.T) {
	requireHTTPS := func(urlStr, field string) envconfig.AssertOpt {
		return func() error {
			return envconfig.Assert(
				envconfig.NotEmpty(urlStr, field),
				envconfig.URL(urlStr, field),
				envconfig.Custom(strings.HasPrefix(urlStr, "https://"), field, "must use HTTPS"),
			)
		}
	}

	t.Run("valid_https", func(t *testing.T) {
		err := requireHTTPS("https://example.com", "API_URL")()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("invalid_http", func(t *testing.T) {
		err := requireHTTPS("http://example.com", "API_URL")()
		if err == nil {
			t.Fatal("Expected error for HTTP URL")
		}
		if !strings.Contains(err.Error(), "HTTPS") {
			t.Errorf("Expected error about HTTPS, got %v", err)
		}
	})

	t.Run("empty_url", func(t *testing.T) {
		err := requireHTTPS("", "API_URL")()
		if err == nil {
			t.Fatal("Expected error for empty URL")
		}
	})
}

func TestNotEquals(t *testing.T) {
	t.Run("integers", func(t *testing.T) {
		tests := []struct {
			name      string
			value     int
			forbidden int
			field     string
			wantError bool
		}{
			{"different_values", 8080, 22, "PORT", false},
			{"equal_values", 22, 22, "PORT", true},
			{"zero_vs_nonzero", 0, 1, "COUNT", false},
			{"both_zero", 0, 0, "COUNT", true},
			{"negative_values", -1, -2, "OFFSET", false},
			{"equal_negatives", -5, -5, "OFFSET", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := envconfig.NotEquals(tt.value, tt.forbidden, tt.field)()
				if (err != nil) != tt.wantError {
					t.Errorf("NotEquals() error = %v, wantError %v", err, tt.wantError)
				}
				if err != nil {
					if !strings.Contains(err.Error(), tt.field) {
						t.Errorf("Error should contain field name %q, got %v", tt.field, err)
					}
					if !strings.Contains(err.Error(), fmt.Sprintf("%v", tt.forbidden)) {
						t.Errorf("Error should contain forbidden value %v, got %v", tt.forbidden, err)
					}
				}
			})
		}
	})

	t.Run("strings", func(t *testing.T) {
		tests := []struct {
			name      string
			value     string
			forbidden string
			field     string
			wantError bool
		}{
			{"different_strings", "allowed", "forbidden", "PASSWORD", false},
			{"equal_strings", "admin", "admin", "PASSWORD", true},
			{"case_sensitive", "Admin", "admin", "USERNAME", false},
			{"empty_vs_nonempty", "", "forbidden", "FIELD", false},
			{"both_empty", "", "", "FIELD", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := envconfig.NotEquals(tt.value, tt.forbidden, tt.field)()
				if (err != nil) != tt.wantError {
					t.Errorf("NotEquals() error = %v, wantError %v", err, tt.wantError)
				}
				if err != nil && !strings.Contains(err.Error(), tt.field) {
					t.Errorf("Error should contain field name %q, got %v", tt.field, err)
				}
			})
		}
	})

	t.Run("floats", func(t *testing.T) {
		tests := []struct {
			name      string
			value     float64
			forbidden float64
			field     string
			wantError bool
		}{
			{"different_floats", 3.14, 2.71, "PI", false},
			{"equal_floats", 1.5, 1.5, "RATIO", true},
			{"zero_float", 0.0, 0.0, "VALUE", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := envconfig.NotEquals(tt.value, tt.forbidden, tt.field)()
				if (err != nil) != tt.wantError {
					t.Errorf("NotEquals() error = %v, wantError %v", err, tt.wantError)
				}
			})
		}
	})

	t.Run("booleans", func(t *testing.T) {
		tests := []struct {
			name      string
			value     bool
			forbidden bool
			field     string
			wantError bool
		}{
			{"true_vs_false", true, false, "FLAG", false},
			{"false_vs_true", false, true, "FLAG", false},
			{"both_true", true, true, "FLAG", true},
			{"both_false", false, false, "FLAG", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := envconfig.NotEquals(tt.value, tt.forbidden, tt.field)()
				if (err != nil) != tt.wantError {
					t.Errorf("NotEquals() error = %v, wantError %v", err, tt.wantError)
				}
			})
		}
	})
}

func TestNot(t *testing.T) {
	t.Run("invert_passing_validation", func(t *testing.T) {
		// NotEmpty passes for non-empty string
		opt := envconfig.NotEmpty("value", "FIELD")
		err := opt()
		if err != nil {
			t.Fatalf("Setup failed: NotEmpty should pass, got %v", err)
		}

		// Not() should make it fail
		invertedOpt := envconfig.Not(opt, "custom error message")
		err = invertedOpt()
		if err == nil {
			t.Fatal("Expected Not() to invert passing validation to fail")
		}
		if !strings.Contains(err.Error(), "custom error message") {
			t.Errorf("Expected custom error message, got %v", err)
		}
	})

	t.Run("invert_failing_validation", func(t *testing.T) {
		// NotEmpty fails for empty string
		opt := envconfig.NotEmpty("", "FIELD")
		err := opt()
		if err == nil {
			t.Fatal("Setup failed: NotEmpty should fail for empty string")
		}

		// Not() should make it pass
		invertedOpt := envconfig.Not(opt, "should not appear")
		err = invertedOpt()
		if err != nil {
			t.Errorf("Expected Not() to invert failing validation to pass, got %v", err)
		}
	})

	t.Run("default_message_when_empty", func(t *testing.T) {
		opt := envconfig.NotEmpty("value", "FIELD")
		invertedOpt := envconfig.Not(opt, "")
		err := invertedOpt()
		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "must not be true") {
			t.Errorf("Expected default message, got %v", err)
		}
	})

	t.Run("invert_range_validation", func(t *testing.T) {
		// Value is in range [1, 10]
		opt := envconfig.Range(5, 1, 10, "PORT")
		invertedOpt := envconfig.Not(opt, "PORT: must not be in range 1-10")
		err := invertedOpt()
		if err == nil {
			t.Fatal("Expected Not(Range) to fail when value is in range")
		}
	})

	t.Run("invert_oneof_validation", func(t *testing.T) {
		opt := envconfig.OneOf("production", "ENV", "production", "staging")
		invertedOpt := envconfig.Not(opt, "ENV: must not be production or staging")
		err := invertedOpt()
		if err == nil {
			t.Fatal("Expected Not(OneOf) to fail when value is in list")
		}
		if !strings.Contains(err.Error(), "must not be production or staging") {
			t.Errorf("Expected custom message, got %v", err)
		}

		// Value is NOT in allowed list
		opt2 := envconfig.OneOf("development", "ENV", "production", "staging")
		invertedOpt2 := envconfig.Not(opt2, "ENV: must not be production or staging")
		err2 := invertedOpt2()
		if err2 != nil {
			t.Errorf("Expected Not(OneOf) to pass when value is not in list, got %v", err2)
		}
	})

	t.Run("invert_pattern_validation", func(t *testing.T) {
		// Pattern matches
		opt := envconfig.Pattern("admin123", "USERNAME", `^admin.*`)
		invertedOpt := envconfig.Not(opt, "USERNAME: must not start with 'admin'")
		err := invertedOpt()
		if err == nil {
			t.Fatal("Expected Not(Pattern) to fail when pattern matches")
		}

		// Pattern doesn't match
		opt2 := envconfig.Pattern("user123", "USERNAME", `^admin.*`)
		invertedOpt2 := envconfig.Not(opt2, "USERNAME: must not start with 'admin'")
		err2 := invertedOpt2()
		if err2 != nil {
			t.Errorf("Expected Not(Pattern) to pass when pattern doesn't match, got %v", err2)
		}
	})
}

func TestNotEqualsComposition(t *testing.T) {
	type ServerConfig struct {
		Port          int
		AdminPassword string
		Environment   string
	}

	t.Run("valid_config", func(t *testing.T) {
		cfg := ServerConfig{
			Port:          8080,
			AdminPassword: "secure_p@ssw0rd",
			Environment:   "production",
		}

		err := envconfig.Assert(
			envconfig.NotEquals(cfg.Port, 22, "PORT"),
			envconfig.NotEquals(cfg.Port, 3389, "PORT"),
			envconfig.NotEquals(cfg.AdminPassword, "admin", "ADMIN_PASSWORD"),
			envconfig.NotEquals(cfg.AdminPassword, "password", "ADMIN_PASSWORD"),
		)

		if err != nil {
			t.Errorf("Expected valid config to pass, got error: %v", err)
		}
	})

	t.Run("invalid_config_multiple_violations", func(t *testing.T) {
		cfg := ServerConfig{
			Port:          22,
			AdminPassword: "admin",
			Environment:   "production",
		}

		err := envconfig.Assert(
			envconfig.NotEquals(cfg.Port, 22, "PORT"),
			envconfig.NotEquals(cfg.Port, 3389, "PORT"),
			envconfig.NotEquals(cfg.AdminPassword, "admin", "ADMIN_PASSWORD"),
			envconfig.NotEquals(cfg.AdminPassword, "password", "ADMIN_PASSWORD"),
		)

		if err == nil {
			t.Fatal("Expected errors, got nil")
		}

		var errVal envconfig.ErrValidation
		ok := errors.As(err, &errVal)
		if !ok {
			t.Fatalf("Expected ErrValidation, got %T", err)
		}

		if len(errVal) != 2 {
			t.Errorf("Expected 2 errors (PORT and ADMIN_PASSWORD), got %d", len(errVal))
		}
	})
}

func TestNotComposition(t *testing.T) {
	t.Run("composed_validators", func(t *testing.T) {
		// Create a validator that ensures port is NOT in reserved range (0-1023)
		notReservedPort := func(port int, field string) envconfig.AssertOpt {
			return envconfig.Not(
				envconfig.Range(port, 0, 1023, field),
				field+": must not be a reserved port (0-1023)",
			)
		}

		// Test with non-reserved port
		err := notReservedPort(8080, "PORT")()
		if err != nil {
			t.Errorf("Expected non-reserved port to pass, got %v", err)
		}

		// Test with reserved port
		err = notReservedPort(80, "PORT")()
		if err == nil {
			t.Fatal("Expected reserved port to fail")
		}
		if !strings.Contains(err.Error(), "reserved port") {
			t.Errorf("Expected 'reserved port' in error, got %v", err)
		}
	})

	t.Run("not_in_blocklist", func(t *testing.T) {
		notInBlocklist := func(value string, field string, blocklist ...string) envconfig.AssertOpt {
			return envconfig.Not(
				envconfig.OneOf(value, field, blocklist...),
				field+": value is in the blocklist",
			)
		}

		// Test with allowed value
		err := notInBlocklist("secure123", "PASSWORD", "password", "12345", "admin")()
		if err != nil {
			t.Errorf("Expected allowed password to pass, got %v", err)
		}

		// Test with blocked value
		err = notInBlocklist("admin", "PASSWORD", "password", "12345", "admin")()
		if err == nil {
			t.Fatal("Expected blocked password to fail")
		}
		if !strings.Contains(err.Error(), "blocklist") {
			t.Errorf("Expected 'blocklist' in error, got %v", err)
		}
	})
}
