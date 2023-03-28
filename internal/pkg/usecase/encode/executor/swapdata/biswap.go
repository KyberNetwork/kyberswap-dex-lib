package swapdata

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackBiSwap(chainID valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildBiswap(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	return packBiswap(swap)
}

func buildBiswap(chainID valueobject.ChainID, swap types.EncodingSwap) (UniSwap, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return UniSwap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[buildBiswap] err :[%v]",
			err,
		)
	}

	var extra struct {
		SwapFee string `json:"swapFee"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return UniSwap{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[buildBiswap] err :[%v]",
			err,
		)
	}

	defaultSwapFee := GetFee(chainID, swap.Exchange)
	swapFeeBI, _ := new(big.Int).SetString(extra.SwapFee, 10)

	// Override custom swap fee value
	fee := new(big.Int).Div(new(big.Int).Mul(swapFeeBI, big.NewInt(int64(defaultSwapFee.Precision))), constant.BONE)

	return UniSwap{
		Pool:              common.HexToAddress(swap.Pool),
		TokenIn:           common.HexToAddress(swap.TokenIn),
		TokenOut:          common.HexToAddress(swap.TokenOut),
		Recipient:         common.HexToAddress(swap.Recipient),
		CollectAmount:     swap.CollectAmount,
		LimitReturnAmount: swap.LimitReturnAmount,
		SwapFee:           uint32(fee.Int64()),
		FeePrecision:      defaultSwapFee.Precision,
		TokenWeightInput:  TokenWeightInputUniSwap,
	}, nil
}

func packBiswap(swap UniSwap) ([]byte, error) {
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
