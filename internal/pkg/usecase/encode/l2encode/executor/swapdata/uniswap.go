package swapdata

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/executor/swapdata"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type Uniswap struct {
	PoolMappingID    pack.UInt24
	Pool             common.Address
	Recipient        common.Address
	CollectAmount    *big.Int
	SwapFee          uint32
	FeePrecision     uint32
	TokenWeightInput uint32

	recipientFlag uint8
	isFirstSwap   bool
}

const TokenWeightInputUniSwap uint32 = 50

func PackUniswap(chainID valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildUniswap(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	return packUniswap(swap)
}

func UnpackUniswap(data []byte, isFirstSwap bool) (Uniswap, error) {
	var swap Uniswap
	var startByte int

	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.Pool, startByte = pack.ReadAddress(data, startByte)
	}

	swap.recipientFlag, startByte = pack.ReadUInt8(data, startByte)
	if swap.recipientFlag == 0 {
		swap.Recipient, startByte = pack.ReadAddress(data, startByte)
	} else {
		swap.Recipient = common.BytesToAddress([]byte{swap.recipientFlag})
	}

	if isFirstSwap {
		swap.CollectAmount, startByte = pack.ReadBigInt(data, startByte)
		swap.isFirstSwap = true
	} else {
		var collectAmountFlag bool
		collectAmountFlag, startByte = pack.ReadBoolean(data, startByte)
		if collectAmountFlag {
			swap.CollectAmount = abi.MaxUint256
		}
	}

	swap.SwapFee, startByte = pack.ReadUInt32(data, startByte)
	swap.FeePrecision, startByte = pack.ReadUInt32(data, startByte)
	swap.TokenWeightInput, _ = pack.ReadUInt32(data, startByte)

	return swap, nil
}

func buildUniswap(chainID valueobject.ChainID, swap types.L2EncodingSwap) (Uniswap, error) {
	swapFee := swapdata.GetFee(chainID, swap.Exchange)

	return Uniswap{
		PoolMappingID:    swap.PoolMappingID,
		Pool:             common.HexToAddress(swap.Pool),
		Recipient:        common.HexToAddress(swap.Recipient),
		CollectAmount:    swap.CollectAmount,
		SwapFee:          swapFee.Fee,
		FeePrecision:     swapFee.Precision,
		TokenWeightInput: TokenWeightInputUniSwap,

		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
	}, nil
}

func packUniswap(swap Uniswap) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.PoolMappingID)
	if swap.PoolMappingID == 0 {
		args = append(args, swap.Pool)
	}

	args = append(args, swap.recipientFlag)
	if swap.recipientFlag == 0 {
		args = append(args, swap.Recipient)
	}

	if swap.isFirstSwap {
		args = append(args, swap.CollectAmount)
	} else {
		var collectAmountFlag bool
		if swap.CollectAmount.Cmp(constant.Zero) > 0 {
			collectAmountFlag = true
		}
		args = append(args, collectAmountFlag)
	}

	args = append(args, swap.SwapFee)
	args = append(args, swap.FeePrecision)
	args = append(args, swap.TokenWeightInput)

	return pack.Pack(args...)
}
