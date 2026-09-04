package odysfun

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	odysHookABI       abi.ABI
	odysHookLegacyABI abi.ABI
)

func init() {
	var err error
	if odysHookABI, err = abi.JSON(bytes.NewReader(odysHookABIJson)); err != nil {
		panic(err)
	}
	if odysHookLegacyABI, err = abi.JSON(bytes.NewReader(odysHookLegacyABIJson)); err != nil {
		panic(err)
	}
}
