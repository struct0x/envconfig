package envconfig

import (
	"os"
)

// IgnoreEmptyEnvLookup wraps os.LookupEnv but treats empty values as unset.
// If the variable is present but "", it returns ok == false.
func IgnoreEmptyEnvLookup(key string) (string, bool) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}
