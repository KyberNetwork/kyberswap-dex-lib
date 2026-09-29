package fables

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var fablesHookABI abi.ABI

func init() {
	var err error
	fablesHookABI, err = abi.JSON(bytes.NewReader(fablesHookABIJson))
	if err != nil {
		panic(err)
	}
}
