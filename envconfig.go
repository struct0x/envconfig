package envconfig

import (
	"encoding"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type LookupEnv = func(string) (string, bool)

// Read populates holder (a pointer to struct) using the provided lookup function to resolve values.
//
// Usage:
//
//	type C struct {
//	  Port int    `env:"PORT" envDefault:"8080"`
//	  TLS  struct {
//	    Enabled bool   `env:"ENABLED"`
//	    Cert    string `env:"CERT" envRequired:"true"`
//	  } `envPrefix:"TLS"` // effective keys: TLS_ENABLED, TLS_CERT
//	}
//	var cfg C
//	if err := envconfig.Read(&cfg); err != nil { log.Fatal(err) }
//
// Lookup source:
//
//	By default Read uses os.LookupEnv. You may pass a custom lookup function,
//	e.g., envconfig.Read(&cfg, myLookup) where myLookup has signature func(string) (string, bool).
//
// Tags (per field):
//   - `env:"NAME"`        : the environment variable name for this field.
//     Use `env:"-"` to skip the field entirely.
//   - `envDefault:"VAL"`  : fallback used only when the variable is UNSET
//     (i.e., lookup returns ok == false). If the variable
//     is present but empty ("", ok == true), the empty
//     value is used and default does NOT apply.
//   - `envRequired:"true"`: if the variable is UNSET and no envDefault is
//     provided, Read returns an error. Only the literal
//     "true" enables this behavior.
//   - `envPrefix:"PFX"`   : for struct-typed fields (including embedded/
//     anonymous ones). Applies a prefix to all descendant
//     leaf env names. Prefixes are joined with "_".
//     Example: `envPrefix:"DB"` -> DB_HOST, DB_PORT.
//
// Embedded and named struct fields:
//   - Embedded (anonymous) and named struct fields are treated "flat" by default
//     (no extra prefix). To prefix a subtree, put `envPrefix` on
//     the field.
//
// Advanced use with EnvCollector:
//
//	EnvCollector is a special interface that can be implemented on a pointer receiver.
//	When encountered, it will be called, moving env reading into the implementor.
//	This is an advanced interface that helps to populate values that cannot be represented using struct tags.
//
// Supported field types:
//   - primitives: string, bool, all int/uint sizes, float32/64
//   - time.Duration (parsed via time.ParseDuration)
//   - arrays, slices: comma-separated values (e.g. "a,b,c")
//   - maps: comma-separated k=v pairs (e.g. "k1=v1,k2=v2"); split on first "="
//   - pointers to any supported type (allocated as needed)
//   - any type implementing (in the priority) json.Unmarshaler > encoding.BinaryUnmarshaler > encoding.TextUnmarshaler
//
// Precedence per leaf field:
//  1. If lookupEnv returns (value, ok==true), that value is used as-is
//     (even if value is the empty string "").
//  2. Else, if `envDefault` is present, it is used.
//  3. Else, if `envRequired:"true"`, Read returns an error.
//  4. Else, the field is left at its zero value.
//
// Errors when:
//   - `env` tag is empty
//   - Struct with `env` tag but no unmarshal interface
//   - Exported values without `env` tag
//   - EnvCollector with value (non-pointer) receiver
//   - A required field is missing and no default is provided
//   - holder is nil or not a pointer to a struct
//   - Struct fields specify both `env` and `envPrefix`
//   - `envPrefix` is empty when present
//   - Parsing/conversion failures (returned errors includes the env key)
//   - Unsupported leaf types (that do not implement a supported unmarshal interface)
func Read[T any](holder *T, lookupEnv ...LookupEnv) error {
	if holder == nil {
		return fmt.Errorf("envconfig: nil holder")
	}

	lookupEnvFunc := os.LookupEnv
	if len(lookupEnv) >= 1 {
		lookupEnvFunc = lookupEnv[0]
	}

	tp := reflect.TypeOf(holder)
	if tp.Kind() != reflect.Ptr {
		panic("envconfig: unreachable")
	}

	tp = tp.Elem()
	if tp.Kind() != reflect.Struct {
		return fmt.Errorf("envconfig.Read only accepts a struct, got %q", tp.Kind().String())
	}

	return read(lookupEnvFunc, "", holder)
}

// EnvGetter provides a convenient way to get values from env variables.
// It is passed to EnvCollector.CollectEnv to allow a custom env collection.
// Under the hood it uses the provided Lookup in Read function.
type EnvGetter interface {
	// Lookup performs a raw lookup for an environment variable.
	Lookup(key string) (string, bool)

	// ReadValue parses a single environment variable into a target.
	// Target must be a pointer.
	ReadValue(key string, target any) error

	// ReadIntoStruct populates the target struct adding envPrefix + "_" for all `env` tags found on a struct.
	// Target must be a pointer to a struct
	ReadIntoStruct(envPrefix string, target any) error
}

type getter struct {
	lookup LookupEnv
}

func (g *getter) Lookup(key string) (string, bool) {
	return g.lookup(key)
}

func (g *getter) ReadValue(key string, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("%q not a pointer", v.Type())
	}

	val, ok := g.lookup(key)
	if !ok {
		return nil
	}

	return setValue(v, val)
}

