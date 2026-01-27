# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-01-27

### Changed

- Improved test coverage for error cases in `setValue`

### Removed

- Removed `assert` functionality (was not part of core API)

## [0.3.0] - 2026-01-26

### Added

- `EnvCollector` interface for handling dynamic environment variables that can't be expressed via struct tags (e.g., `USER_1`, `PASS_1`, `USER_2`, `PASS_2`)
- `EnvGetter` interface with `Lookup`, `ReadValue`, and `Read` methods for use within collectors
- Pointer validation in `EnvGetter.ReadValue`

### Changed

- Custom unmarshalers (`TextUnmarshaler`, etc.) now require `env` tag, not `envPrefix`

### Removed

- Whole-struct decoding via `envPrefix` (was buggy, returned nil-wrapped errors)

### Fixed

- README: OS env takes precedence over `.env` file, not vice versa

## [0.2.0] - 2025-10-24

### Added

- `Validator` interface for custom field validation after population
- Code coverage badge workflow

## [0.1.1] - 2025-09-27

### Changed

- Simplified internal decoding logic

## [0.1.0] - 2025-09-13

### Added

- **Core API**: Generic `Read[T]` function to populate structs from environment variables
- **Struct Tags**:
  - `env:"NAME"` - specify environment variable name
  - `envDefault:"VALUE"` - fallback when variable is unset
  - `envRequired:"true"` - error if variable is unset and no default
  - `envPrefix:"PREFIX"` - prefix for nested struct fields
  - `env:"-"` - skip field entirely
- **Nested Structs**: Support for embedded/anonymous structs with optional prefix
- **Type Support**:
  - Primitives: string, bool, all int/uint sizes, float32/64
  - `time.Duration` (parsed via `time.ParseDuration`)
  - Slices: comma-separated values (e.g., `"a,b,c"`)
  - Arrays: comma-separated values with length validation
  - Maps: comma-separated key=value pairs (e.g., `"k1=v1,k2=v2"`)
  - Pointers to any supported type (auto-allocated)
- **Custom Decoding**: Support for types implementing `encoding.TextUnmarshaler`, `encoding.BinaryUnmarshaler`, `json.Unmarshaler`
- **Dotenv Support**: `LoadDotEnv()` helper to load `.env` files
- **Custom Lookup**: Pass custom lookup function to `Read` for alternative env sources
- **IgnoreEmptyEnvLookup**: Helper that treats empty values as unset
- Strict validation for missing/invalid tags

### Fixed

- Treat empty env values as nil slice/map (not empty allocation)
- Avoid unnecessary map allocation for empty values
