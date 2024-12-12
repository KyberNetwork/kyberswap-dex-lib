package virtualfun

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	erc20ABI   abi.ABI
	pairABI    abi.ABI
	factoryABI abi.ABI
	bondingABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&erc20ABI, erc20ABIJson,
		},
		{
			&pairABI, pairABIJson,
		},
		{
			&factoryABI, factoryABIJson,
		},
		{
			&bondingABI, bodingABIJson,
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