func (g *getter) ReadIntoStruct(prefix string, target any) error {
	tp := reflect.TypeOf(target)
	if tp.Kind() != reflect.Ptr {
		return fmt.Errorf("envconfig: Read target must be a pointer, got %q", tp.Kind())
	}
	if tp.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("envconfig: Read target must be a pointer to struct, got pointer to %q", tp.Elem().Kind())
	}
	return read(g.lookup, prefix+"_", target)
}

// EnvCollector is an advanced interface for collecting custom environment variables
// that can't be easily expressed via struct tags.
// For example, a custom collector can handle environment variables with complex
// naming conventions like USER_1, PASS_1, USER_2, PASS_2.
//
// See TestEnvCollector for a concrete example.
type EnvCollector interface {
	// CollectEnv is called with an EnvGetter for reading env values.
	CollectEnv(env EnvGetter) error
}

func read(le func(string) (string, bool), prefix string, holder any) error {
	holderPtr := reflect.ValueOf(holder)
	holderValue := holderPtr.Elem()
	fields := reflect.VisibleFields(holderValue.Type())

	for _, field := range fields {
		if field.PkgPath != "" {
			continue
		}

		fieldVal := holderValue.FieldByName(field.Name)
		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(ft.Elem()))
			}
			ft = ft.Elem()
			fieldVal = fieldVal.Elem()
		}

		if fieldVal.CanAddr() {
			if fieldVal.Type().Implements(envCollectorType) {
				return fmt.Errorf("envconfig: field %q implements EnvCollector but not for a pointer receiver", field.Name)
			}

			if collector, ok := fieldVal.Addr().Interface().(EnvCollector); ok {
				get := &getter{lookup: le}
				if err := collector.CollectEnv(get); err != nil {
					return fmt.Errorf("envconfig: %q CollectEnv failed: %w", field.Name, err)
				}
				continue
			}
		}

		env, hasEnv := field.Tag.Lookup("env")
		if env == "-" {
			continue
		}
		if hasEnv && env == "" {
			return fmt.Errorf("envconfig: tag \"env\" can't be empty: %q", field.Name)
		}

		pref, hasPrefix := field.Tag.Lookup("envPrefix")
		if hasEnv && hasPrefix {
			return fmt.Errorf("envconfig: both \"env\"  and \"envPrefix\" does not make sense. If a field is a struct pick \"envPrefix\" if you want to populate it using composite env keys, use \"env\" if you implement encoding.TextUnmarshaler / encoding.BinaryUnmarshaler / json.Unmarshaler, or remove tags to treat is flat")
		}
		if hasPrefix && pref == "" {
			return fmt.Errorf("envconfig: tag \"envPrefix\" can't be empty: %q", field.Name)
		}

		if ft.Kind() == reflect.Struct {
			if !hasEnv && !hasPrefix {
				if err := read(le, prefix, fieldVal.Addr().Interface()); err != nil {
					return err
				}

				continue
			}

			if hasPrefix {
				realPref := pref + "_"
				if prefix != "" {
					realPref = prefix + realPref
				}
				if err := read(le, realPref, fieldVal.Addr().Interface()); err != nil {
					return err
				}

				continue
			}
		}

		if !hasEnv {
			return fmt.Errorf("envconfig: field %q does not have \"env\" tag", field.Name)
		}

		envVal, ok := le(prefix + env)
		if !ok {
			defaultVal, hasDefault := field.Tag.Lookup("envDefault")
			if !hasDefault && field.Tag.Get("envRequired") == "true" {
				return fmt.Errorf("envconfig: required field %q is empty", prefix+env)
			} else if !hasDefault {
				continue
			}

			envVal = defaultVal
		}

		if fieldVal.CanAddr() {
			var fn func(val []byte) error
			if u, ok := fieldVal.Addr().Interface().(encoding.TextUnmarshaler); ok {
				fn = u.UnmarshalText
			}
			if u, ok := fieldVal.Addr().Interface().(encoding.BinaryUnmarshaler); ok {
				fn = u.UnmarshalBinary
			}
			if u, ok := fieldVal.Addr().Interface().(json.Unmarshaler); ok {
				fn = u.UnmarshalJSON
			}

			if fn != nil {
				if err := fn([]byte(envVal)); err != nil {
					return fmt.Errorf("envconfig: error decoding %q field: %w", field.Name, err)
				}
				continue
			}
			if fieldVal.Kind() == reflect.Struct {
				return fmt.Errorf("envconfig: field %q is a struct with \"env\" tag but does not implement encoding.TextUnmarshaler / encoding.BinaryUnmarshaler / json.Unmarshaler", field.Name)
			}
		}

		if err := setValue(fieldVal, envVal); err != nil {
			return fmt.Errorf("envconfig: field %q failed to populate: %w", field.Name, err)
		}
	}

	return nil
}

