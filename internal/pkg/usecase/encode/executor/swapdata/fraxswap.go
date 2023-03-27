package swapdata

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/types"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

func PackFraxSwap(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildFraxSwap(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packFraxSwap(swap)
}

func buildFraxSwap(swap types.EncodingSwap) (UniSwap, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return UniSwap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[builFraxSwap] err :[%v]",
			err,
		)
	}

	var extra struct {
		SwapFee      uint32 `json:"swapFee"`
		FeePrecision uint32 `json:"feePrecision"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return UniSwap{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[builFraxSwap] err :[%v]",
			err,
		)
	}

	return UniSwap{
		Pool:              common.HexToAddress(swap.Pool),
		TokenIn:           common.HexToAddress(swap.TokenIn),
		TokenOut:          common.HexToAddress(swap.TokenOut),
		Recipient:         common.HexToAddress(swap.Recipient),
		CollectAmount:     swap.CollectAmount,
		LimitReturnAmount: swap.LimitReturnAmount,
		SwapFee:           extra.SwapFee,
		FeePrecision:      extra.FeePrecision,
		TokenWeightInput:  TokenWeightInputUniSwap,
	}, nil
}

func packFraxSwap(swap UniSwap) ([]byte, error) {
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
