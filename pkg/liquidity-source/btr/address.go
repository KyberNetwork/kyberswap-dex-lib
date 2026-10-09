package btr

import "strings"

// lowerAddr normalizes an address for map keys and token comparisons.
func lowerAddr(a string) string { return strings.ToLower(a) }
