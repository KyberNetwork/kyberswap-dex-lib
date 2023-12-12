package swapdata

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackVelocoreV2(chainID valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildVelocoreV2(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	return packVelocoreV2(swap)
}

func UnpackVelocoreV2(data []byte) (VelocoreV2, error) {
	unpacked, err := VelocoreV2Arguments.Unpack(data)
	if err != nil {
		return VelocoreV2{}, err
	}

	var swap VelocoreV2
	if err = VelocoreV2Arguments.Copy(&swap, unpacked); err != nil {
		return VelocoreV2{}, err
	}

	return swap, nil
}

func packVelocoreV2(swap VelocoreV2) ([]byte, error) {
	return VelocoreV2Arguments.Pack(
		swap.Vault,
		swap.Amount,
		swap.TokenIn,
		swap.TokenOut,
		swap.StablePool,
		swap.WrapToken,
		swap.IsConvertFirst,
	)
}

func buildVelocoreV2(chainID valueobject.ChainID, swap types.EncodingSwap) (VelocoreV2, error) {
	type Extra struct {
		Vault    string            `json:"vault"`
		Wrappers map[string]string `json:"wrappers"`
	}

	extraBytes, err := json.Marshal(swap.Extra)
	if err != nil {
		return VelocoreV2{}, err
	}

	var extra Extra
	if err := json.Unmarshal(extraBytes, &extra); err != nil {
		return VelocoreV2{}, err
	}

	var (
		vault    = common.HexToAddress(extra.Vault)
		amount   = swap.SwapAmount
		tokenIn  = common.HexToAddress(swap.TokenIn)
		tokenOut = common.HexToAddress(swap.TokenOut)

		stablePool     = common.HexToAddress(valueobject.ZeroAddress)
		wrapToken      = common.HexToAddress(valueobject.ZeroAddress)
		isConvertFirst = false
	)

	if swap.Exchange == valueobject.ExchangeVelocoreV2WombatStable {
		stablePool = common.HexToAddress(swap.Pool)
		if wToken, ok := extra.Wrappers[swap.TokenIn]; ok {
			wrapToken = common.HexToAddress(wToken)
			isConvertFirst = true
		}
		if wToken, ok := extra.Wrappers[swap.TokenOut]; ok {
			wrapToken = common.HexToAddress(wToken)
			isConvertFirst = false
		}
	}

	return VelocoreV2{
		Vault:          vault,
		Amount:         amount,
		TokenIn:        tokenIn,
		TokenOut:       tokenOut,
		StablePool:     stablePool,
		WrapToken:      wrapToken,
		IsConvertFirst: isConvertFirst,
	}, nil
}
