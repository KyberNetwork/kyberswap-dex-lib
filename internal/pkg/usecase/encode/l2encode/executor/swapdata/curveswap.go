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
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type CurveSwap struct {
	CanGetToken       bool
	PoolMappingID     pack.UInt24
	Pool              common.Address
	TokenTo           common.Address
	TokenIndexFrom    uint8
	TokenIndexTo      uint8
	Dx                *big.Int
	UsePoolUnderlying bool
	UseTriCrypto      bool

	isFirstSwap bool
}

func PackCurveSwap(chainID valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildCurveSwap(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	return packCurveSwap(swap)
}

func UnpackCurveSwap(data []byte, isFirstSwap bool) (CurveSwap, error) {
	var swap CurveSwap
	var startByte int

	swap.CanGetToken, startByte = pack.ReadBoolean(data, startByte)
	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.Pool, startByte = pack.ReadAddress(data, startByte)
	}

	if !swap.CanGetToken {
		swap.TokenTo, startByte = pack.ReadAddress(data, startByte)
		swap.TokenIndexFrom, startByte = pack.ReadUInt8(data, startByte)
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
	swap.UsePoolUnderlying, startByte = pack.ReadBoolean(data, startByte)
	swap.UseTriCrypto, _ = pack.ReadBoolean(data, startByte)

	return swap, nil
}

func buildCurveSwap(chainID valueobject.ChainID, swap types.L2EncodingSwap) (CurveSwap, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return CurveSwap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildCurveSwap] err :[%v]",
			err,
		)
	}

	var extra struct {
		TokenInIndex  uint8 `json:"tokenInIndex"`
		TokenOutIndex uint8 `json:"tokenOutIndex"`
		Underlying    bool  `json:"underlying"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return CurveSwap{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildCurveSwap] err :[%v]",
			err,
		)
	}

	useTriCrypto := swap.PoolType == constant.PoolTypes.CurveTricrypto || swap.PoolType == constant.PoolTypes.CurveTwo

	tokenTo := common.HexToAddress(swap.TokenOut)
	if !useTriCrypto && eth.IsWETH(swap.TokenOut, chainID) {
		tokenTo = common.HexToAddress(valueobject.EtherAddress)
	}

	// canGetToken true if Curve pool allows to read pool's tokens, by exposing function `coins` or `underlying_coins`.
	canGetToken := true

	// CurveMeta pools don't have `underlying_coins` function,
	// so we set canGetToken = false if swapping underlying coins through these pools.
	if extra.Underlying && swap.PoolType == constant.PoolTypes.CurveMeta {
		canGetToken = false
	}

	return CurveSwap{
		CanGetToken:       canGetToken,
		PoolMappingID:     swap.PoolMappingID,
		Pool:              common.HexToAddress(swap.Pool),
		TokenTo:           tokenTo,
		TokenIndexFrom:    extra.TokenInIndex,
		TokenIndexTo:      extra.TokenOutIndex,
		Dx:                swap.SwapAmount,
		UsePoolUnderlying: extra.Underlying,
		UseTriCrypto:      useTriCrypto,

		isFirstSwap: swap.IsFirstSwap,
	}, nil
}

func packCurveSwap(swap CurveSwap) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.CanGetToken, swap.PoolMappingID)
	if swap.PoolMappingID == 0 {
		args = append(args, swap.Pool)
	}
	if !swap.CanGetToken {
		args = append(args, swap.TokenTo, swap.TokenIndexFrom)
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
	args = append(args, swap.UsePoolUnderlying, swap.UseTriCrypto)

	return pack.Pack(args...)
}
