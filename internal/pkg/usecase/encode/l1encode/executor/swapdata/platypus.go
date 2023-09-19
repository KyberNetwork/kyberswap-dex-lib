package swapdata

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackPlatypus(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap := buildPlatypus(encodingSwap)

	return packPlatypus(swap)
}

func UnpackPlatypus(encodedSwap []byte) (Platypus, error) {
	unpacked, err := PlatypusABIArguments.Unpack(encodedSwap)
	if err != nil {
		return Platypus{}, err
	}

	var swap Platypus
	if err = PlatypusABIArguments.Copy(&swap, unpacked); err != nil {
		return Platypus{}, err
	}

	return swap, nil
}

func buildPlatypus(swap types.EncodingSwap) Platypus {
	return Platypus{
		Pool:              common.HexToAddress(swap.Pool),
		TokenIn:           common.HexToAddress(swap.TokenIn),
		TokenOut:          common.HexToAddress(swap.TokenOut),
		Recipient:         common.HexToAddress(swap.Recipient),
		CollectAmount:     swap.CollectAmount,
		LimitReturnAmount: swap.LimitReturnAmount,
	}
}

func packPlatypus(swap Platypus) ([]byte, error) {
	return PlatypusABIArguments.Pack(
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.Recipient,
		swap.CollectAmount,
		swap.LimitReturnAmount,
	)
}
