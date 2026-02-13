package envconfig

import (
	"os"
	"testing"
)

func TestCustomLookup(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		const keyEmpty = "__CUSTOM_LOOKUP_EMPTY_KEY__"
		t.Setenv(keyEmpty, "")

		_, ignoreExists := IgnoreEmptyEnvLookup(keyEmpty)
		_, osExists := os.LookupEnv(keyEmpty)

		if ignoreExists == osExists {
			t.Fatalf("Expected %q to not exists got %v != %v", keyEmpty, ignoreExists, osExists)
		}
	})

	t.Run("non_empty", func(t *testing.T) {
		const keyNotEmpty = "__CUSTOM_LOOKUP_KEY__"
		t.Setenv(keyNotEmpty, "__value__")

		ignoreVal, ignoreExists := IgnoreEmptyEnvLookup(keyNotEmpty)
		osVal, osExists := os.LookupEnv(keyNotEmpty)

		if ignoreExists != osExists {
			t.Fatalf("Expected %q to not exists got %v != %v", keyNotEmpty, ignoreExists, osExists)
		}

		if ignoreVal != osVal {
			t.Fatalf("Expected %q to have the same values: %v != %v", keyNotEmpty, ignoreVal, osVal)
		}
	})
}
