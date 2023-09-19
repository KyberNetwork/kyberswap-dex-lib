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

type Synthetix struct {
	PoolMappingID          pack.UInt24
	SynthetixProxy         common.Address
	TokenOut               common.Address
	SourceCurrencyKey      [32]byte
	SourceAmount           *big.Int
	DestinationCurrencyKey [32]byte
	UseAtomicExchange      bool

	isFirstSwap bool
}

func PackSynthetix(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildSynthetix(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packSynthetix(swap)
}

func UnpackSynthetix(data []byte, isFirstSwap bool) (Synthetix, error) {
	var swap Synthetix
	var startByte int

	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.SynthetixProxy, startByte = pack.ReadAddress(data, startByte)
	}

	swap.TokenOut, startByte = pack.ReadAddress(data, startByte)
	for i := 0; i < 32; i++ {
		swap.SourceCurrencyKey[i], startByte = pack.ReadUInt8(data, startByte)
	}

	if isFirstSwap {
		swap.SourceAmount, startByte = pack.ReadBigInt(data, startByte)
		swap.isFirstSwap = true
	} else {
		var collectAmountFlag bool
		collectAmountFlag, startByte = pack.ReadBoolean(data, startByte)
		if collectAmountFlag {
			swap.SourceAmount = abi.MaxUint256
		}
	}

	for i := 0; i < 32; i++ {
		swap.DestinationCurrencyKey[i], startByte = pack.ReadUInt8(data, startByte)
	}
	swap.UseAtomicExchange, _ = pack.ReadBoolean(data, startByte)

	return swap, nil
}

func buildSynthetix(swap types.L2EncodingSwap) (Synthetix, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return Synthetix{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildSynthetix] err :[%v]",
			err,
		)
	}

	var meta struct {
		SourceCurrencyKey      string `json:"sourceCurrencyKey"`
		DestinationCurrencyKey string `json:"destinationCurrencyKey"`
		UseAtomicExchange      bool   `json:"useAtomicExchange"`
	}

	if err = json.Unmarshal(byteData, &meta); err != nil {
		return Synthetix{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildSynthetix] err :[%v]",
			err,
		)
	}

	return Synthetix{
		PoolMappingID:          swap.PoolMappingID,
		SynthetixProxy:         common.HexToAddress(swap.Pool),
		TokenOut:               common.HexToAddress(swap.TokenOut),
		SourceCurrencyKey:      eth.StringToBytes32(meta.SourceCurrencyKey),
		SourceAmount:           swap.SwapAmount,
		DestinationCurrencyKey: eth.StringToBytes32(meta.DestinationCurrencyKey),
		UseAtomicExchange:      meta.UseAtomicExchange,

		isFirstSwap: swap.IsFirstSwap,
	}, nil
}

func packSynthetix(swap Synthetix) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.PoolMappingID)
	if swap.PoolMappingID == 0 {
		args = append(args, swap.SynthetixProxy)
	}

	args = append(args, swap.TokenOut)

	for _, v := range swap.SourceCurrencyKey {
		args = append(args, v)
	}

	if swap.isFirstSwap {
		args = append(args, swap.SourceAmount)
	} else {
		var collectAmountFlag bool
		if swap.SourceAmount.Cmp(constant.Zero) > 0 {
			collectAmountFlag = true
		}
		args = append(args, collectAmountFlag)
	}

	for _, v := range swap.DestinationCurrencyKey {
		args = append(args, v)
	}

	args = append(args, swap.UseAtomicExchange)

	return pack.Pack(args...)
}
