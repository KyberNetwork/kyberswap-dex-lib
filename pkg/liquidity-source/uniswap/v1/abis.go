package uniswapv1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	erc20ABI           abi.ABI
	uniswapExchangeABI abi.ABI
	uniswapFactoryABI  abi.ABI
	multicallABI       abi.ABI
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
			&uniswapExchangeABI, exchangeABIJson,
		},
		{
			&uniswapFactoryABI, factoryABIJson,
		},
		{
			&multicallABI, multicallABIJson,
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
