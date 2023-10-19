package swapdata

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackVooi(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildVooi(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packVooi(swap)
}

func UnpackVooi(encodedSwap []byte) (Vooi, error) {
	unpacked, err := VooiArguments.Unpack(encodedSwap)
	if err != nil {
		return Vooi{}, err
	}

	var swap Vooi
	if err = VooiArguments.Copy(&swap, unpacked); err != nil {
		return Vooi{}, err
	}

	return swap, nil
}

func buildVooi(swap types.EncodingSwap) (Vooi, error) {
	byteData, err := json.Marshal(swap.PoolExtra)
	if err != nil {
		return Vooi{}, errors.Wrapf(
			ErrMarshalFailed,
			"[buildVooi] err :[%v]",
			err,
		)
	}

	var extra struct {
		FromID int64 `json:"fromId"`
		ToID   int64 `json:"toId"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return Vooi{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[buildVooi] err :[%v]",
			err,
		)
	}
	return Vooi{
		Pool:       common.HexToAddress(swap.Pool),
		FromToken:  common.HexToAddress(swap.TokenIn),
		ToToken:    common.HexToAddress(swap.TokenOut),
		FromID:     big.NewInt(extra.FromID),
		ToID:       big.NewInt(extra.ToID),
		FromAmount: swap.SwapAmount,
		To:         common.HexToAddress(swap.Recipient),
	}, nil
}

func packVooi(swap Vooi) ([]byte, error) {
	return VooiArguments.Pack(
		swap.Pool,
		swap.FromToken,
		swap.ToToken,
		swap.FromID,
		swap.ToID,
		swap.FromAmount,
		swap.To,
	)
}
