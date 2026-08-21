package synthereum

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	multiLpPoolABI abi.ABI
	vaultABI       abi.ABI
	finderABI      abi.ABI
	priceFeedABI   abi.ABI
	wrapperABI     abi.ABI
	registryABI    abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&multiLpPoolABI, multiLpPoolABIData},
		{&vaultABI, vaultABIData},
		{&finderABI, finderABIData},
		{&priceFeedABI, priceFeedABIData},
		{&wrapperABI, wrapperABIData},
		{&registryABI, registryABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
