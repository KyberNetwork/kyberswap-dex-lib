package swapdata

import (
	"encoding/json"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

func PackCamelot(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildCamelot(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packUniswap(swap)
}

func UnpackCamelot(data []byte, isFirstSwap bool) (Uniswap, error) {
	return UnpackUniswap(data, isFirstSwap)
}

func buildCamelot(swap types.L2EncodingSwap) (Uniswap, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return Uniswap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[buildCamelot] err :[%v]",
			err,
		)
	}

	var extra struct {
		SwapFee      uint32 `json:"swapFee"`
		FeePrecision uint32 `json:"feePrecision"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return Uniswap{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[buildCamelot] err :[%v]",
			err,
		)
	}

	return Uniswap{
		PoolMappingID:    swap.PoolMappingID,
		Pool:             common.HexToAddress(swap.Pool),
		Recipient:        common.HexToAddress(swap.Recipient),
		CollectAmount:    swap.CollectAmount,
		SwapFee:          extra.SwapFee,
		FeePrecision:     extra.FeePrecision,
		TokenWeightInput: TokenWeightInputUniSwap,

		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
	}, nil
}
