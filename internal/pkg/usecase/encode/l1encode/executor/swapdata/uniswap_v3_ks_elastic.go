package swapdata

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/helper"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackUniswapV3KSElastic(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap := buildUniswapV3KSElastic(encodingSwap)

	return packUniswapV3KSElastic(swap)
}

func UnpackUniswapV3KSElastic(encodedSwap []byte) (UniswapV3KSElastic, error) {
	unpacked, err := UniswapV3KSElasticABIArgument.Unpack(encodedSwap)
	if err != nil {
		return UniswapV3KSElastic{}, err
	}

	var swap UniswapV3KSElastic
	if err = UniswapV3KSElasticABIArgument.Copy(&swap, unpacked); err != nil {
		return UniswapV3KSElastic{}, err
	}

	return swap, nil
}

func buildUniswapV3KSElastic(swap types.EncodingSwap) UniswapV3KSElastic {
	return UniswapV3KSElastic{
		Recipient:         common.HexToAddress(swap.Recipient),
		Pool:              common.HexToAddress(swap.Pool),
		TokenIn:           common.HexToAddress(swap.TokenIn),
		TokenOut:          common.HexToAddress(swap.TokenOut),
		SwapAmount:        swap.SwapAmount,
		SqrtPriceLimitX96: constant.Zero,
		IsUniV3:           helper.IsUniV3Type(swap.PoolType),
	}
}

func packUniswapV3KSElastic(swap UniswapV3KSElastic) ([]byte, error) {
	return UniswapV3KSElasticABIArgument.Pack(
		swap.Recipient,
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.SwapAmount,
		swap.SqrtPriceLimitX96,
		swap.IsUniV3,
	)
}