var (
	durationType     = reflect.TypeOf(time.Duration(0))
	byteSliceType    = reflect.TypeOf([]byte{})
	envCollectorType = reflect.TypeOf((*EnvCollector)(nil)).Elem()
)

func setValue(inp reflect.Value, value string) error {
	if inp.Kind() == reflect.Ptr {
		if inp.IsNil() {
			inp.Set(reflect.New(inp.Type().Elem()))
		}
		return setValue(inp.Elem(), value)
	}

	if inp.CanAddr() {
		if u, ok := inp.Addr().Interface().(json.Unmarshaler); ok {
			return u.UnmarshalJSON([]byte(value))
		}
		if u, ok := inp.Addr().Interface().(encoding.BinaryUnmarshaler); ok {
			return u.UnmarshalBinary([]byte(value))
		}
		if u, ok := inp.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return u.UnmarshalText([]byte(value))
		}
	}

	switch inp.Type() {
	case durationType:
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		inp.Set(reflect.ValueOf(d))
		return nil

	case byteSliceType:
		inp.Set(reflect.ValueOf([]byte(value)))
		return nil
	}

	switch inp.Kind() {
	case reflect.String:
		inp.SetString(value)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		inp.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bits := inp.Type().Bits()
		i, err := strconv.ParseInt(value, 10, bits)
		if err != nil {
			return err
		}
		inp.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bits := inp.Type().Bits()
		u, err := strconv.ParseUint(value, 10, bits)
		if err != nil {
			return err
		}
		inp.SetUint(u)
	case reflect.Float32, reflect.Float64:
		bits := inp.Type().Bits()
		f, err := strconv.ParseFloat(value, bits)
		if err != nil {
			return err
		}
		inp.SetFloat(f)
	case reflect.Array:
		arr := split(value)
		if len(arr) < inp.Len() {
			return fmt.Errorf("array needs %d elements, got %d", inp.Len(), len(arr))
		}
		for i := 0; i < inp.Len(); i++ {
			err := setValue(inp.Index(i), arr[i])
			if err != nil {
				return err
			}
		}
	case reflect.Slice:
		arr := split(value)
		for i := 0; i < len(arr); i++ {
			elem := reflect.New(inp.Type().Elem()).Elem()
			err := setValue(elem, arr[i])
			if err != nil {
				return err
			}
			inp.Set(reflect.Append(inp, elem))
		}
	case reflect.Map:
		arr := split(value)
		if len(arr) == 0 {
			return nil
		}
		mp := reflect.MakeMap(inp.Type())
		for i := 0; i < len(arr); i++ {
			kv := strings.SplitN(arr[i], "=", 2)
			if len(kv) != 2 {
				return fmt.Errorf("invalid map value %s", value)
			}
			key := reflect.New(inp.Type().Key()).Elem()
			err := setValue(key, strings.TrimSpace(kv[0]))
			if err != nil {
				return err
			}
			val := reflect.New(inp.Type().Elem()).Elem()
			err = setValue(val, kv[1])
			if err != nil {
				return err
			}
			mp.SetMapIndex(key, val)
		}
		inp.Set(mp)
	default:
		return fmt.Errorf("unsupported type %q it's not primitive nor implements supported unmarshaling interfaces", inp.Type())
	}

	return nil
}

func split(s string) []string {
	if s == "" {
		return nil
	}

	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, it := range raw {
		out = append(out, strings.TrimSpace(it))
	}
	return out
}
