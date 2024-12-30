package env

import (
	"os"
	"strconv"
	"strings"
)

type Environment string

const (
	Production Environment = "production"
	PreRelease Environment = "pre-release"
)

// StringFromEnv returns the env variable for the given key
// and falls back to the given defaultValue if not set
func StringFromEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimSpace(v)
	}
	return defaultValue
}

// ParseIntFromEnv helper function to parse a number from an environment variable. Returns a
// default if env is not set, is not parseable to a number, exceeds max (if
// max is greater than 0) or is less than min.
func ParseIntFromEnv(env string, defaultValue, min, max int) int {
	str := os.Getenv(env)
	if str == "" {
		return defaultValue
	}
	num, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	if num < min {
		return defaultValue
	}
	if num > max {
		return defaultValue
	}
	return num
}

// ParseFloatFromEnv helper function to parse a number from an environment variable. Returns a
// default if env is not set, is not parseable to a number, exceeds max (if
// max is greater than 0) or is less than min.
func ParseFloatFromEnv(env string, defaultValue, min, max float64) float64 {
	str := os.Getenv(env)
	if str == "" {
		return defaultValue
	}
	num, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return defaultValue
	}
	if num < min {
		return defaultValue
	}
	if num > max {
		return defaultValue
	}
	return num
}

// ParseBoolFromEnv retrieves a boolean value from given environment envVar.
// Returns default value if envVar is not set.
func ParseBoolFromEnv(envVar string, defaultValue bool) bool {
	if val := os.Getenv(envVar); val != "" {
		val = strings.TrimSpace(strings.ToLower(val))
		switch val {
		case "true":
			return true
		case "1":
			return true
		case "false":
			return false
		case "0":
			return false
		}
	}

	return defaultValue
}

func IsProductionMode(env string) bool {
	return Environment(env) == Production
}
