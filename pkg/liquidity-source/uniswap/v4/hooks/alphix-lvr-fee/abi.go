package alphixlvrfee

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
