package swapdata

import (
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

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
	encodedSwapStr := hex.EncodeToString(data)
	packedEncodedSwapDataStr := strings.Replace(encodedSwapStr, OffsetToTheStartOfData, "", 1)
	packedEncodedSwapBytes := common.Hex2Bytes(packedEncodedSwapDataStr)

	unpacked, err := SyncSwapABIArguments.Unpack(packedEncodedSwapBytes)
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

	byteData, err := json.Marshal(swap.Extra)
	if err != nil {
		return SyncSwap{}, errors.Wrapf(
			ErrMarshalFailed,
			"[BuildSyncSwap] err :[%v]",
			err,
		)
	}
	var extra struct {
		VaultAddress string `json:"vaultAddress"`
	}
	if err := json.Unmarshal(byteData, &extra); err != nil {
		return SyncSwap{}, err
	}

	return SyncSwap{
		Data:          data,
		Vault:         common.HexToAddress(extra.VaultAddress),
		TokenIn:       common.HexToAddress(swap.TokenIn),
		Pool:          common.HexToAddress(swap.Pool),
		CollectAmount: swap.CollectAmount,
	}, nil
}

func packSyncSwap(swap SyncSwap) ([]byte, error) {
	encoded, err := SyncSwapABIArguments.Pack(
		swap.Data,
		swap.Vault,
		swap.TokenIn,
		swap.Pool,
		swap.CollectAmount,
	)
	if err != nil {
		return nil, err
	}

	return hex.DecodeString(OffsetToTheStartOfData + common.Bytes2Hex(encoded))
}
