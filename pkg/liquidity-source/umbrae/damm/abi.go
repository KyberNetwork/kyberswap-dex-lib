package umbraedamm

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var pairABI abi.ABI

var PairABI abi.ABI

func init() {
	var err error
	pairABI, err = abi.JSON(bytes.NewReader(pairABIJson))
	if err != nil {
		panic(err)
	}
	PairABI = pairABI
}
