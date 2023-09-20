package swapdata

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackWombat(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap := buildWombat(encodingSwap)

	return packWombat(swap)
}

func UnpackWombat(encodedSwap []byte) (Wombat, error) {
	unpacked, err := WombatABIArguments.Unpack(encodedSwap)
	if err != nil {
		return Wombat{}, err
	}

	var swap Wombat
	if err = WombatABIArguments.Copy(&swap, unpacked); err != nil {
		return Wombat{}, err
	}

	return swap, nil
}

func buildWombat(swap types.EncodingSwap) Wombat {
	return Wombat{
		Pool:      common.HexToAddress(swap.Pool),
		TokenIn:   common.HexToAddress(swap.TokenIn),
		TokenOut:  common.HexToAddress(swap.TokenOut),
		Amount:    swap.AmountOut,
		Recipient: common.HexToAddress(swap.Recipient),
	}
}

func packWombat(swap Wombat) ([]byte, error) {
	return WombatABIArguments.Pack(
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.Amount,
		swap.Recipient,
	)
}
