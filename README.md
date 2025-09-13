# envconfig
A small,
dependency-free Go library for loading configuration from environment variables directly into your structs.

It supports nested structs, prefixes, defaults, required fields,
slices, maps, arrays, pointers, durations, and custom (un)marshalers.
A helper is provided to read variables from a .env file.

- Zero dependencies
- Simple, tag-driven API
- Works with standard os.LookupEnv or a custom lookups
- Optional .env file loader (supports comments, export, quoting, inline comments)

## Installation

```bash 
go get github.com/struct0x/envconfig
```

## Quick start

```go
package main

import (
	"fmt"

	"github.com/struct0x/envconfig"
)

type HTTPServer struct {
	Host    string            `env:"HOST" envDefault:"127.0.0.1"`
	Port    int               `env:"PORT" envRequired:"true"`
	Enabled bool              `env:"ENABLED"`
	Tags    []string          `env:"TAGS"`    // "a,b,c" -> []string{"a","b","c"}
	Headers map[string]string `env:"HEADERS"` // "k1=v1,k2=v2"
}

func main() {
	var cfg HTTPServer

	// Use OS environment by default
	if err := envconfig.Read(&cfg); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", cfg)
}
```

Example environment:

```shell
export PORT=8080
export ENABLED=true
export TAGS="alpha,beta"
export HEADERS="X-Req=abc,X-Trace=on"
```

## Using a .env file

Use EnvFileLookup to source values from a .env file. Lines use KEY=VALUE, support comments and export statements, and handle quoted values with inline comments.

```go
package main

import (
	"fmt"

	"github.com/struct0x/envconfig"
)

type App struct {
	Name string `env:"NAME" envDefault:"demo"`
	Port int    `env:"PORT" envRequired:"true"`
}

func main() {
	var cfg App
	
	if err := envconfig.Read(&cfg, envconfig.EnvFileLookup(".env")); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", cfg)
}
```

Notes:
- If both the .env file and the OS define a key, the .env value wins for that lookup.
- EnvFileLookup panics if the file cannot be read.

## Tags

Add struct field tags to control how values are loaded:
- `env`: the env variable name. Use env:"-" to skip a field.
- `envDefault`: fallback value if the variable is not set.
- `envRequired:"true"`: marks the field as required, returns error when not set, and no default provided.
- `envPrefix`: only for struct-typed fields; prepends a prefix (with underscore) for all nested fields under that struct.

Precedence per field:
1. Value from lookupEnv(name)
2. envDefault (if present)
3. Error if `envRequired:"true"`

### Examples

Basic tags:

```go
package main

type DB struct {
	Host string `env:"DB_HOST" envDefault:"localhost"`
	Port int    `env:"DB_PORT" envRequired:"true"`
}

```

Nested with prefix:

```go
package main

type SubConfig struct {
	Enabled bool   `env:"ENABLED"`
	Mode    string `env:"MODE" envDefault:"safe"`
}

type Root struct {
	Name string     `env:"NAME"`
	Sub  *SubConfig `envPrefix:"SUB"` // Reads SUB_ENABLED, SUB_MODE
}
```

Skipping a field:

```go
package main

type T struct {
	Ignored string `env:"-"`
}
```

## Supported types

- string, bool
- Integers: int, int8, int16, int32, int64
- Unsigned integers: uint, uint8, uint16, uint32, uint64
- Floats: float32, float64
- time.Duration via time.ParseDuration
- Arrays and slices (comma-separated values): "a,b,c"
- Maps (comma-separated key=value pairs): "k1=v1,k2=v2"
- Pointers to supported types (allocated when needed)
- Custom types implementing any of:
    - encoding.TextUnmarshaler
    - encoding.BinaryUnmarshaler
    - json.Unmarshaler

If a value cannot be parsed into the target type, `Read` returns a descriptive error.

## Custom lookup (probably don't need this)

You can provide any lookup function with signature `func(string) (string, bool)` —
for example, a map-based lookup in tests:

```go
package main

import (
	"github.com/struct0x/envconfig"
)

func mapLookup(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

type C struct {
	N int `env:"N"`
}

func main() {
	var c C
	_ = envconfig.Read(&c, mapLookup(map[string]string{"N": "42"}))
}
```

## Error handling

`Read` returns an error when:
- The holder is not a non-nil pointer to a struct
- A required field is missing and no default is provided
- A value cannot be parsed into the target type

Errors include the env variable name and context to aid debugging.


## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
