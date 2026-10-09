package btr

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Minimal ABI surface used by the adapter. Source of truth is the embedded
// dex-evm/abi/Pool.json (repo pin), refreshed by `go:generate`, never
// hand-copied.
var poolABI abi.ABI

func init() {
	var err error
	poolABI, err = abi.JSON(bytes.NewReader(poolABIJSONBytes))
	if err != nil {
		panic("btr: bad pool ABI: " + err.Error())
	}
}

func mustABI(s string) abi.ABI {
	a, err := abi.JSON(strings.NewReader(s))
	if err != nil {
		panic("btr: bad inline ABI: " + err.Error())
	}
	return a
}
