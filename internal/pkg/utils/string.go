package utils

import (
	"math/big"
	"strconv"
	"strings"
)

const (
	EmptyString      = ""
	defaultSeparator = ":"
)

func Join(args ...interface{}) string {
	s := make([]string, len(args))
	for i, v := range args {
		switch v := v.(type) {
		case string:
			s[i] = v
		case int64:
			s[i] = strconv.FormatInt(v, 10)
		case uint8:
			s[i] = strconv.FormatInt(int64(v), 10)
		case uint64:
			s[i] = strconv.FormatUint(v, 10)
		case float64:
			s[i] = strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			if v {
				s[i] = "1"
			} else {
				s[i] = "0"
			}
		case *big.Int:
			if v != nil {
				s[i] = v.String()
			} else {
				s[i] = "0"
			}
		case *big.Rat:
			if v != nil {
				s[i] = v.FloatString(9)
			} else {
				s[i] = "0"
			}
		default:
			panic("Invalid type specified for conversion")
		}
	}
	return strings.Join(s, defaultSeparator)
}

func IsEmptyString(str string) bool {
	return str == EmptyString
}

// CompareStringSlices
// return -1 if len(a) < len(b) or a < b in alphabet order,
// return 0 if equal
// else return 1
// Compare two string slices lexicographically (alphabetically)
func CompareStringSlices(a, b []string) int {
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return compareStrings(a[i], b[i])
		}
	}

	return 0
}

// Compare two strings lexicographically (alphabetically)
func compareStrings(a, b string) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}
