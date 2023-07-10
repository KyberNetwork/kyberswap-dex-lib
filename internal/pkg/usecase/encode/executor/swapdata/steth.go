package swapdata

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackStETH(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	return StETHABIArguments.Pack(encodingSwap.SwapAmount)
}

func UnpackStETH(encodedSwap []byte) (*big.Int, error) {
	unpacked, err := StETHABIArguments.Unpack(encodedSwap)
	if err != nil {
		return nil, err
	}

	amount := new(big.Int)
	if err = StETHABIArguments.Copy(&amount, unpacked); err != nil {
		return nil, err
	}

	return amount, nil
}
