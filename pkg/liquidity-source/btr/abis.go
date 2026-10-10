package btr

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Minimal ABI surface used by the adapter. Source of truth is the embedded
// abis/Pool.json, a pinned copy of the deployed BTR Pool ABI.
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
