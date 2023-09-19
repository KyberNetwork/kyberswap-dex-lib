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

type BalancerV2 struct {
	VaultMappingID pack.UInt24
	Vault          common.Address
	PoolId         [32]byte
	AssetOut       uint8
	Amount         *big.Int

	isFirstSwap bool
}

func PackBalancerV2(_ valueobject.ChainID, encodingSwap types.L2EncodingSwap) ([]byte, error) {
	swap, err := buildBalancerV2(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packBalancerV2(swap)
}

func UnpackBalancerV2(data []byte, isFirstSwap bool) (BalancerV2, error) {
	var swap BalancerV2
	var startByte int

	swap.VaultMappingID, startByte = pack.ReadUInt24(data, startByte)
	if swap.VaultMappingID == 0 {
		swap.Vault, startByte = pack.ReadAddress(data, startByte)
	}

	for i := 0; i < 32; i++ {
		swap.PoolId[i], startByte = pack.ReadUInt8(data, startByte)
	}
	swap.AssetOut, startByte = pack.ReadUInt8(data, startByte)

	if isFirstSwap {
		swap.Amount, _ = pack.ReadBigInt(data, startByte)
		swap.isFirstSwap = true
	} else {
		var collectAmountFlag bool
		collectAmountFlag, _ = pack.ReadBoolean(data, startByte)
		if collectAmountFlag {
			swap.Amount = abi.MaxUint256
		}
	}

	return swap, nil
}

func buildBalancerV2(swap types.L2EncodingSwap) (BalancerV2, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return BalancerV2{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildBalancerV2] err :[%v]",
			err,
		)
	}

	var extra struct {
		VaultAddress           string         `json:"vault"`
		PoolId                 string         `json:"poolId"`
		MapTokenAddressToIndex map[string]int `json:"mapTokenAddressToIndex"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return BalancerV2{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildBalancerV2] err :[%v]",
			err,
		)
	}

	var pool [32]byte
	copy(pool[:], common.FromHex(extra.PoolId))

	var assetOutIndex int
	assetOutIndex, exist := extra.MapTokenAddressToIndex[swap.TokenOut]
	if !exist {
		return BalancerV2{}, errors.Errorf("[BuildBalancerV2] cannot find asset out %s in pool %s", swap.TokenOut, swap.Pool)
	}

	return BalancerV2{
		VaultMappingID: swap.PoolMappingID,
		Vault:          common.HexToAddress(extra.VaultAddress),
		PoolId:         pool,
		AssetOut:       uint8(assetOutIndex),
		Amount:         swap.SwapAmount,

		isFirstSwap: swap.IsFirstSwap,
	}, nil
}

func packBalancerV2(swap BalancerV2) ([]byte, error) {
	var args []interface{}

	args = append(args, swap.VaultMappingID)
	if swap.VaultMappingID == 0 {
		args = append(args, swap.Vault)
	}
	for _, item := range swap.PoolId {
		args = append(args, item)
	}
	args = append(args, swap.AssetOut)
	if swap.isFirstSwap {
		args = append(args, swap.Amount)
	} else {
		var collectAmountFlag bool
		if swap.Amount.Cmp(constant.Zero) > 0 {
			collectAmountFlag = true
		}
		args = append(args, collectAmountFlag)
	}

	return pack.Pack(args...)
}
