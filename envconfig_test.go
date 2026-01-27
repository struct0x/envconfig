package envconfig_test

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/struct0x/envconfig"
)

func TestReadValues(t *testing.T) {
	le := func(key string) (string, bool) {
		switch key {
		case "-":
			t.Fatalf("- should not be searched for.")
		case "A":
			return "hello", true
		case "B", "C", "D", "E", "F", "G", "H", "I", "J", "K":
			return "42", true
		case "L", "M":
			return "42.42", true
		case "N", "R", "S":
			return "true", true
		case "O", "P":
			return "hello, world", true
		case "Q":
			return "key1=value1, key2=value2", true
		case "Z":
			return "embedded_value", true
		case "EMB_ZA":
			return "emb_value", true
		case "CUSTOM":
			return "custom", true
		case "CUSTOM_TEXT":
			return "custom_text", true
		case "CUSTOM_BINARY":
			return "custom_binary", true
		case "CUSTOM_JSON":
			return "custom_json", true
		case "DURATION":
			return "1h", true
		case "SDUR":
			return "1h,2h,3h", true
		case "MDUR":
			return "key1=1h,key2=2h,key3=3h", true
		case "SUB_AA":
			return "sub", true
		case "SUB_SUB2_FF":
			return "aaa", true
		case "URL":
			return "file:///etc/passwd", true
		}

		return "", false
	}
	_ = le

	var cfg Config
	if err := envconfig.Read(&cfg, le); err != nil {
		t.Error(err)
	}

	uri, _ := url.Parse("file:///etc/passwd")

	want := Config{
		NotPopulated: "",
		unexported:   "",
		String:       "hello",
		Int:          42,
		Int8:         42,
		Int16:        42,
		Int32:        42,
		Int64:        42,
		Uint:         42,
		Uint8:        42,
		Uint16:       42,
		Uint32:       42,
		Uint64:       42,
		Float32:      42.42,
		Float64:      42.42,
		Bool:         true,
		ArrString:    [2]string{"hello", "world"},
		SliceString:  []string{"hello", "world"},
		Map: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		PtrBool:       ptr(true),
		PtrPtrBool:    ptr(ptr(true)),
		StringDefault: "Default Value",
		CustomTextUnmarshaler: CustomTextUnmarshaler{
			Value: "***custom_text***",
		},
		CustomBinaryUnmarshaler: CustomBinaryUnmarshaler{
			Value: "***custom_binary2***",
		},
		CustomJSONUnmarshaler: CustomJSONUnmarshaler{
			Value: "***custom_json3***",
		},
		CustomTextUnmarshaler2: CustomTextUnmarshaler{
			Value: "***custom***",
		},
		CustomBinaryUnmarshaler2: CustomBinaryUnmarshaler{
			Value: "***custom2***",
		},
		CustomJSONUnmarshaler2: CustomJSONUnmarshaler{
			Value: "***custom3***",
		},
		Duration: time.Hour,
		SliceDuration: []time.Duration{
			time.Hour,
			2 * time.Hour,
			3 * time.Hour,
		},
		MapDuration: map[string]time.Duration{
			"key1": time.Hour,
			"key2": 2 * time.Hour,
			"key3": 3 * time.Hour,
		},
		URL: uri,
	}

	if diff := reflect.DeepEqual(cfg, want); !diff {
		t.Errorf("expected equal:\n %v", want)
	}
}

