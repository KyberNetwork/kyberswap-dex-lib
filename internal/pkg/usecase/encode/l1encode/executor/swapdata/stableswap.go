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

func PackStableSwap(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildStableSwap(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packStableSwap(swap)
}
func UnpackStableSwap(encodedSwap []byte) (StableSwap, error) {
	unpacked, err := StableSwapABIArguments.Unpack(encodedSwap)
	if err != nil {
		return StableSwap{}, err
	}

	var swap StableSwap
	if err = StableSwapABIArguments.Copy(&swap, unpacked); err != nil {
		return StableSwap{}, err
	}

	return swap, nil
}

func buildStableSwap(swap types.EncodingSwap) (StableSwap, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return StableSwap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildStableSwap] err :[%v]",
			err,
		)
	}

	var extra struct {
		TokenInIndex  uint8 `json:"tokenInIndex"`
		TokenOutIndex uint8 `json:"tokenOutIndex"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return StableSwap{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildStableSwap] err :[%v]",
			err,
		)
	}

	return StableSwap{
		Pool:           common.HexToAddress(swap.Pool),
		TokenFrom:      common.HexToAddress(swap.TokenIn),
		TokenTo:        common.HexToAddress(swap.TokenOut),
		TokenIndexFrom: extra.TokenInIndex,
		TokenIndexTo:   extra.TokenOutIndex,
		Dx:             swap.SwapAmount,
		PoolLength:     big.NewInt(int64(swap.PoolLength)),
		PoolLp:         common.HexToAddress(swap.Pool),
		IsSaddle:       swap.PoolType == constant.PoolTypes.Saddle,
	}, nil
}

func packStableSwap(swap StableSwap) ([]byte, error) {
	return StableSwapABIArguments.Pack(
		swap.Pool,
		swap.TokenFrom,
		swap.TokenTo,
		swap.TokenIndexFrom,
		swap.TokenIndexTo,
		swap.Dx,
		swap.PoolLength,
		swap.PoolLp,
		swap.IsSaddle,
	)
}
