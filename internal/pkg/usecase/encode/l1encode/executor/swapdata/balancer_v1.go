package swapdata

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackBalancerV1(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap := BalancerV1{
		Pool:     common.HexToAddress(encodingSwap.Pool),
		TokenIn:  common.HexToAddress(encodingSwap.TokenIn),
		TokenOut: common.HexToAddress(encodingSwap.TokenOut),
		Amount:   encodingSwap.SwapAmount,
	}

	return packBalancerV1(swap)
}

func UnpackBalancerV1(encodedSwap []byte) (BalancerV1, error) {
	unpacked, err := BalancerV1Arguments.Unpack(encodedSwap)
	if err != nil {
		return BalancerV1{}, err
	}

	var swap BalancerV1
	if err = BalancerV1Arguments.Copy(&swap, unpacked); err != nil {
		return BalancerV1{}, err
	}

	return swap, nil
}

func packBalancerV1(swap BalancerV1) ([]byte, error) {
	return BalancerV1Arguments.Pack(
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.Amount,
	)
}
