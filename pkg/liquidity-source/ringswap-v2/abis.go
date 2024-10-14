package ringswapv2

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
	zkSwapFinancePairABI  abi.ABI
	fewWrappedTokenABI    abi.ABI
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
		{
			&zkSwapFinancePairABI, zkSwapFinancePairABIJson,
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