func TestReadAdvanced(t *testing.T) {
	le := func(key string) (string, bool) {
		switch key {
		case "AA":
			return "AA", true
		case "SUB2_FF":
			return "aaa", true

		case "PREFIX_AA":
			return "AA", true
		case "PREFIX_SUB2_FF":
			return "aaa", true
		}

		return "", false
	}

	t.Run("embedded_struct", func(t *testing.T) {
		var cfg struct {
			SubConfig
		}
		if err := envconfig.Read(&cfg, le); err != nil {
			t.Error(err)
		}

		want := struct {
			SubConfig
		}{
			SubConfig: SubConfig{
				A: "AA",
				SubSub: SubSubConfig{
					A: "aaa",
				},
			},
		}

		if diff := reflect.DeepEqual(cfg, want); !diff {
			t.Error("expected equal")
		}
	})

	t.Run("struct_without_env", func(t *testing.T) {
		var cfg struct {
			Name SubConfig
		}
		if err := envconfig.Read(&cfg, le); err == nil {
			t.Error(err)
		}
	})

	t.Run("struct_with_empty_env", func(t *testing.T) {
		var cfg struct {
			Name SubConfig `env:""`
		}
		if err := envconfig.Read(&cfg, le); err == nil {
			t.Error(err)
		}
	})

	t.Run("embedded_struct_with_env", func(t *testing.T) {
		var cfg struct {
			SubConfig `env:"ENV"`
		}
		if err := envconfig.Read(&cfg, le); err == nil {
			t.Error("want error")
		}
	})

	t.Run("embedded_struct_with_prefix", func(t *testing.T) {
		var cfg struct {
			SubConfig `envPrefix:"PREFIX"`
		}
		if err := envconfig.Read(&cfg, le); err != nil {
			t.Error(err)
		}

		want := struct {
			SubConfig `envPrefix:"PREFIX"`
		}{
			SubConfig: SubConfig{
				A: "AA",
				SubSub: SubSubConfig{
					A: "aaa",
				},
			},
		}

		if diff := reflect.DeepEqual(cfg, want); !diff {
			t.Error("expected equal")
		}
	})

	t.Run("embedded_struct_with_empty_prefix", func(t *testing.T) {
		var cfg struct {
			SubConfig `envPrefix:""`
		}
		if err := envconfig.Read(&cfg, le); err == nil {
			t.Error("want error")
		}
	})

	t.Run("struct_with_empty_prefix", func(t *testing.T) {
		var cfg struct {
			Name SubConfig `envPrefix:""`
		}
		if err := envconfig.Read(&cfg, le); err == nil {
			t.Error("want error")
		}
	})
	t.Run("struct_with_both_env_and_prefix", func(t *testing.T) {
		var cfg struct {
			Name SubConfig `env:"AA" envPrefix:"BB"`
		}
		if err := envconfig.Read(&cfg, le); err == nil {
			t.Error("want error")
		}
	})
}

func TestInvalid(t *testing.T) {
	le := func(key string) (string, bool) {
		return "invalid", true
	}

	var cfg struct {
		Data struct {
			String  string `env:"STRING"`
			Int     int    `env:"INT"`
			Bool    bool   `env:"BOOL"`
			Default string `env:"DEFAULT" envDefault:"default"`
		} `env:"KEY"`
	}

	err := envconfig.Read(&cfg, le)
	if err == nil {
		t.Error("expected error")
	}
}

func TestReadRequired(t *testing.T) {
	le := func(key string) (string, bool) {
		return "", false
	}

	var cfg struct {
		Env string `env:"ENV" envRequired:"true"`
	}
	err := envconfig.Read(&cfg, le)
	if err == nil {
		t.Error("expected error")
	}
}

type Config struct {
	NotPopulated string `env:"-"`
	unexported   string

	String      string            `env:"A"`
	Int         int               `env:"B"`
	Int8        int8              `env:"C"`
	Int16       int16             `env:"D"`
	Int32       int32             `env:"E"`
	Int64       int64             `env:"F"`
	Uint        uint              `env:"G"`
	Uint8       uint8             `env:"H"`
	Uint16      uint16            `env:"I"`
	Uint32      uint32            `env:"J"`
	Uint64      uint64            `env:"K"`
	Float32     float32           `env:"L"`
	Float64     float64           `env:"M"`
	Bool        bool              `env:"N"`
	ArrString   [2]string         `env:"O"`
	SliceString []string          `env:"P"`
	Map         map[string]string `env:"Q"`
	PtrBool     *bool             `env:"R"`
	PtrPtrBool  **bool            `env:"S"`

	StringDefault string `env:"MISSING" envDefault:"Default Value"`
	MissingValue  string `env:"MISSING"`

	CustomTextUnmarshaler   CustomTextUnmarshaler   `env:"CUSTOM_TEXT"`
	CustomBinaryUnmarshaler CustomBinaryUnmarshaler `env:"CUSTOM_BINARY"`
	CustomJSONUnmarshaler   CustomJSONUnmarshaler   `env:"CUSTOM_JSON"`

	CustomTextUnmarshaler2   CustomTextUnmarshaler    `env:"CUSTOM"`
	CustomBinaryUnmarshaler2 CustomBinaryUnmarshaler  `env:"CUSTOM"`
	CustomJSONUnmarshaler2   CustomJSONUnmarshaler    `env:"CUSTOM"`
	Duration                 time.Duration            `env:"DURATION"`
	SliceDuration            []time.Duration          `env:"SDUR"`
	MapDuration              map[string]time.Duration `env:"MDUR"`

	URL *url.URL `env:"URL"`
}

type SubConfig struct {
	A string `env:"AA"`

	SubSub SubSubConfig `envPrefix:"SUB2"`
}

type SubSubConfig struct {
	A string `env:"FF" envRequired:"true"`
}

type CustomTextUnmarshaler struct {
	Value string `env:"VALUE"`
}

