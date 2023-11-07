package swapdata

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
)

func PackSmardex(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildSmardex(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packSmardex(swap)
}

func UnpackSmardex(encodedSwap []byte) (Smardex, error) {
	unpacked, err := SmardexArguments.Unpack(encodedSwap)
	if err != nil {
		return Smardex{}, err
	}

	var swap Smardex
	if err = SmardexArguments.Copy(&swap, unpacked); err != nil {
		return Smardex{}, err
	}

	return swap, nil
}

func buildSmardex(swap types.EncodingSwap) (Smardex, error) {
	return Smardex{
		Pool:      common.HexToAddress(swap.Pool),
		TokenIn:   common.HexToAddress(swap.TokenIn),
		TokenOut:  common.HexToAddress(swap.TokenOut),
		Amount:    swap.SwapAmount,
		Recipient: common.HexToAddress(swap.Recipient),
	}, nil
}

func packSmardex(swap Smardex) ([]byte, error) {
	return SmardexArguments.Pack(
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.Amount,
		swap.Recipient,
	)
}
