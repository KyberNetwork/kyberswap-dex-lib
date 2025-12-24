package warpx

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	pairABI    abi.ABI
	factoryABI abi.ABI
)

func init() {
	var err error

	pairABI, err = abi.JSON(bytes.NewReader(pairABIJson))
	if err != nil {
		panic(err)
	}

	factoryABI, err = abi.JSON(bytes.NewReader(factoryABIJson))
	if err != nil {
		panic(err)
	}
}
