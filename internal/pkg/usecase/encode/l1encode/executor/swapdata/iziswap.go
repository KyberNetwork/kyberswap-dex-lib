package swapdata

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackIZiSwap(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildIZiSwap(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packIZiSwap(swap)
}

func UnpackIZiSwap(encodedSwap []byte) (IZiSwap, error) {
	unpacked, err := IZiSwapArguments.Unpack(encodedSwap)
	if err != nil {
		return IZiSwap{}, err
	}

	var swap IZiSwap
	if err = IZiSwapArguments.Copy(&swap, unpacked); err != nil {
		return IZiSwap{}, err
	}

	return swap, nil
}

func buildIZiSwap(swap types.EncodingSwap) (IZiSwap, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return IZiSwap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildIZiSwap] err :[%v]",
			err,
		)
	}

	var extra struct {
		LimitPoint int64 `json:"limitPoint"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return IZiSwap{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildIZiSwap] err :[%v]",
			err,
		)
	}

	return IZiSwap{
		Pool:       common.HexToAddress(swap.Pool),
		TokenIn:    common.HexToAddress(swap.TokenIn),
		TokenOut:   common.HexToAddress(swap.TokenOut),
		Recipient:  common.HexToAddress(swap.Recipient),
		SwapAmount: swap.SwapAmount,
		LimitPoint: new(big.Int).SetInt64(extra.LimitPoint),
	}, nil
}

func packIZiSwap(swap IZiSwap) ([]byte, error) {
	return IZiSwapArguments.Pack(
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.Recipient,
		swap.SwapAmount,
		swap.LimitPoint,
	)
}