func (c *CustomTextUnmarshaler) UnmarshalText(text []byte) error {
	c.Value = "***" + string(text) + "***"
	return nil
}

type CustomBinaryUnmarshaler struct {
	Value string `env:"VALUE"`
}

func (c *CustomBinaryUnmarshaler) UnmarshalBinary(text []byte) error {
	c.Value = "***" + string(text) + "2***"
	return nil
}

type CustomJSONUnmarshaler struct {
	Value string `env:"VALUE"`
}

func (c *CustomJSONUnmarshaler) UnmarshalJSON(text []byte) error {
	c.Value = "***" + string(text) + "3***"
	return nil
}

func ptr[T any](t T) *T {
	return &t
}

func TestEmptySliceMapParsing(t *testing.T) {
	type Config struct {
		Strings []string       `env:"STRINGS"`
		IntMap  map[string]int `env:"INT_MAP"`
	}

	le := func(key string) (string, bool) {
		switch key {
		case "STRINGS":
			return "", true
		case "INT_MAP":
			return "", true
		}
		return "", false
	}

	var cfg Config
	err := envconfig.Read(&cfg, le)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if cfg.Strings != nil {
		t.Errorf("Expected nil slice, got %v", cfg.Strings)
	}
	if cfg.IntMap != nil {
		t.Errorf("Expected nil map, got %v", cfg.IntMap)
	}
}

func TestNestedStructPrefixHandling(t *testing.T) {
	type SubConfig struct {
		Port int `env:"PORT"`
	}

	type Config struct {
		Sub1 SubConfig `envPrefix:"APP1"`
		Sub2 SubConfig `envPrefix:"APP2"`
	}

	le := func(key string) (string, bool) {
		switch key {
		case "APP1_PORT":
			return "8080", true
		case "APP2_PORT":
			return "9090", true
		}
		return "", false
	}

	var cfg Config
	err := envconfig.Read(&cfg, le)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if cfg.Sub1.Port != 8080 || cfg.Sub2.Port != 9090 {
		t.Errorf("Incorrect port values: Sub1=%d, Sub2=%d", cfg.Sub1.Port, cfg.Sub2.Port)
	}
}

func TestEmptyStringDefaultValue(t *testing.T) {
	type Config struct {
		Value string `env:"VALUE" envDefault:"default"`
	}

	le := func(key string) (string, bool) {
		if key == "VALUE" {
			return "", true // Explicitly set to empty
		}
		return "", false
	}

	var cfg Config
	err := envconfig.Read(&cfg, le)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if cfg.Value != "" {
		t.Errorf("Expected empty string, got %q", cfg.Value)
	}
}

func TestEmptySlice(t *testing.T) {
	type Config struct {
		Value []string `env:"VALUE"`
	}

	le := func(key string) (string, bool) {
		if key == "VALUE" {
			return ",,,,", true
		}
		return "", false
	}

	var cfg Config
	err := envconfig.Read(&cfg, le)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(cfg.Value) != 5 {
		t.Errorf("Expected empty string, got %q", cfg.Value)
	}
}

// Credential represents a single user credential
type Credential struct {
	User string `env:"USER"`
	Pass string `env:"PASS"`
}

// Credentials is a list of credentials that implements EnvCollector
type Credentials []Credential

// CollectEnv implements the EnvCollector interface
func (c *Credentials) CollectEnv(prefix string, env envconfig.EnvGetter) error {
	// Read the list of IDs from the prefix key itself (e.g., CREDS=0,1,2,3)
	var ids []string
	if err := env.ReadValue(prefix, &ids); err != nil {
		return err
	}

	for _, id := range ids {
		var cred Credential
		// Read each credential struct using the full tag support
		if err := env.Read(prefix+"_"+id, &cred); err != nil {
			return err
		}
		*c = append(*c, cred)
	}
	return nil
}

type ConfigWithCollector struct {
	Credentials Credentials `envPrefix:"CREDS"`
}

func TestEnvCollector(t *testing.T) {
	le := func(key string) (string, bool) {
		switch key {
		case "CREDS":
			return "0,1,2,4", true

		case "CREDS_0_USER":
			return "user0", true
		case "CREDS_0_PASS":
			return "pass0", true

		case "CREDS_1_USER":
			return "user1", true
		case "CREDS_1_PASS":
			return "pass1", true

		case "CREDS_2_USER":
			return "user2", true
		case "CREDS_2_PASS":
			return "pass2", true

		case "CREDS_4_USER":
			return "user4", true
		case "CREDS_4_PASS":
			return "pass4", true
		}
		return "", false
	}

	var cfg ConfigWithCollector
	err := envconfig.Read(&cfg, le)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(cfg.Credentials) != 4 {
		t.Errorf("Expected 4 credentials, got %d", len(cfg.Credentials))
	}

	expected := []Credential{
		{User: "user0", Pass: "pass0"},
		{User: "user1", Pass: "pass1"},
		{User: "user2", Pass: "pass2"},
		{User: "user4", Pass: "pass4"},
	}

	for i, cred := range cfg.Credentials {
		if cred.User != expected[i].User || cred.Pass != expected[i].Pass {
			t.Errorf("Credential %d: expected %+v, got %+v", i, expected[i], cred)
		}
	}
}

