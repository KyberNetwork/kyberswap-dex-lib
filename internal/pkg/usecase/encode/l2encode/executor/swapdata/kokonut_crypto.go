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

type KokonutCrypto struct {
	PoolMappingID  pack.UInt24
	Pool           common.Address
	TokenIndexFrom uint8
	Dx             *big.Int

	isFirstSwap bool
}

func PackKokonutCrypto(chainID valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildKokonutCrypto(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	return packKokonutCrypto(swap)
}

func UnpackKokonutCrypto(data []byte, isFirstSwap bool) (KokonutCrypto, error) {
	var swap KokonutCrypto
	var startByte int

	swap.PoolMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.PoolMappingID == 0 {
		swap.Pool, startByte = pack.ReadAddress(data, startByte)
	}
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
	swap.TokenIndexFrom, _ = pack.ReadUInt8(data, startByte)

	return swap, nil
}

func buildKokonutCrypto(_ valueobject.ChainID, swap types.L2EncodingSwap) (KokonutCrypto, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return KokonutCrypto{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildKokonutCrypto] err :[%v]",
			err,
		)
	}

	var extra struct {
		TokenInIndex uint8 `json:"tokenInIndex"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return KokonutCrypto{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildKokonutCrypto] err :[%v]",
			err,
		)
	}

	return KokonutCrypto{
		PoolMappingID:  swap.PoolMappingID,
		Pool:           common.HexToAddress(swap.Pool),
		TokenIndexFrom: extra.TokenInIndex,
		Dx:             swap.SwapAmount,

		isFirstSwap: swap.IsFirstSwap,
	}, nil
}

func packKokonutCrypto(swap KokonutCrypto) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.PoolMappingID)
	if swap.PoolMappingID == 0 {
		args = append(args, swap.Pool)
	}
	if swap.isFirstSwap {
		args = append(args, swap.Dx)
	} else {
		var collectAmountFlag bool
		if swap.Dx.Cmp(constant.Zero) > 0 {
			collectAmountFlag = true
		}
		args = append(args, collectAmountFlag)
	}
	args = append(args, swap.TokenIndexFrom)

	return pack.Pack(args...)
}
