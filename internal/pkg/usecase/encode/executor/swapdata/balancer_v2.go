package swapdata

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackBalancerV2(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildBalancerV2(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packBalancerV2(swap)
}

func UnpackBalancerV2(encodedSwap []byte) (BalancerV2, error) {
	unpacked, err := BalancerV2ABIArguments.Unpack(encodedSwap)
	if err != nil {
		return BalancerV2{}, err
	}

	var swap BalancerV2
	if err = BalancerV2ABIArguments.Copy(&swap, unpacked); err != nil {
		return BalancerV2{}, err
	}

	return swap, nil
}

func buildBalancerV2(swap types.EncodingSwap) (BalancerV2, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return BalancerV2{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildBalancerV2] err :[%v]",
			err,
		)
	}

	var extra struct {
		VaultAddress string `json:"vault"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return BalancerV2{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[BuildBalancerV2] err :[%v]",
			err,
		)
	}

	var pool [32]byte
	copy(pool[:], common.FromHex(swap.Pool))

	return BalancerV2{
		Vault:    common.HexToAddress(extra.VaultAddress),
		PoolId:   pool,
		AssetIn:  common.HexToAddress(swap.TokenIn),
		AssetOut: common.HexToAddress(swap.TokenOut),
		Amount:   swap.SwapAmount,
		Limit:    swap.LimitReturnAmount,
	}, nil
}

func packBalancerV2(swap BalancerV2) ([]byte, error) {
	return BalancerV2ABIArguments.Pack(
		swap.Vault,
		swap.PoolId,
		swap.AssetIn,
		swap.AssetOut,
		swap.Amount,
		swap.Limit,
	)
}
