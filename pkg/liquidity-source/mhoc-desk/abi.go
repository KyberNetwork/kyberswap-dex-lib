package mhocdesk

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	propAMMABI  abi.ABI
	registryABI abi.ABI
)

func init() {
	var err error
	if propAMMABI, err = abi.JSON(bytes.NewReader(propAMMABIData)); err != nil {
		panic(err)
	}
	if registryABI, err = abi.JSON(bytes.NewReader(registryABIData)); err != nil {
		panic(err)
	}
}
