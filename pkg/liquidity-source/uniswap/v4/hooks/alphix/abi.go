package alphix

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var alphixHookABI abi.ABI

func init() {
	var err error
	alphixHookABI, err = abi.JSON(bytes.NewReader(alphixHookABIJson))
	if err != nil {
		panic(err)
	}
}
