package dexLite

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Parsed ABI instances
var (
	fluidDexLiteABI abi.ABI
	erc20ABI        abi.ABI
)

func init() {
	var err error

	fluidDexLiteABI, err = abi.JSON(strings.NewReader(string(fluidDexLiteABIBytes)))
	if err != nil {
		panic(err)
	}

	erc20ABI, err = abi.JSON(strings.NewReader(string(erc20ABIBytes)))
	if err != nil {
		panic(err)
	}
}