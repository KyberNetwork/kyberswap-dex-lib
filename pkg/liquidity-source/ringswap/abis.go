package ringswap

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	UniswapV2PairABI    abi.ABI
	uniswapV2FactoryABI abi.ABI
	fewWrappedTokenABI  abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&UniswapV2PairABI, pairABIJson,
		},
		{
			&uniswapV2FactoryABI, factoryABIJson,
		},
		{
			&fewWrappedTokenABI, fewWrappedTokenABIJson,
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
