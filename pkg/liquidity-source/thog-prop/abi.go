package thogprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var thogAMMABI abi.ABI

func init() {
	var err error
	thogAMMABI, err = abi.JSON(bytes.NewReader(thogAMMBytes))
	if err != nil {
		panic(err)
	}
}
