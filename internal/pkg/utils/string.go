package utils

import (
	"math/big"
	"strconv"
	"strings"
)

const (
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

func Split(data string) []string {
	return strings.Split(data, defaultSeparator)
}

func CoalesceEmptyString(values ...string) string {
	for _, v := range values {
		if v != EmptyString {
			return v
		}
	}
	return EmptyString
}

func IsEmptyString(str string) bool {
	return str == EmptyString
}
