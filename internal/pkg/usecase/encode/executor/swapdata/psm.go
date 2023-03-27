package swapdata

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/types"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

func PackPSM(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap := buildPSM(encodingSwap)

	return packPSM(swap)
}

func UnpackPSM(encodedSwap []byte) (PSM, error) {
	unpacked, err := PSMABIArguments.Unpack(encodedSwap)
	if err != nil {
		return PSM{}, err
	}

	var swap PSM
	if err = PSMABIArguments.Copy(&swap, unpacked); err != nil {
		return PSM{}, err
	}

	return swap, nil
}

func buildPSM(swap types.EncodingSwap) PSM {
	return PSM{
		Router:    common.HexToAddress(swap.Pool),
		TokenIn:   common.HexToAddress(swap.TokenIn),
		TokenOut:  common.HexToAddress(swap.TokenOut),
		AmountIn:  swap.SwapAmount,
		Recipient: common.HexToAddress(swap.Recipient),
	}
}

func packPSM(swap PSM) ([]byte, error) {
	return PSMABIArguments.Pack(
		swap.Router,
		swap.TokenIn,
		swap.TokenOut,
		swap.AmountIn,
		swap.Recipient,
	)
}
