package abi

import (
	"bytes"
	"io"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// MustParseABI parses ABI JSON, panicking if it is malformed.
func MustParseABI[T ~string | ~[]byte](data T) abi.ABI {
	parsed, err := abi.JSON(abiReader(data))
	if err != nil {
		panic(err)
	}

	return parsed
}

// abiReader avoids copying the common []byte case; anything else goes through a
// string conversion, which also covers named types over string or []byte.
func abiReader[T ~string | ~[]byte](data T) io.Reader {
	if b, ok := any(data).([]byte); ok {
		return bytes.NewReader(b)
	}

	return strings.NewReader(string(data))
}
