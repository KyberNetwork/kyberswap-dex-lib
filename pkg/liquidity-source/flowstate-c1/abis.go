package flowstatec1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var marketABI abi.ABI

func init() {
	var err error
	marketABI, err = abi.JSON(bytes.NewReader(marketABIJson))
	if err != nil {
		panic(err)
	}
}
