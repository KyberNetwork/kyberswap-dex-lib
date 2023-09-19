package swapdata

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type GMX struct {
	VaultMappingID pack.UInt24
	Vault          common.Address
	TokenOut       common.Address
	Amount         *big.Int
	Receiver       common.Address

	recipientFlag uint8
	isFirstSwap   bool
}

func PackGMX(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap := buildGMX(encodingSwap)

	return packGMX(swap)
}

func UnpackGMX(data []byte, isFirstSwap bool) (GMX, error) {
	var swap GMX
	var startByte int

	swap.VaultMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.VaultMappingID == 0 {
		swap.Vault, startByte = pack.ReadAddress(data, startByte)
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
		swap.Receiver, _ = pack.ReadAddress(data, startByte)
	} else {
		swap.Receiver = common.BytesToAddress([]byte{swap.recipientFlag})
	}

	return swap, nil
}

func buildGMX(swap types.L2EncodingSwap) GMX {
	return GMX{
		VaultMappingID: swap.PoolMappingID,
		Vault:          common.HexToAddress(swap.Pool),
		TokenOut:       common.HexToAddress(swap.TokenOut),
		Amount:         swap.SwapAmount,
		Receiver:       common.HexToAddress(swap.Recipient),

		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
	}
}

func packGMX(swap GMX) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.VaultMappingID)
	if swap.VaultMappingID == 0 {
		args = append(args, swap.Vault)
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
		args = append(args, swap.Receiver)
	}

	return pack.Pack(args...)
}
