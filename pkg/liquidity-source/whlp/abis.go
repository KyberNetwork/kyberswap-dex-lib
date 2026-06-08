package whlp

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var accountantABI abi.ABI

func init() {
	var err error
	accountantABI, err = abi.JSON(bytes.NewReader(accountantABIJson))
	if err != nil {
		panic(err)
	}
}
