package swapdata

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackCamelot(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildCamelot(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packCamelot(swap)
}

func buildCamelot(swap types.EncodingSwap) (Uniswap, error) {
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
		Pool:             common.HexToAddress(swap.Pool),
		TokenIn:          common.HexToAddress(swap.TokenIn),
		TokenOut:         common.HexToAddress(swap.TokenOut),
		Recipient:        common.HexToAddress(swap.Recipient),
		CollectAmount:    swap.CollectAmount,
		SwapFee:          extra.SwapFee,
		FeePrecision:     extra.FeePrecision,
		TokenWeightInput: TokenWeightInputUniSwap,
	}, nil
}

func packCamelot(swap Uniswap) ([]byte, error) {
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
