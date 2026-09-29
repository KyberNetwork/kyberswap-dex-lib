package cashcat

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var cashCatHookV2ABI abi.ABI

func init() {
	var err error
	cashCatHookV2ABI, err = abi.JSON(bytes.NewReader(cashCatHookV2ABIJson))
	if err != nil {
		panic(err)
	}
}
