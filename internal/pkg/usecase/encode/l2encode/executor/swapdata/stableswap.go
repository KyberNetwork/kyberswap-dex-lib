package swapdata

import (
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

type StableSwap struct {
	PoolMappingID pack.UInt24
	Pool          common.Address
	TokenIndexTo  uint8
	Dx            *big.Int
	PoolLp        common.Address
	IsSaddle      bool

	isFirstSwap bool
}

func PackStableSwap(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildStableSwap(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packStableSwap(swap)
}

func UnpackStableSwap(data []byte, isFirstSwap bool) (StableSwap, error) {
	var swap StableSwap
	var startByte int

	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.Pool, startByte = pack.ReadAddress(data, startByte)
	}
	swap.TokenIndexTo, startByte = pack.ReadUInt8(data, startByte)

	if isFirstSwap {
		swap.Dx, startByte = pack.ReadBigInt(data, startByte)
		swap.isFirstSwap = true
	} else {
		var collectAmountFlag bool
		collectAmountFlag, startByte = pack.ReadBoolean(data, startByte)
		if collectAmountFlag {
			swap.Dx = abi.MaxUint256
		}
	}

	swap.PoolLp, startByte = pack.ReadAddress(data, startByte)
	swap.IsSaddle, _ = pack.ReadBoolean(data, startByte)

	return swap, nil
}

func buildStableSwap(swap types.L2EncodingSwap) (StableSwap, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return StableSwap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[PackStableSwap] err :[%v]",
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
			"[PackStableSwap] err :[%v]",
			err,
		)
	}

	return StableSwap{
		PoolMappingID: swap.PoolMappingID,
		Pool:          common.HexToAddress(swap.Pool),
		TokenIndexTo:  extra.TokenOutIndex,
		Dx:            swap.SwapAmount,
		PoolLp:        common.HexToAddress(swap.Pool),
		IsSaddle:      swap.PoolType == constant.PoolTypes.Saddle,

		isFirstSwap: swap.IsFirstSwap,
	}, nil
}

func packStableSwap(swap StableSwap) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.PoolMappingID)
	if swap.PoolMappingID == 0 {
		args = append(args, swap.Pool)
	}
	args = append(args, swap.TokenIndexTo)
	if swap.isFirstSwap {
		args = append(args, swap.Dx)
	} else {
		var collectAmountFlag bool
		if swap.Dx.Cmp(constant.Zero) > 0 {
			collectAmountFlag = true
		}
		args = append(args, collectAmountFlag)
	}
	args = append(args, swap.PoolLp, swap.IsSaddle)

	return pack.Pack(args...)
}
