package main

import "os"

// DefaultEnv retrieves the value of the environment variable named by the key.
// If the variable is not present in the environment, defaultValue is returned.
func DefaultEnv(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		value = defaultValue
	}
	return value
}
