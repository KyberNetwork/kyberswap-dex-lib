package ponsv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var hookABI abi.ABI

func init() {
	var err error
	hookABI, err = abi.JSON(bytes.NewReader(hookABIJson))
	if err != nil {
		panic(err)
	}
}

// HookABI returns the launches getter ABI for compatible hook adapters.
func HookABI() abi.ABI { return hookABI }
