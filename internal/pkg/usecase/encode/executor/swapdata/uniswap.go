package swapdata

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/types"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

// TokenWeightInputUniSwap weight of tokenIn, it's 50 with Uniswap
// TODO: Should read from extra field of the swap instead
const TokenWeightInputUniSwap = 50

func PackUniSwap(chainID valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildUniSwap(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	return packUniSwap(swap)
}

func UnpackUniSwap(data []byte) (UniSwap, error) {
	unpacked, err := UniSwapABIArguments.Unpack(data)
	if err != nil {
		return UniSwap{}, err
	}

	var swap UniSwap
	if err = UniSwapABIArguments.Copy(&swap, unpacked); err != nil {
		return UniSwap{}, err
	}

	return swap, nil
}

func buildUniSwap(chainID valueobject.ChainID, swap types.EncodingSwap) (UniSwap, error) {
	swapFee := GetFee(chainID, swap.Exchange)

	return UniSwap{
		Pool:              common.HexToAddress(swap.Pool),
		TokenIn:           common.HexToAddress(swap.TokenIn),
		TokenOut:          common.HexToAddress(swap.TokenOut),
		Recipient:         common.HexToAddress(swap.Recipient),
		CollectAmount:     swap.CollectAmount,
		LimitReturnAmount: swap.LimitReturnAmount,
		SwapFee:           swapFee.Fee,
		FeePrecision:      swapFee.Precision,
		TokenWeightInput:  TokenWeightInputUniSwap,
	}, nil
}

func packUniSwap(swap UniSwap) ([]byte, error) {
	return UniSwapABIArguments.Pack(
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.Recipient,
		swap.CollectAmount,
		swap.LimitReturnAmount,
		swap.SwapFee,
		swap.FeePrecision,
		swap.TokenWeightInput,
	)
}
