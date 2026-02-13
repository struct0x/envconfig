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
		case "T":
			return "bytes", true
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
		PtrBool: ptr(true),
		NilPtr:  nil,
		NilPtrStruct: ptr(struct {
			A string  `env:"A"`
			B *string `env:"MISSING"`
		}{
			A: "hello",
			B: nil,
		}),
		PtrPtrBool:    ptr(ptr(true)),
		Bytes:         []byte("bytes"),
		StringDefault: "Default Value",
		MissingValue:  "",
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
		CustomString: "***custom_text***",
		Duration:     time.Hour,
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

	t.Log(cfg)
	t.Log(want)
}

func TestReadAdvanced(t *testing.T) {
	le := func(key string) (string, bool) {
		switch key {
		case "ENV":
			return "ENV", true
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
			Sub2 SubConfig
		}
		if err := envconfig.Read(&cfg, le); err != nil {
			t.Error(err)
		}

		want := struct {
			SubConfig
			Sub2 SubConfig
		}{
			SubConfig: SubConfig{
				A: "AA",
				SubSub: SubSubConfig{
					A: "aaa",
				},
			},

			Sub2: SubConfig{
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

	t.Run("struct_with_empty_env", func(t *testing.T) {
		var cfg struct {
			Name SubConfig `env:""`
		}
		err := envconfig.Read(&cfg, le)
		assertErr(t, err, `envconfig: tag "env" can't be empty: "Name"`)
	})

	t.Run("embedded_struct_with_env", func(t *testing.T) {
		var cfg struct {
			SubConfig `env:"ENV"`
		}
		err := envconfig.Read(&cfg, le)
		assertErr(t, err, `envconfig: field "SubConfig" is a struct with "env" tag but does not implement encoding.TextUnmarshaler / encoding.BinaryUnmarshaler / json.Unmarshaler`)
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
		err := envconfig.Read(&cfg, le)
		assertErr(t, err, `envconfig: tag "envPrefix" can't be empty: "SubConfig"`)
	})

	t.Run("struct_with_empty_prefix", func(t *testing.T) {
		var cfg struct {
			Name SubConfig `envPrefix:""`
		}
		err := envconfig.Read(&cfg, le)
		assertErr(t, err, `envconfig: tag "envPrefix" can't be empty: "Name"`)
	})
	t.Run("struct_with_both_env_and_prefix", func(t *testing.T) {
		var cfg struct {
			Name SubConfig `env:"AA" envPrefix:"BB"`
		}
		err := envconfig.Read(&cfg, le)
		assertErr(t, err, `envconfig: both "env"  and "envPrefix" does not make sense. If a field is a struct pick "envPrefix" if you want to populate it using composite env keys, use "env" if you implement encoding.TextUnmarshaler / encoding.BinaryUnmarshaler / json.Unmarshaler, or remove tags to treat is flat`)
	})
}

func TestInvalid(t *testing.T) {
	le := func(key string) (string, bool) {
		return "invalid", true
	}

	var cfg struct {
		Data struct {
			Int int `env:"INT"`
		}
	}

	err := envconfig.Read(&cfg, le)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != `envconfig: field "Int" failed to populate: strconv.ParseInt: parsing "invalid": invalid syntax` {
		t.Fatalf("Wrong error: %v", err)
	}
	t.Log(err)
}

func TestPrimitiveNoEnvTag(t *testing.T) {
	le := func(key string) (string, bool) {
		return "invalid", true
	}

	var cfg struct {
		Data struct {
			String string
		}
	}

	err := envconfig.Read(&cfg, le)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != `envconfig: field "String" does not have "env" tag` {
		t.Fatalf("Wrong error: %v", err)
	}
	t.Log(err)
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

	String        string            `env:"A"`
	Int           int               `env:"B"`
	Int8          int8              `env:"C"`
	Int16         int16             `env:"D"`
	Int32         int32             `env:"E"`
	Int64         int64             `env:"F"`
	Uint          uint              `env:"G"`
	Uint8         uint8             `env:"H"`
	Uint16        uint16            `env:"I"`
	Uint32        uint32            `env:"J"`
	Uint64        uint64            `env:"K"`
	Float32       float32           `env:"L"`
	Float64       float64           `env:"M"`
	Bool          bool              `env:"N"`
	ArrString     [2]string         `env:"O"`
	SliceString   []string          `env:"P"`
	Map           map[string]string `env:"Q"`
	PtrBool       *bool             `env:"R"`
	NilPtrIgnored *int              `env:"-"`
	NilPtr        *string           `env:"MISSING"`
	NilPtrStruct  *struct {
		A string  `env:"A"`
		B *string `env:"MISSING"`
	}
	PtrPtrBool **bool `env:"S"`
	Bytes      []byte `env:"T"`

	StringDefault string `env:"MISSING" envDefault:"Default Value"`
	MissingValue  string `env:"MISSING"`

	CustomTextUnmarshaler   CustomTextUnmarshaler   `env:"CUSTOM_TEXT"`
	CustomBinaryUnmarshaler CustomBinaryUnmarshaler `env:"CUSTOM_BINARY"`
	CustomJSONUnmarshaler   CustomJSONUnmarshaler   `env:"CUSTOM_JSON"`

	CustomString CustomString `env:"CUSTOM_TEXT"`

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

type CustomString string

func (c *CustomString) UnmarshalJSON(text []byte) error {
	*c = CustomString("***" + string(text) + "***")
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
		Sub1      SubConfig `envPrefix:"APP1"`
		SubConfig `envPrefix:"APP2"`
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

	if cfg.Sub1.Port != 8080 || cfg.Port != 9090 {
		t.Errorf("Incorrect port values: Sub1=%d, Sub2=%d", cfg.Sub1.Port, cfg.Port)
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
func (c *Credentials) CollectEnv(env envconfig.EnvGetter) error {
	// Read the list of IDs from the prefix key itself (e.g., CREDS=0,1,2,3)
	var ids []string
	if err := env.ReadValue("CREDS", &ids); err != nil {
		return err
	}

	for _, id := range ids {
		var cred Credential
		// Read each credential struct using the full tag support
		if err := env.ReadIntoStruct("CREDS_"+id, &cred); err != nil {
			return err
		}
		*c = append(*c, cred)
	}
	return nil
}

type ConfigWithCollector struct {
	Credentials Credentials
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

// badCollectorNonPointer tests EnvGetter.Read with non-pointer target
type badCollectorNonPointer struct{}

func (b *badCollectorNonPointer) CollectEnv(env envconfig.EnvGetter) error {
	var target struct{ Name string }
	return env.ReadIntoStruct("", target) // non-pointer - should error
}

// badCollectorNonPointer tests EnvGetter.Read with non-pointer target
type badCollectorNotPointerReceiver struct{}

func (b badCollectorNotPointerReceiver) CollectEnv(env envconfig.EnvGetter) error {
	var target struct{ Name string }
	return env.ReadIntoStruct("", target) // non-pointer - should error
}

type badCollectorNonStruct struct{}

func (b *badCollectorNonStruct) CollectEnv(env envconfig.EnvGetter) error {
	var target string
	return env.ReadIntoStruct("", &target)
}

func TestEnvGetterReadValidation(t *testing.T) {
	le := func(key string) (string, bool) {
		return "", false
	}

	t.Run("non_pointer_receiver_target", func(t *testing.T) {
		var cfg struct {
			Bad badCollectorNotPointerReceiver
		}
		err := envconfig.Read(&cfg, le)
		assertErr(t, err, "envconfig: field \"Bad\" implements EnvCollector but not for a pointer receiver")
	})

	t.Run("non_pointer_target", func(t *testing.T) {
		var cfg struct {
			Bad badCollectorNonPointer
		}
		err := envconfig.Read(&cfg, le)
		assertErr(t, err, "envconfig: \"Bad\" CollectEnv failed: envconfig: Read target must be a pointer, got \"struct\"")
	})

	t.Run("non_struct_target", func(t *testing.T) {
		var cfg struct {
			Bad badCollectorNonStruct
		}
		err := envconfig.Read(&cfg, le)
		assertErr(t, err, "envconfig: \"Bad\" CollectEnv failed: envconfig: Read target must be a pointer to struct, got pointer to \"string\"")
	})
}

func TestNilPointer(t *testing.T) {
	err := envconfig.Read((*Config)(nil))

	assertErr(t, err, "envconfig: nil holder")
}

func TestPointerStructDeallocation(t *testing.T) {
	type Sub struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	t.Run("nil_when_no_env_vars_set", func(t *testing.T) {
		type Config struct {
			DB *Sub `envPrefix:"DB"`
		}

		le := func(key string) (string, bool) {
			return "", false
		}

		var cfg Config
		if err := envconfig.Read(&cfg, le); err != nil {
			t.Fatal(err)
		}

		if cfg.DB != nil {
			t.Errorf("expected DB to be nil, got %+v", cfg.DB)
		}
	})

	t.Run("allocated_when_any_env_var_set", func(t *testing.T) {
		type Config struct {
			DB *Sub `envPrefix:"DB"`
		}

		le := func(key string) (string, bool) {
			if key == "DB_HOST" {
				return "localhost", true
			}
			return "", false
		}

		var cfg Config
		if err := envconfig.Read(&cfg, le); err != nil {
			t.Fatal(err)
		}

		if cfg.DB == nil {
			t.Fatal("expected DB to be allocated")
		}
		if cfg.DB.Host != "localhost" {
			t.Errorf("expected Host=localhost, got %q", cfg.DB.Host)
		}
		if cfg.DB.Port != 0 {
			t.Errorf("expected Port=0, got %d", cfg.DB.Port)
		}
	})

	t.Run("nested_pointer_struct_nil", func(t *testing.T) {
		type Inner struct {
			Cert string `env:"CERT"`
		}
		type Outer struct {
			Host string `env:"HOST"`
			TLS  *Inner `envPrefix:"TLS"`
		}
		type Config struct {
			DB *Outer `envPrefix:"DB"`
		}

		le := func(key string) (string, bool) {
			if key == "DB_HOST" {
				return "localhost", true
			}
			return "", false
		}

		var cfg Config
		if err := envconfig.Read(&cfg, le); err != nil {
			t.Fatal(err)
		}

		if cfg.DB == nil {
			t.Fatal("expected DB to be allocated")
		}
		if cfg.DB.TLS != nil {
			t.Errorf("expected TLS to be nil, got %+v", cfg.DB.TLS)
		}
	})
}

func TestUnmarshalerInCompositeTypes(t *testing.T) {
	t.Run("slice_of_text_unmarshaler", func(t *testing.T) {
		type Config struct {
			Values []CustomString `env:"VALUES"`
		}

		le := func(key string) (string, bool) {
			if key == "VALUES" {
				return "hello,world", true
			}
			return "", false
		}

		var cfg Config
		if err := envconfig.Read(&cfg, le); err != nil {
			t.Fatal(err)
		}

		if len(cfg.Values) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(cfg.Values))
		}
		if cfg.Values[0] != "***hello***" {
			t.Errorf("expected ***hello***, got %q", cfg.Values[0])
		}
		if cfg.Values[1] != "***world***" {
			t.Errorf("expected ***world***, got %q", cfg.Values[1])
		}
	})

	t.Run("map_value_text_unmarshaler", func(t *testing.T) {
		type Config struct {
			Values map[string]CustomString `env:"VALUES"`
		}

		le := func(key string) (string, bool) {
			if key == "VALUES" {
				return "a=hello,b=world", true
			}
			return "", false
		}

		var cfg Config
		if err := envconfig.Read(&cfg, le); err != nil {
			t.Fatal(err)
		}

		if len(cfg.Values) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(cfg.Values))
		}
		if cfg.Values["a"] != "***hello***" {
			t.Errorf("expected ***hello***, got %q", cfg.Values["a"])
		}
		if cfg.Values["b"] != "***world***" {
			t.Errorf("expected ***world***, got %q", cfg.Values["b"])
		}
	})

	t.Run("array_of_text_unmarshaler", func(t *testing.T) {
		type Config struct {
			Values [2]CustomString `env:"VALUES"`
		}

		le := func(key string) (string, bool) {
			if key == "VALUES" {
				return "hello,world", true
			}
			return "", false
		}

		var cfg Config
		if err := envconfig.Read(&cfg, le); err != nil {
			t.Fatal(err)
		}

		if cfg.Values[0] != "***hello***" {
			t.Errorf("expected ***hello***, got %q", cfg.Values[0])
		}
		if cfg.Values[1] != "***world***" {
			t.Errorf("expected ***world***, got %q", cfg.Values[1])
		}
	})
}

func assertErr(t *testing.T, err error, exp string) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected error got nil")
	}

	if err.Error() != exp {
		t.Fatalf("Expected %q got %q", exp, err.Error())
	}
}
