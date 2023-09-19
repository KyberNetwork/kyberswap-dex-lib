package swapdata

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/helper"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type UniswapV3KSElastic struct {
	Recipient     common.Address
	PoolMappingID pack.UInt24
	Pool          common.Address
	SwapAmount    *big.Int
	IsUniV3       bool

	recipientFlag uint8
	isFirstSwap   bool
}

func PackUniswapV3KSElastic(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap := buildUniswapV3KSElastic(encodingSwap)

	return packUniswapV3KSElastic(swap)
}

func UnpackUniswapV3KSElastic(data []byte, isFirstSwap bool) (UniswapV3KSElastic, error) {
	var swap UniswapV3KSElastic
	var startByte int

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
	swap.IsUniV3, _ = pack.ReadBoolean(data, startByte)

	return swap, nil
}

func buildUniswapV3KSElastic(swap types.L2EncodingSwap) UniswapV3KSElastic {
	return UniswapV3KSElastic{
		Recipient:     common.HexToAddress(swap.Recipient),
		PoolMappingID: swap.PoolMappingID,
		Pool:          common.HexToAddress(swap.Pool),
		SwapAmount:    swap.SwapAmount,
		IsUniV3:       helper.IsUniV3Type(swap.PoolType),

		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
	}
}

func packUniswapV3KSElastic(swap UniswapV3KSElastic) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.recipientFlag)
	if swap.recipientFlag == 0 {
		args = append(args, swap.Recipient)
	}

	args = append(args, swap.PoolMappingID)
	if swap.PoolMappingID == 0 {
		args = append(args, swap.Pool)
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
	args = append(args, swap.IsUniV3)

	return pack.Pack(args...)
}
