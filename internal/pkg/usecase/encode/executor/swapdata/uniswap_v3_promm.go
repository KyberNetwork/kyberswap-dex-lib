package swapdata

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackUniSwapV3ProMM(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap := buildUniSwapV3ProMM(encodingSwap)

	return packUniSwapV3ProMM(swap)
}

func UnpackUniSwapV3ProMM(encodedSwap []byte) (UniSwapV3ProMM, error) {
	unpacked, err := UniSwapV3ProMMABIArguments.Unpack(encodedSwap)
	if err != nil {
		return UniSwapV3ProMM{}, err
	}

	var swap UniSwapV3ProMM
	if err = UniSwapV3ProMMABIArguments.Copy(&swap, unpacked); err != nil {
		return UniSwapV3ProMM{}, err
	}

	return swap, nil
}

func buildUniSwapV3ProMM(swap types.EncodingSwap) UniSwapV3ProMM {
	return UniSwapV3ProMM{
		Recipient:         common.HexToAddress(swap.Recipient),
		Pool:              common.HexToAddress(swap.Pool),
		TokenIn:           common.HexToAddress(swap.TokenIn),
		TokenOut:          common.HexToAddress(swap.TokenOut),
		SwapAmount:        swap.SwapAmount,
		LimitReturnAmount: swap.LimitReturnAmount,
		SqrtPriceLimitX96: constant.Zero,
		IsUniV3:           swap.PoolType == constant.PoolTypes.UniV3,
	}
}

func packUniSwapV3ProMM(swap UniSwapV3ProMM) ([]byte, error) {
	return UniSwapV3ProMMABIArguments.Pack(
		swap.Recipient,
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.SwapAmount,
		swap.LimitReturnAmount,
		swap.SqrtPriceLimitX96,
		swap.IsUniV3,
	)
}
