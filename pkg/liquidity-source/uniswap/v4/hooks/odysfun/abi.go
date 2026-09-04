package odysfun

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var odysHookABI abi.ABI

func init() {
	var err error
	odysHookABI, err = abi.JSON(bytes.NewReader(odysHookABIJson))
	if err != nil {
		panic(err)
	}
}
