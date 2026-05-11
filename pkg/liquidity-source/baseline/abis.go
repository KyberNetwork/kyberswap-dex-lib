package baseline

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var relayABI abi.ABI

func init() {
	var err error
	relayABI, err = abi.JSON(bytes.NewReader(relayABIJson))
	if err != nil {
		panic(err)
	}
}
