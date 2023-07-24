package abi

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// GenMethodID generates method id from rawName and inputs types
// implemented following go-ethereum method https://github.com/ethereum/go-ethereum/blob/master/accounts/abi/method.go#L117-L118
func GenMethodID(rawName string, types []string) (id [4]byte) {
	sig := fmt.Sprintf("%v(%v)", rawName, strings.Join(types, ","))

	copy(id[:], crypto.Keccak256([]byte(sig)))

	return
}
