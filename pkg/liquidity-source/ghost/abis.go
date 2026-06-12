package ghost

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	feeABI        abi.ABI
	routerABI     abi.ABI
	routingFeeABI abi.ABI
)

func init() {
	var err error
	feeABI, err = abi.JSON(bytes.NewReader(feeABIData))
	if err != nil {
		panic(err)
	}
	routerABI, err = abi.JSON(bytes.NewReader(routerABIData))
	if err != nil {
		panic(err)
	}
	routingFeeABI, err = abi.JSON(bytes.NewReader(routingFeeABIData))
	if err != nil {
		panic(err)
	}
}
