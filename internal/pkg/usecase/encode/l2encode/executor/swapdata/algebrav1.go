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

type AlgebraV1 struct {
	Recipient           common.Address
	PoolMappingID       pack.UInt24
	Pool                common.Address
	TokenOut            common.Address
	SwapAmount          *big.Int
	SenderFeeOnTransfer *big.Int

	recipientFlag uint8
	isFirstSwap   bool
}

func PackAlgebraV1(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap := buildAlgebraV1(encodingSwap)

	return packAlgebraV1(swap)
}

func UnpackAlgebraV1(data []byte, isFirstSwap bool) (AlgebraV1, error) {
	var swap AlgebraV1
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

	swap.TokenOut, startByte = pack.ReadAddress(data, startByte)

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

	swap.SenderFeeOnTransfer, _ = pack.ReadBigIntAsInt256(data, startByte)

	return swap, nil
}

func buildAlgebraV1(swap types.L2EncodingSwap) AlgebraV1 {
	return AlgebraV1{
		Recipient:           common.HexToAddress(swap.Recipient),
		PoolMappingID:       swap.PoolMappingID,
		Pool:                common.HexToAddress(swap.Pool),
		TokenOut:            common.HexToAddress(swap.TokenOut),
		SwapAmount:          swap.SwapAmount,
		SenderFeeOnTransfer: constant.Zero, // TODO: support FoT token

		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
	}
}

func packAlgebraV1(swap AlgebraV1) ([]byte, error) {
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

	if swap.isFirstSwap {
		args = append(args, swap.SwapAmount)
	} else {
		var collectAmountFlag bool
		if swap.SwapAmount.Cmp(constant.Zero) > 0 {
			collectAmountFlag = true
		}
		args = append(args, collectAmountFlag)
	}

	// SenderFeeOnTransfer packs to 32 bytes,
	// [ FoT_FLAG(1 bit) ... SENDER_ADDRESS(160 bits) ]
	args = append(args, pack.RawBytes(pack.PackBigInt(swap.SenderFeeOnTransfer, 32)))

	return pack.Pack(args...)
}
