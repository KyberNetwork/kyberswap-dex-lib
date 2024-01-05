package swapdata

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// TokenWeightInputUniSwap weight of tokenIn, it's 50 with Uniswap
// TODO: Should read from extra field of the swap instead
const TokenWeightInputUniSwap = 50

type UniswapV2PoolExtra struct {
	Fee          uint32 `json:"fee"`
	FeePrecision uint32 `json:"feePrecision"`
}

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
	swapFee, err := GetFeeFromPoolExtra(swap)
	if err != nil {
		swapFee = GetFee(chainID, swap.Exchange)
	}

	fee := getCustomSwapFee(swap.Exchange, swapFee.Fee)

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

func GetFeeFromPoolExtra(poolExtra interface{}) (Fee, error) {
	byteData, err := json.Marshal(poolExtra)
	if err != nil {
		return Fee{}, errors.Wrapf(
			ErrMarshalFailed,
			"[getFeeFromPoolExtra] err :[%v]",
			err,
		)
	}

	var extra struct {
		Fee          uint32 `json:"fee"`
		FeePrecision uint32 `json:"feePrecision"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return Fee{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[getFeeFromPoolExtra] err :[%v]",
			err,
		)
	}

	if extra.Fee == 0 || extra.FeePrecision == 0 {
		return Fee{}, errors.New("invalid fee")
	}

	return Fee{Fee: extra.Fee, Precision: extra.FeePrecision}, nil
}

func getCustomSwapFee(exchange valueobject.Exchange, fee uint32) uint32 {
	// Contexts:
	// https://team-kyber.slack.com/archives/C04R9NSNEKF/p1689673420103049
	// https://team-kyber.slack.com/archives/C04R9NSNEKF/p1690536551669419
	// swapFee: [16 bits for actualAmountOutPercents][16 bits for dex swap fee]
	switch exchange {
	case valueobject.ExchangeGravity:
		// For Gravity, actualAmountOutPercents = 9995
		return fee | (9995 << 16)
	case valueobject.ExchangeEchoDex:
		// For Echo dex, actualAmountOutPercents = 9970
		return 9970 << 16
	default:
		return fee
	}
}
