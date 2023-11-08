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

type Smardex struct {
	Recipient     common.Address
	PoolMappingID pack.UInt24
	Pool          common.Address
	TokenIn       common.Address
	TokenOut      common.Address
	Amount        *big.Int

	recipientFlag uint8
	isFirstSwap   bool
}

func PackSmardex(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildSmardex(encodingSwap)
	if err != nil {
		return nil, err
	}
	return packSmardex(swap)
}

func UnpackSmardex(data []byte, isFirstSwap bool) (Smardex, error) {
	var swap Smardex
	var startByte int

	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.Pool, startByte = pack.ReadAddress(data, startByte)
	}
	swap.TokenOut, startByte = pack.ReadAddress(data, startByte)

	if isFirstSwap {
		swap.Amount, startByte = pack.ReadBigInt(data, startByte)
		swap.isFirstSwap = true
	} else {
		var collectAmountFlag bool
		collectAmountFlag, startByte = pack.ReadBoolean(data, startByte)
		if collectAmountFlag {
			swap.Amount = abi.MaxUint256
		}
	}

	swap.recipientFlag, startByte = pack.ReadUInt8(data, startByte)
	if swap.recipientFlag == 0 {
		swap.Recipient, _ = pack.ReadAddress(data, startByte)
	} else {
		swap.Recipient = common.BytesToAddress([]byte{swap.recipientFlag})
	}

	return swap, nil
}

func buildSmardex(swap types.L2EncodingSwap) (Smardex, error) {
	return Smardex{
		Recipient:     common.HexToAddress(swap.Recipient),
		PoolMappingID: swap.PoolMappingID,
		Pool:          common.HexToAddress(swap.Pool),
		TokenIn:       common.HexToAddress(swap.TokenIn),
		TokenOut:      common.HexToAddress(swap.TokenOut),
		Amount:        swap.SwapAmount,

		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
	}, nil
}

func packSmardex(swap Smardex) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.PoolMappingID)
	if swap.PoolMappingID == 0 {
		args = append(args, swap.Pool)
	}
	args = append(args, swap.TokenOut)

	if swap.isFirstSwap {
		args = append(args, swap.Amount)
	} else {
		var collectAmountFlag bool
		if swap.Amount.Cmp(constant.Zero) > 0 {
			collectAmountFlag = true
		}
		args = append(args, collectAmountFlag)
	}

	args = append(args, swap.recipientFlag)
	if swap.recipientFlag == 0 {
		args = append(args, swap.Recipient)
	}

	return pack.Pack(args...)
}
