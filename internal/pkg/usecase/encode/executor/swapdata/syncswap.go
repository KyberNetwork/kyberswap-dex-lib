package swapdata

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
)

type withdrawMode uint8

const (
	defaultMode   withdrawMode = 0
	unwrappedMode withdrawMode = 1
	wrappedMode   withdrawMode = 2
)

func PackSyncSwap(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildSyncSwap(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packSyncSwap(swap)
}

func UnpackSyncSwap(data []byte) (SyncSwap, error) {
	unpacked, err := SyncSwapABIArguments.Unpack(data)
	if err != nil {
		return SyncSwap{}, err
	}

	var swap SyncSwap
	if err = SyncSwapABIArguments.Copy(&swap, unpacked); err != nil {
		return SyncSwap{}, err
	}

	return swap, nil
}

func buildSyncSwap(swap types.EncodingSwap) (SyncSwap, error) {
	// _data encode of (address, address, uint8) : (tokenIn, recipient, withdrawMode)
	// withdrawMode: always using 0 (DEFAULT)
	data, err := SyncSwapDataABIArguments.Pack(common.HexToAddress(swap.TokenIn), common.HexToAddress(swap.Recipient), defaultMode)
	if err != nil {
		return SyncSwap{}, err
	}

	return SyncSwap{
		Data:          data,
		TokenIn:       common.HexToAddress(swap.TokenIn),
		Pool:          common.HexToAddress(swap.Pool),
		CollectAmount: swap.CollectAmount,
	}, nil
}

func packSyncSwap(swap SyncSwap) ([]byte, error) {
	return SyncSwapABIArguments.Pack(
		swap.Data,
		swap.TokenIn,
		swap.Pool,
		swap.CollectAmount,
	)
}
