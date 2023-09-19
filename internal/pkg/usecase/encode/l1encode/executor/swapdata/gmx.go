package swapdata

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackGMX(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap := buildGMX(encodingSwap)

	return packGMX(swap)
}

func UnpackGMX(encodedSwap []byte) (GMX, error) {
	unpacked, err := GMXABIArguments.Unpack(encodedSwap)
	if err != nil {
		return GMX{}, err
	}

	var swap GMX
	if err = GMXABIArguments.Copy(&swap, unpacked); err != nil {
		return GMX{}, err
	}

	return swap, nil
}

func buildGMX(swap types.EncodingSwap) GMX {
	return GMX{
		Vault:    common.HexToAddress(swap.Pool),
		TokenIn:  common.HexToAddress(swap.TokenIn),
		TokenOut: common.HexToAddress(swap.TokenOut),
		Amount:   swap.SwapAmount,
		Receiver: common.HexToAddress(swap.Recipient),
	}
}

func packGMX(swap GMX) ([]byte, error) {
	return GMXABIArguments.Pack(
		swap.Vault,
		swap.TokenIn,
		swap.TokenOut,
		swap.Amount,
		swap.Receiver,
	)
}
