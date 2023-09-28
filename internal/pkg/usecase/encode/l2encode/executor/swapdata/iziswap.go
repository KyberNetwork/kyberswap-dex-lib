package swapdata

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IZiSwap struct {
	PoolMappingID pack.UInt24
	Pool          common.Address
	TokenOut      common.Address
	Recipient     common.Address
	SwapAmount    *big.Int
	LimitPoint    pack.Int24

	recipientFlag uint8
	isFirstSwap   bool
}

func PackIZiSwap(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildIZiSwap(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packIZiSwap(swap)
}

func UnpackIZiSwap(data []byte, isFirstSwap bool) (IZiSwap, error) {
	var swap IZiSwap
	var startByte int

	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.Pool, startByte = pack.ReadAddress(data, startByte)
	}

	swap.TokenOut, startByte = pack.ReadAddress(data, startByte)

	swap.recipientFlag, startByte = pack.ReadUInt8(data, startByte)
	if swap.recipientFlag == 0 {
		swap.Recipient, startByte = pack.ReadAddress(data, startByte)
	} else {
		swap.Recipient = common.BytesToAddress([]byte{swap.recipientFlag})
	}

	if isFirstSwap {
		swap.SwapAmount, startByte = pack.ReadBigInt(data, startByte)
		swap.isFirstSwap = true
	} else {
		var collectAmountFlag bool
		collectAmountFlag, startByte = pack.ReadBoolean(data, startByte)
		if collectAmountFlag {
			swap.SwapAmount = abi.MaxUint256
		}
	}

	swap.LimitPoint, _ = pack.ReadInt24(data, startByte)

	return swap, nil
}

func buildIZiSwap(swap types.L2EncodingSwap) (IZiSwap, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return IZiSwap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildIZiSwap] err :[%v]",
			err,
		)
	}

	var extra struct {
		LimitPoint int32 `json:"limitPoint"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return IZiSwap{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildIZiSwap] err :[%v]",
			err,
		)
	}

	return IZiSwap{
		Pool:          common.HexToAddress(swap.Pool),
		TokenOut:      common.HexToAddress(swap.TokenOut),
		Recipient:     common.HexToAddress(swap.Recipient),
		SwapAmount:    swap.SwapAmount,
		LimitPoint:    pack.Int24(extra.LimitPoint),
		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
	}, nil
}

func packIZiSwap(swap IZiSwap) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.PoolMappingID)
	if swap.PoolMappingID == 0 {
		args = append(args, swap.Pool)
	}

	args = append(args, swap.TokenOut)

	args = append(args, swap.recipientFlag)
	if swap.recipientFlag == 0 {
		args = append(args, swap.Recipient)
	}

	if swap.isFirstSwap {
		args = append(args, swap.SwapAmount)
	} else {
		var collectAmountFlag bool
		if swap.SwapAmount.Cmp(constant.Zero) > 0 {
			collectAmountFlag = true
		}
		args = append(args, collectAmountFlag)
	}

	args = append(args, pack.Int24(swap.LimitPoint))

	return pack.Pack(args...)
}
