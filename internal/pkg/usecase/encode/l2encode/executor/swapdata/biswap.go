package swapdata

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/executor/swapdata"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackBiswap(chainID valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildBiswap(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	return packUniswap(swap)
}

func UnpackBiswap(data []byte, isFirstSwap bool) (Uniswap, error) {
	return UnpackUniswap(data, isFirstSwap)
}

func buildBiswap(chainID valueobject.ChainID, swap types.L2EncodingSwap) (Uniswap, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return Uniswap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[buildBiswap] err :[%v]",
			err,
		)
	}

	var extra struct {
		SwapFee string `json:"swapFee"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return Uniswap{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[buildBiswap] err :[%v]",
			err,
		)
	}

	defaultSwapFee := swapdata.GetFee(chainID, swap.Exchange)
	swapFeeBI, _ := new(big.Int).SetString(extra.SwapFee, 10)

	// Override custom swap fee value
	fee := new(big.Int).Div(new(big.Int).Mul(swapFeeBI, big.NewInt(int64(defaultSwapFee.Precision))), constant.BONE)

	return Uniswap{
		PoolMappingID:    swap.PoolMappingID,
		Pool:             common.HexToAddress(swap.Pool),
		Recipient:        common.HexToAddress(swap.Recipient),
		CollectAmount:    swap.CollectAmount,
		SwapFee:          uint32(fee.Int64()),
		FeePrecision:     defaultSwapFee.Precision,
		TokenWeightInput: TokenWeightInputUniSwap,

		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
	}, nil
}
