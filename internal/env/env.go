package env

// Package env provides helpers for reading and writing environment variables
// using the charmed application "APP_" prefix convention.

import "os"

// Prefix is automatically prepended to environment variable names.
// Charmed applications expose their configuration with an "APP_" prefix,
// so callers can reference the unprefixed name (e.g. "GH_TOKEN").
const Prefix = "APP_"

// Get returns the value of the environment variable named by key,
// automatically prepending the charmed application "APP_" prefix.
// For example, Get("GH_TOKEN") looks up "APP_GH_TOKEN".
func GetGoEnv(key string) string {
	return os.Getenv(Prefix + key)
}

// Set sets the environment variable named by key, automatically
// prepending the charmed application "APP_" prefix.
// For example, Set("GH_TOKEN", v) sets "APP_GH_TOKEN".
func SetGoEnv(key, value string) error {
	return os.Setenv(Prefix+key, value)
}
