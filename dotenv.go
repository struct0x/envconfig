package envconfig

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// EnvFileLookup returns a lookup function that reads environment variables
// from a .env file. It panics if a file cannot be read.
// The .env file should have lines in the format KEY=VALUE.
// Comments starting with # are ignored.
// Empty lines are ignored.
// Notes:
//   - If both the .env file and OS environment define a key, the OS environment value wins.
//   - Lines like `export KEY=VALUE` are supported.
func EnvFileLookup(filePath string) func(string) (string, bool) {
	envMap := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		panic(fmt.Sprintf("envconfig: reading %q: %v", filePath, err))
	}

	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "envconfig: failed to close %q file: %v", filePath, err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	const (
		initialBufSize = 64 * 1024
		maxBufSize     = 1024 * 1024
	)
	scanner.Buffer(make([]byte, 0, initialBufSize), maxBufSize)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		quoted := false
		if len(value) > 0 && (value[0] == '"' || value[0] == '\'') {
			q := value[0] // " or '

			if idx := strings.LastIndexByte(value, q); idx > 0 {
				inner := value[1:idx]
				rest := strings.TrimSpace(value[idx+1:])

				// If rest starts with a comment, ignore it entirely.
				if rest == "" || strings.HasPrefix(rest, "#") {
					value = inner
					quoted = true
				}
				// If the rest doesn't start with #, we fall through to unquoted handling,
			}
		}

		if !quoted {
			// Remove inline comment for unquoted values: value [space]# comment
			// We do not strip # without prior whitespace to allow "value#partofvalue".
			if idx := strings.Index(value, "#"); idx >= 0 {
				// Only consider as comment if there's whitespace before the '#'
				// e.g., "value # comment", not "value#partofvalue"
				spaceIdx := strings.LastIndexAny(value[:idx], " \t")
				if spaceIdx >= len(value[:idx])-1 {
					value = strings.TrimSpace(value[:idx])
				}
			}
		}

		envMap[key] = value
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Sprintf("envconfig: scanning %q: %v", filePath, err))
	}

	return func(key string) (string, bool) {
		if value, exists := os.LookupEnv(key); exists {
			return value, true
		}

		if value, exists := envMap[key]; exists {
			return value, true
		}

		return "", false
	}
}
