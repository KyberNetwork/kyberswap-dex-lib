package swapdata

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// TokenWeightInputUniSwap weight of tokenIn, it's 50 with Uniswap
// TODO: Should read from extra field of the swap instead
const TokenWeightInputUniSwap = 50
const actualAmountOutPercents = 9995

func PackUniSwap(chainID valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildUniSwap(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	return packUniSwap(swap)
}

func UnpackUniSwap(data []byte) (Uniswap, error) {
	unpacked, err := UniswapABIArguments.Unpack(data)
	if err != nil {
		return Uniswap{}, err
	}

	var swap Uniswap
	if err = UniswapABIArguments.Copy(&swap, unpacked); err != nil {
		return Uniswap{}, err
	}

	return swap, nil
}

func buildUniSwap(chainID valueobject.ChainID, swap types.EncodingSwap) (Uniswap, error) {
	swapFee := GetFee(chainID, swap.Exchange)

	fee := swapFee.Fee
	if shouldAddActualAmountOutPercents(swap.Exchange) {
		// [16 bits for actualAmountOutPercents][16 bits for dex swap fee]
		fee |= actualAmountOutPercents << 16
	}

	return Uniswap{
		Pool:             common.HexToAddress(swap.Pool),
		TokenIn:          common.HexToAddress(swap.TokenIn),
		TokenOut:         common.HexToAddress(swap.TokenOut),
		Recipient:        common.HexToAddress(swap.Recipient),
		CollectAmount:    swap.CollectAmount,
		SwapFee:          fee,
		FeePrecision:     swapFee.Precision,
		TokenWeightInput: TokenWeightInputUniSwap,
	}, nil
}

func packUniSwap(swap Uniswap) ([]byte, error) {
	return UniswapABIArguments.Pack(
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.Recipient,
		swap.CollectAmount,
		swap.SwapFee,
		swap.FeePrecision,
		swap.TokenWeightInput,
	)
}

func shouldAddActualAmountOutPercents(exchange valueobject.Exchange) bool {
	return exchange == valueobject.ExchangeGravity || exchange == valueobject.ExchangeEchoDex
}
