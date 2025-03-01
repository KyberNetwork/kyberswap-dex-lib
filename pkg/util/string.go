package util

import (
	"math/big"
	"strconv"
	"strings"
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
	return strings.Join(s, ":")
}

// FormatKey returns a key from a list of strings.
func FormatKey(sep string, args ...string) string {
	return strings.Join(args, sep)
}

// MaxBytesToString returns a string up to maxBytes of bytes.
func MaxBytesToString(b []byte, maxBytes int) string {
	if len(b) > maxBytes {
		return string(b[:maxBytes]) + ".."
	}
	return string(b)
}
