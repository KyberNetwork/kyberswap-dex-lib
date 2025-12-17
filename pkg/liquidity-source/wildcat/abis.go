package wildcat

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	factoryABI   abi.ABI
	pairABI      abi.ABI
	erc20ABI     abi.ABI
	multicallABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&factoryABI, factoryABIData},
		{&pairABI, pairABIData},
		{&erc20ABI, erc20ABIData},
		{&multicallABI, multicallABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