func TestSetValueErrors(t *testing.T) {
	tests := []struct {
		name    string
		sut     func(le envconfig.LookupEnv) error
		envVal  string
		wantErr string
	}{
		{
			name: "invalid_bool",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V bool `env:"V"`
				}{}, le)
			},
			envVal:  "notabool",
			wantErr: "invalid syntax",
		},
		{
			name: "invalid_int",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V int `env:"V"`
				}{}, le)
			},
			envVal:  "notanint",
			wantErr: "invalid syntax",
		},
		{
			name: "int_overflow",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V int8 `env:"V"`
				}{}, le)
			},
			envVal:  "999",
			wantErr: "out of range",
		},
		{
			name: "invalid_uint",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V uint `env:"V"`
				}{}, le)
			},
			envVal:  "notauint",
			wantErr: "invalid syntax",
		},
		{
			name: "uint_negative",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V uint `env:"V"`
				}{}, le)
			},
			envVal:  "-1",
			wantErr: "invalid syntax",
		},
		{
			name: "invalid_float",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V float64 `env:"V"`
				}{}, le)
			},
			envVal:  "notafloat",
			wantErr: "invalid syntax",
		},
		{
			name: "invalid_duration",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V time.Duration `env:"V"`
				}{}, le)
			},
			envVal:  "notaduration",
			wantErr: "invalid duration",
		},
		{
			name: "array_too_short",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V [3]string `env:"V"`
				}{}, le)
			},
			envVal:  "a,b",
			wantErr: "array needs 3 elements, got 2",
		},
		{
			name: "array_invalid_element",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V [3]int `env:"V"`
				}{}, le)
			},
			envVal:  "1,notint,3",
			wantErr: "invalid syntax",
		},
		{
			name: "map_invalid_format",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V map[string]string `env:"V"`
				}{}, le)
			},
			envVal:  "keyonly",
			wantErr: "invalid map value",
		},
		{
			name: "map_invalid_value",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V map[string]int `env:"V"`
				}{}, le)
			},
			envVal:  "key=notint",
			wantErr: "invalid syntax",
		},
		{
			name: "map_invalid_key",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V map[int]int `env:"V"`
				}{}, le)
			},
			envVal:  "key=12",
			wantErr: "invalid syntax",
		},
		{
			name: "slice_invalid_element",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V []int `env:"V"`
				}{}, le)
			},
			envVal:  "1,2,notint",
			wantErr: "invalid syntax",
		},
		{
			name: "unsupported_type",
			sut: func(le envconfig.LookupEnv) error {
				return envconfig.Read(&struct {
					V chan int `env:"V"`
				}{}, le)
			},
			envVal:  "anything",
			wantErr: "unsupported type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			le := func(key string) (string, bool) {
				if key == "V" {
					return tt.envVal, true
				}
				return "", false
			}

			err := tt.sut(le)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			t.Log(err)
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestEnvCollectorErrors(t *testing.T) {
	le := func(key string) (string, bool) {
		return "", false
	}

	t.Run("collector_with_env_tag", func(t *testing.T) {
		var cfg struct {
			Creds Credentials `env:"CREDS"`
		}
		err := envconfig.Read(&cfg, le)
		if err.Error() != `envconfig: "Creds" implements EnvCollector, use "envPrefix" instead of env` {
			t.Error("expected error for collector with env tag")
		}
	})

	t.Run("collector_without_tag", func(t *testing.T) {
		var cfg struct {
			Creds Credentials
		}
		err := envconfig.Read(&cfg, le)
		if err.Error() != "envconfig: field \"Creds\" does not have \"env\" or \"envPrefix\" tags. Ignore it explicitly with `env:\"-\"` or embed to treat it flat" {
			t.Error("expected error for collector without tag")
		}
	})

	t.Run("collector_with_empty_prefix", func(t *testing.T) {
		var cfg struct {
			Creds Credentials `envPrefix:""`
		}
		err := envconfig.Read(&cfg, le)
		if err.Error() != `envconfig: "Creds" implements EnvCollector with empty "envPrefix"` {
			t.Error("expected error for collector with empty prefix")
		}
	})
}
