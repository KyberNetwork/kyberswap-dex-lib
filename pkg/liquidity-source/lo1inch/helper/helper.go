package helper

import (
	"encoding/hex"
	"strings"
)

func isHexBytes(s string) bool {
	s = strings.TrimPrefix(s, "0x")
	if len(s)%2 != 0 {
		return false
	}

	_, err := hex.DecodeString(s)
	return err == nil
}

func isHexString(s string) bool {
	if s == ZX {
		return true
	}

	return strings.HasPrefix(s, "0x") && isHexBytes(s)
}
