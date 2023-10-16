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

type GmxGlp struct {
	RewardRouter common.Address
	StakedGLP    common.Address
	GlpManager   common.Address
	YearnVault   common.Address
	TokenIn      common.Address
	TokenOut     common.Address
	SwapAmount   *big.Int
	Recipient    common.Address

	PoolMappingID pack.UInt24
	recipientFlag uint8
	isFirstSwap   bool
	directionFlag uint8
}

func PackGmxGlp(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildGmxGlp(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packGmxGlp(swap)
}

func UnpackGmxGlp(data []byte, isFirstSwap bool) (GmxGlp, error) {
	var swap GmxGlp
	var startByte int

	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.RewardRouter, startByte = pack.ReadAddress(data, startByte)
	}

	swap.YearnVault, startByte = pack.ReadAddress(data, startByte)

	swap.directionFlag, startByte = pack.ReadUInt8(data, startByte)
	if swap.directionFlag == 1 {
		swap.TokenOut, startByte = pack.ReadAddress(data, startByte)
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

	swap.recipientFlag, startByte = pack.ReadUInt8(data, startByte)
	if swap.recipientFlag == 0 {
		swap.Recipient, _ = pack.ReadAddress(data, startByte)
	} else {
		swap.Recipient = common.BytesToAddress([]byte{swap.recipientFlag})
	}

	return swap, nil
}

func buildGmxGlp(swap types.L2EncodingSwap) (GmxGlp, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return GmxGlp{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildGmxGlp] err :[%v]",
			err,
		)
	}
	var extra struct {
		StakeGLP      string `json:"stakeGLP"`
		GlpManager    string `json:"glpManager"`
		YearnVault    string `json:"yearnVault"`
		DirectionFlag uint8  `json:"directionFlag"`
	}

	if err := json.Unmarshal(byteData, &extra); err != nil {
		return GmxGlp{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildGmxGlp] err :[%v]",
			err,
		)
	}

	return GmxGlp{
		RewardRouter: common.HexToAddress(swap.Pool),
		GlpManager:   common.HexToAddress(extra.GlpManager),
		StakedGLP:    common.HexToAddress(extra.StakeGLP),
		YearnVault:   common.HexToAddress(extra.YearnVault),
		TokenIn:      common.HexToAddress(swap.TokenIn),
		TokenOut:     common.HexToAddress(swap.TokenOut),
		SwapAmount:   swap.SwapAmount,
		Recipient:    common.HexToAddress(swap.Recipient),

		PoolMappingID: swap.PoolMappingID,
		recipientFlag: swap.RecipientFlag,
		isFirstSwap:   swap.IsFirstSwap,
		directionFlag: extra.DirectionFlag,
	}, nil
}

func packGmxGlp(swap GmxGlp) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.PoolMappingID)
	if swap.PoolMappingID == 0 {
		args = append(args, swap.RewardRouter)
	}

	args = append(args, swap.YearnVault)

	args = append(args, swap.directionFlag)
	if swap.directionFlag == 1 {
		args = append(args, swap.TokenOut)
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

	args = append(args, swap.recipientFlag)
	if swap.recipientFlag == 0 {
		args = append(args, swap.Recipient)
	}

	return pack.Pack(args...)
}
