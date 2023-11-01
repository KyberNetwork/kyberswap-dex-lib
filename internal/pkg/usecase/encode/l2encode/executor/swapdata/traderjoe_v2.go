package swapdata

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type TraderJoeV2 struct {
	PoolMappingID  pack.UInt24
	Recipient      common.Address
	Pool           common.Address
	TokenOut       common.Address
	CollectAmount  *big.Int
	IsTraderJoeV20 bool

	recipientFlag uint8
	isFirstSwap   bool
}

func PackTraderJoeV2(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildTraderJoeV2(encodingSwap)
	if err != nil {
		return nil, err
	}
	return packTraderJoeV2(swap)
}

func UnpackTraderJoeV2(data []byte, isFirstSwap bool) (TraderJoeV2, error) {
	var (
		swap      TraderJoeV2
		startByte int
	)

	swap.recipientFlag, startByte = pack.ReadUInt8(data, startByte)
	if swap.recipientFlag == 0 {
		swap.Recipient, startByte = pack.ReadAddress(data, startByte)
	} else {
		swap.Recipient = common.BytesToAddress([]byte{swap.recipientFlag})
	}

	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.Pool, startByte = pack.ReadAddress(data, startByte)
	}

	swap.TokenOut, startByte = pack.ReadAddress(data, startByte)

	swap.IsTraderJoeV20, startByte = pack.ReadBoolean(data, startByte)

	if isFirstSwap {
		swap.CollectAmount, _ = pack.ReadBigInt(data, startByte)
		swap.isFirstSwap = true
	} else {
		var collectAmountFlag bool
		collectAmountFlag, _ = pack.ReadBoolean(data, startByte)
		if collectAmountFlag {
			swap.CollectAmount = abi.MaxUint256
		}
	}

	return swap, nil
}

func buildTraderJoeV2(swap types.L2EncodingSwap) (TraderJoeV2, error) {
	var isV20 bool
	switch swap.Exchange {
	case valueobject.ExchangeTraderJoeV20:
		isV20 = true
	case valueobject.ExchangeTraderJoeV21:
		isV20 = false
	default:
		return TraderJoeV2{}, fmt.Errorf("[buildTraderJoeV2] unsupported exchange %s", swap.Exchange)
	}

	return TraderJoeV2{
		PoolMappingID:  swap.PoolMappingID,
		Recipient:      common.HexToAddress(swap.Recipient),
		Pool:           common.HexToAddress(swap.Pool),
		TokenOut:       common.HexToAddress(swap.TokenOut),
		CollectAmount:  swap.CollectAmount,
		IsTraderJoeV20: isV20,

		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
	}, nil
}

func packTraderJoeV2(swap TraderJoeV2) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.recipientFlag)
	if swap.recipientFlag == 0 {
		args = append(args, swap.Recipient)
	}

	args = append(args, swap.PoolMappingID)
	if swap.PoolMappingID == 0 {
		args = append(args, swap.Pool)
	}

	args = append(args, swap.TokenOut)

	args = append(args, swap.IsTraderJoeV20)

	if swap.isFirstSwap {
		args = append(args, swap.CollectAmount)
	} else {
		var collectAmountFlag bool
		if swap.CollectAmount.Cmp(constant.Zero) > 0 {
			collectAmountFlag = true
		}
		args = append(args, collectAmountFlag)
	}

	return pack.Pack(args...)
}
