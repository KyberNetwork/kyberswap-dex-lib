package fxdx

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	vaultABI          abi.ABI
	erc20ABI          abi.ABI
	vaultPriceFeedABI abi.ABI
	chainlinkABI      abi.ABI
	pancakePairABI    abi.ABI
	fastPriceFeedABI  abi.ABI
	priceFeedABI      abi.ABI
	feeUtilsABI       abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&vaultABI, vaultJson},
		{&erc20ABI, erc20Json},
		{&vaultPriceFeedABI, vaultPriceFeedJson},
		{&chainlinkABI, chainlinkFlagsJson},
		{&pancakePairABI, pancakePairJson},
		{&fastPriceFeedABI, fastPriceFeedJson},
		{&priceFeedABI, priceFeedJson},
		{&feeUtilsABI, feeUtilsV2Json},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
