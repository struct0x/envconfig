package envconfig

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// AssertOpt represents a single validation check that returns an error if validation fails.
// AssertOpt functions are designed to be composable and can be passed to Assert() to perform
// multiple validations at once.
//
// Example:
//
//	func (cfg MyConfig) Validate() error {
//	    return Assert(
//	        NotEmpty(cfg.Host, "HOST"),
//	        Range(cfg.Port, 1, 65535, "PORT"),
//	    )
//	}
type AssertOpt func() error

// Assert runs all provided validation checks and collects any errors that occur.
// If any validation fails, it returns an ErrValidation containing all failures.
// If all validations pass, it returns nil.
//
// Assert is designed to be used with validation helper functions like NotEmpty, Range, etc.
// All validations are executed regardless of failures, allowing users to see all
// validation issues at once rather than fixing them one at a time.
//
// Example:
//
//	func (cfg Config) Validate() error {
//	    return Assert(
//	        NotEmpty(cfg.APIKey, "API_KEY"),
//	        Range(cfg.Port, 1, 65535, "PORT"),
//	        OneOf(cfg.Environment, "ENVIRONMENT", "dev", "staging", "production"),
//	    )
//	}
func Assert(opts ...AssertOpt) error {
	var errs []error
	for _, opt := range opts {
		if err := opt(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return ErrValidation(errs)
}

// ErrValidation is a collection of validation errors that occurred during Assert().
// It implements the error interface and formats multiple errors into a single,
// human-readable error message.
type ErrValidation []error

// Error returns a formatted string containing all validation errors,
// separated by semicolons.
func (e ErrValidation) Error() string {
	if len(e) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("validation failed:")
	for i, err := range e {
		if i > 0 {
			sb.WriteString(";")
		}
		sb.WriteString(" ")
		sb.WriteString(err.Error())
	}
	return sb.String()
}

// NotEmpty validates that a string value is not empty.
// Returns an AssertOpt that fails if the value is an empty string.
//
// Parameters:
//   - value: the string to validate
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	NotEmpty(cfg.APIKey, "API_KEY")
func NotEmpty(value, field string) AssertOpt {
	return func() error {
		if value == "" {
			return fmt.Errorf("%s: must not be empty", field)
		}
		return nil
	}
}

// Number is a constraint for all numeric types that can be compared
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Range validates that an integer value falls within a specified range (inclusive).
// Returns an AssertOpt that fails if the value is less than min or greater than max.
//
// Parameters:
//   - value: the integer to validate
//   - min: the minimum allowed value (inclusive)
//   - max: the maximum allowed value (inclusive)
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	Range(cfg.Port, 1, 65535, "PORT")
func Range[T Number](value, min, max T, field string) AssertOpt {
	return func() error {
		if value < min || value > max {
			return fmt.Errorf("%s: must be between %v and %v, got %v", field, min, max, value)
		}
		return nil
	}
}

// Positive validates that an integer value is greater than zero.
// Returns an AssertOpt that fails if the value is less than or equal to zero.
//
// Parameters:
//   - value: the integer to validate
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	Positive(cfg.Workers, "WORKERS")
func Positive[T Number](value T, field string) AssertOpt {
	return func() error {
		if value <= 0 {
			return fmt.Errorf("%s: must be positive, got %v", field, value)
		}
		return nil
	}
}

// NonNegative validates that an integer value is greater than or equal to zero.
// Returns an AssertOpt that fails if the value is negative.
//
// Parameters:
//   - value: the integer to validate
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	NonNegative(cfg.Retries, "RETRIES")
func NonNegative[T Number](value T, field string) AssertOpt {
	return func() error {
		if value < 0 {
			return fmt.Errorf("%s: must be non-negative, got %v", field, value)
		}
		return nil
	}
}

// OneOf validates that a string value matches one of the allowed values.
// The comparison is case-sensitive.
// Returns an AssertOpt that fails if the value is not in the allowed list.
//
// Parameters:
//   - value: the string to validate
//   - field: the name of the field (used in error messages)
//   - allowed: the list of allowed values
//
// Example:
//
//	OneOf(cfg.LogLevel, "LOG_LEVEL", "debug", "info", "warn", "error")
func OneOf(value string, field string, allowed ...string) AssertOpt {
	return func() error {
		for _, a := range allowed {
			if value == a {
				return nil
			}
		}
		return fmt.Errorf("%s: must be one of %v, got %q", field, allowed, value)
	}
}

// Custom validates a custom condition and returns a specified error message if it fails.
// This is a generic validator for arbitrary conditions that don't fit other helpers.
// Returns an AssertOpt that fails if the condition is false.
//
// Parameters:
//   - condition: the boolean condition to check
//   - field: the name of the field (used in error messages)
//   - message: the error message to return if the condition is false
//
// Example:
//
//	Custom(cfg.MaxRetries < cfg.Timeout, "TIMEOUT", "must be greater than MAX_RETRIES")
func Custom(condition bool, field, message string) AssertOpt {
	return func() error {
		if !condition {
			return fmt.Errorf("%s: %s", field, message)
		}
		return nil
	}
}

// MinLength validates that a string has at least the specified minimum length.
// Returns an AssertOpt that fails if the string length is less than min.
//
// Parameters:
//   - value: the string to validate
//   - min: the minimum required length
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	MinLength(cfg.Password, 8, "PASSWORD")
func MinLength(value string, min int, field string) AssertOpt {
	return func() error {
		if len(value) < min {
			return fmt.Errorf("%s: minimum length is %d, got %d", field, min, len(value))
		}
		return nil
	}
}

// MaxLength validates that a string does not exceed the specified maximum length.
// Returns an AssertOpt that fails if the string length is greater than max.
//
// Parameters:
//   - value: the string to validate
//   - max: the maximum allowed length
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	MaxLength(cfg.Username, 50, "USERNAME")
func MaxLength(value string, max int, field string) AssertOpt {
	return func() error {
		if len(value) > max {
			return fmt.Errorf("%s: maximum length is %d, got %d", field, max, len(value))
		}
		return nil
	}
}

// Pattern validates that a string matches the specified regular expression pattern.
// Returns an AssertOpt that fails if the string does not match the pattern or if
// the pattern itself is invalid.
//
// Parameters:
//   - value: the string to validate
//   - field: the name of the field (used in error messages)
//   - pattern: the regular expression pattern to match against
//
// Example:
//
//	Pattern(cfg.Email, "EMAIL", `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
func Pattern(value, field, pattern string) AssertOpt {
	return func() error {
		matched, err := regexp.MatchString(pattern, value)
		if err != nil {
			return fmt.Errorf("%s: invalid pattern: %w", field, err)
		}
		if !matched {
			return fmt.Errorf("%s: must match pattern %q", field, pattern)
		}
		return nil
	}
}

// URL validates that a string is a valid URL according to Go's url.Parse.
// Returns an AssertOpt that fails if the URL cannot be parsed.
//
// Parameters:
//   - value: the URL string to validate
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	URL(cfg.APIEndpoint, "API_ENDPOINT")
func URL(value, field string) AssertOpt {
	return func() error {
		if value == "" {
			return fmt.Errorf("%s: must not be empty", field)
		}
		if _, err := url.Parse(value); err != nil {
			return fmt.Errorf("%s: invalid URL: %w", field, err)
		}
		return nil
	}
}

// FileExists validates that a file or directory exists at the specified path.
// Returns an AssertOpt that fails if the path does not exist.
//
// Parameters:
//   - path: the file or directory path to check
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	FileExists(cfg.ConfigFile, "CONFIG_FILE")
func FileExists(path, field string) AssertOpt {
	return func() error {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("%s: file does not exist: %s", field, path)
		} else if err != nil {
			return fmt.Errorf("%s: cannot access file: %w", field, err)
		}
		return nil
	}
}

// MinSliceLen validates that a slice has at least the specified minimum length.
// Returns an AssertOpt that fails if the slice length is less than min.
//
// Parameters:
//   - length: the actual length of the slice
//   - min: the minimum required length
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	MinSliceLen(len(cfg.Servers), 1, "SERVERS")
func MinSliceLen(length, min int, field string) AssertOpt {
	return func() error {
		if length < min {
			return fmt.Errorf("%s: minimum length is %d, got %d", field, min, length)
		}
		return nil
	}
}

// MaxSliceLen validates that a slice does not exceed the specified maximum length.
// Returns an AssertOpt that fails if the slice length is greater than max.
//
// Parameters:
//   - length: the actual length of the slice
//   - max: the maximum allowed length
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	MaxSliceLen(len(cfg.Tags), 10, "TAGS")
func MaxSliceLen(length, max int, field string) AssertOpt {
	return func() error {
		if length > max {
			return fmt.Errorf("%s: maximum length is %d, got %d", field, max, length)
		}
		return nil
	}
}

// NotEquals validates that a value does not equal the forbidden value.
// This is a generic function that works with any comparable type.
// Returns an AssertOpt that fails if the value equals the forbidden value.
//
// Parameters:
//   - value: the value to validate
//   - forbidden: the value that should not be matched
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	NotEquals(cfg.Port, 22, "PORT") // disallow SSH port
//	NotEquals(cfg.Mode, "insecure", "MODE")
//	NotEquals(cfg.AdminPassword, "admin", "ADMIN_PASSWORD")
func NotEquals[T comparable](value, forbidden T, field string) AssertOpt {
	return func() error {
		if value == forbidden {
			return fmt.Errorf("%s: must not equal %v", field, forbidden)
		}
		return nil
	}
}

// NotBlank validates that a string is not empty and not just whitespace.
// NotEmpty is already defined, but here's NotBlank for completeness
// Returns an AssertOpt that fails if the value is empty or contains only whitespace.
//
// Parameters:
//   - value: the string to validate
//   - field: the name of the field (used in error messages)
//
// Example:
//
//	NotBlank(cfg.APIKey, "API_KEY")
func NotBlank(value, field string) AssertOpt {
	return func() error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s: must not be blank", field)
		}
		return nil
	}
}

// Not inverts any AssertOpt, making it fail when the original would succeed
// and succeed when the original would fail.
// This is useful for creating negative assertions from existing validators.
// Returns an AssertOpt that inverts the result of the provided validator.
//
// Parameters:
//   - opt: the AssertOpt to invert
//   - customMessage: optional custom error message (if empty, a generic message is used)
//
// Example:
//
//	Not(OneOf(cfg.Environment, "ENV", "production", "staging"), "must not be production or staging")
//	Not(Pattern(cfg.Username, "USERNAME", `^admin.*`), "username must not start with 'admin'")
func Not(opt AssertOpt, customMessage string) AssertOpt {
	return func() error {
		err := opt()
		if err == nil {
			if customMessage != "" {
				return fmt.Errorf("%s", customMessage)
			}
			return fmt.Errorf("validation condition must not be true")
		}
		return nil
	}
}
