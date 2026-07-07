package uniswapv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/abis"
)

var (
	uniswapV2PairABI         abi.ABI
	uniswapV2FactoryABI      abi.ABI
	uniswapV2FactoryFilterer *uniswapv2.UniswapV2FactoryFilterer
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
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
	uniswapV2FactoryFilterer = lo.Must(uniswapv2.NewUniswapV2FactoryFilterer(common.Address{}, nil))
}
