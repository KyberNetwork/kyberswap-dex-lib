package swapdata

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type MaverickV1Swap struct {
	PoolMappingID pack.UInt24
	Pool          common.Address
	TokenOut      common.Address
	Recipient     common.Address
	SwapAmount    *big.Int

	recipientFlag uint8
	isFirstSwap   bool
}

func PackMaverickV1(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildMaverickV1(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packMaverickV1(swap)
}

func UnpackMaverickV1(data []byte, isFirstSwap bool) (MaverickV1Swap, error) {
	var swap MaverickV1Swap
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
		swap.SwapAmount, _ = pack.ReadBigInt(data, startByte)
		swap.isFirstSwap = true
	} else {
		var collectAmountFlag bool
		collectAmountFlag, _ = pack.ReadBoolean(data, startByte)
		if collectAmountFlag {
			swap.SwapAmount = abi.MaxUint256
		}
	}

	return swap, nil
}

func buildMaverickV1(swap types.L2EncodingSwap) (MaverickV1Swap, error) {
	return MaverickV1Swap{
		PoolMappingID: swap.PoolMappingID,
		Pool:          common.HexToAddress(swap.Pool),
		TokenOut:      common.HexToAddress(swap.TokenOut),
		Recipient:     common.HexToAddress(swap.Recipient),
		SwapAmount:    swap.SwapAmount,

		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
	}, nil
}

func packMaverickV1(swap MaverickV1Swap) ([]byte, error) {
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

	return pack.Pack(args...)
}
