package poolsidev1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolsideV1PairABI        abi.ABI
	poolsideV1FactoryABI     abi.ABI
	poolsideV1ButtonTokenABI abi.ABI
	erc20ABI                 abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&poolsideV1PairABI, pairABIJson,
		},
		{
			&poolsideV1FactoryABI, factoryABIJson,
		},
		{
			&poolsideV1ButtonTokenABI, buttonTokenABIJson,
		},
		{
			&erc20ABI, erc20ABIJson,
		},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
