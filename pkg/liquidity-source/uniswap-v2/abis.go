package uniswapv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	uniswapV2PairABI      abi.ABI
	uniswapV2FactoryABI   abi.ABI
	meerkatPairABI        abi.ABI
	mdexFactoryABI        abi.ABI
	shibaswapPairABI      abi.ABI
	croDefiSwapFactoryABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&uniswapV2PairABI, pairABIJson,
		},
		{
			&uniswapV2FactoryABI, factoryABIJson,
		},
		{
			&meerkatPairABI, meerkatPairABIJson,
		},
		{
			&mdexFactoryABI, mdexFactoryABIJson,
		},
		{
			&shibaswapPairABI, shibaswapPairABIJson,
		},
		{
			&croDefiSwapFactoryABI, croDefiSwapFactoryABIJson,
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
