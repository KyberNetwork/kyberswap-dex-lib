package daikonprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var daikonABI abi.ABI

func init() {
	var err error
	if daikonABI, err = abi.JSON(bytes.NewReader(daikonABIJson)); err != nil {
		panic(err)
	}
}
