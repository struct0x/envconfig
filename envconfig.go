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
// Embedded vs named struct fields:
//   - Embedded (anonymous) struct fields are treated "flat" by default
//     (no extra prefix). To prefix an embedded subtree, put `envPrefix` on
//     the embedded field.
//   - Named struct fields may also carry `envPrefix:"PFX"`; they must NOT
//     also have an `env` tag.
//
// Whole-struct (single-key) decoding:
//
//	If a struct-typed field (or embedded struct) has an effective prefix PFX_,
//	and the holder type implements one of the standard decoders below, a single
//	env variable named "PFX" (without the trailing underscore) can be used to
//	populate the entire struct at once. When present, this whole-struct value
//	takes precedence and field-by-field decoding is skipped.
//	Supported decoders:
//	  - encoding.TextUnmarshaler
//	  - encoding.BinaryUnmarshaler
//	  - json.Unmarshaler
//
// Supported field types:
//   - primitives: string, bool, all int/uint sizes, float32/64
//   - time.Duration (parsed via time.ParseDuration)
//   - arrays, slices: comma-separated values (e.g. "a,b,c")
//   - maps: comma-separated k=v pairs (e.g. "k1=v1,k2=v2"); split on first "="
//   - pointers to any supported type (allocated as needed)
//   - any type implementing encoding.TextUnmarshaler / BinaryUnmarshaler / json.Unmarshaler
//
// Precedence per leaf field:
//  1. If lookupEnv returns (value, ok==true), that value is used as-is
//     (even if value is the empty string "").
//  2. Else, if `envDefault` is present, it is used.
//  3. Else, if `envRequired:"true"`, Read returns an error.
//  4. Else, the field is left at its zero value.
//
// Validation & errors:
//   - holder must be a non-nil pointer to a struct.
//   - Non-embedded struct fields must have either `env` or `envPrefix`
//     (or be explicitly skipped with `env:"-"`); otherwise an error is returned.
//   - Struct fields must not specify both `env` and `envPrefix`.
//   - `envPrefix` must not be empty when present.
//   - Parsing/conversion failures return errors that include the env key.
//   - Unsupported leaf types (that do not implement a supported unmarshal
//     interface) cause an error.
//   - any type can implement Validator interface, and it will be called as soon as value if populated.
//
// Note on empties:
//
//	An env var that is present but empty (lookup ok == true, value == "") is
//	considered "set": it suppresses `envDefault` and does not trigger
//	`envRequired`. If you want defaulting on empty strings, use IgnoreEmptyEnvLookup,
//	which wraps os.LookupEnv and treats empty values as unset (returns ok == false when value == "").
func Read[T any](holder *T, lookupEnv ...func(string) (string, bool)) error {
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

type Validator interface {
	Validate() error
}

func read(le func(string) (string, bool), prefix string, holder any) error {
	if len(prefix) > 0 {
		if err, ok := tryUnmarshalKnownInterface(le, prefix, holder); ok {
			return fmt.Errorf("envconfig: %q prefix failed to populate: %w", prefix, err)
		}
	}

	holderPtr := reflect.ValueOf(holder)
	holderValue := holderPtr.Elem()
	fields := reflect.VisibleFields(holderValue.Type())

	for _, field := range fields {
		env, hasEnv := field.Tag.Lookup("env")
		pref, hasPrefix := field.Tag.Lookup("envPrefix")
		if env == "-" {
			continue
		}
		if (hasEnv && env == "") && !hasPrefix {
			return fmt.Errorf("envconfig: tag \"env\" can't be empty: %q", field.Name)
		}

		fieldVal := holderValue.FieldByName(field.Name)

		if !hasEnv && !hasPrefix && !field.Anonymous && fieldVal.CanSet() {
			return fmt.Errorf("envconfig: field %q does not have \"env\" or \"envPrefix\" tags. Ignore it explicitly with `env:\"-\"` or embed to treat it flat", field.Name)
		}

		ft := field.Type
		if ft.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(ft.Elem()))
			}
			ft = ft.Elem()
			fieldVal = fieldVal.Elem()
		}

		if field.Anonymous {
			if hasEnv {
				return fmt.Errorf("envconfig: %q is embedded use \"envPrefix\" to add prefix or remove \"env\" to treat struct flat", field.Name)
			}

			prefix = ""
			if hasPrefix && pref == "" {
				return fmt.Errorf("envconfig: %q field with empty \"envPrefix\" tag", field.Name)
			} else if pref != "" {
				prefix = pref + "_"
			}

			err := read(le, prefix, fieldVal.Addr().Interface())
			if err != nil {
				return err
			}
			continue
		}

		if ft.Kind() == reflect.Struct && hasPrefix {
			if pref == "" {
				return fmt.Errorf("envconfig: %q field with empty \"envPrefix\" tag", field.Name)
			}
			if hasEnv {
				return fmt.Errorf("envconfig: struct %q can't have both \"envPrefix\" and \"env\" tags", field.Name)
			}

			err := read(le, prefix+pref+"_", fieldVal.Addr().Interface())
			if err != nil {
				return err
			}
			continue
		}

		envVal, ok := le(prefix + env)
		if !ok {
			if defaultVal := field.Tag.Get("envDefault"); defaultVal != "" {
				envVal = defaultVal
			} else if field.Tag.Get("envRequired") == "true" {
				return fmt.Errorf("envconfig: required field %q is empty", prefix+env)
			} else {
				continue
			}
		}

		if err := setValue(fieldVal, envVal); err != nil {
			return fmt.Errorf("envconfig: %q failed to populate: %w", field.Name, err)
		}

		if validator, ok := reflect.TypeAssert[Validator](fieldVal); ok {
			if err := validator.Validate(); err != nil {
				return fmt.Errorf("envconfig: %q failed to validate: %w", field.Name, err)
			}
		}
	}

	if validator, ok := reflect.TypeAssert[Validator](holderPtr); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("envconfig: failed to validate: %w", err)
		}
	}

	return nil
}

func tryUnmarshalKnownInterface(le func(string) (string, bool), prefix string, holder any) (error, bool) {
	if i, ok := holder.(encoding.TextUnmarshaler); ok {
		envValue, ok := le(prefix[:len(prefix)-1])
		if !ok {
			return nil, true
		}

		if err := i.UnmarshalText([]byte(envValue)); err != nil {
			return err, true
		}
	}
	if i, ok := holder.(encoding.BinaryUnmarshaler); ok {
		envValue, ok := le(prefix[:len(prefix)-1])
		if !ok {
			return nil, true
		}

		if err := i.UnmarshalBinary([]byte(envValue)); err != nil {
			return err, true
		}
	}
	if i, ok := holder.(json.Unmarshaler); ok {
		envValue, ok := le(prefix[:len(prefix)-1])
		if !ok {
			return nil, true
		}

		if err := i.UnmarshalJSON([]byte(envValue)); err != nil {
			return err, true
		}
	}
	return nil, false
}

var durationType = reflect.TypeOf(time.Duration(0))

func setValue(inp reflect.Value, value string) error {
	if inp.Kind() == reflect.Ptr {
		if inp.IsNil() {
			inp.Set(reflect.New(inp.Type().Elem()))
		}
		return setValue(inp.Elem(), value)
	}

	if inp.CanAddr() {
		if u, ok := inp.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return u.UnmarshalText([]byte(value))
		}
		if u, ok := inp.Addr().Interface().(encoding.BinaryUnmarshaler); ok {
			return u.UnmarshalBinary([]byte(value))
		}
		if u, ok := inp.Addr().Interface().(json.Unmarshaler); ok {
			return u.UnmarshalJSON([]byte(value))
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
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
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
