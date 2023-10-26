package swapdata

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func PackMaticMigrate(_ valueobject.ChainID, encodingSwap types.EncodingSwap) ([]byte, error) {
	swap, err := buildMaticMigrate(encodingSwap)
	if err != nil {
		return nil, err
	}

	return packMaticMigrate(swap)
}

func UnpackMaticMigrate(encodedSwap []byte) (MaticMigrate, error) {
	unpacked, err := MaticMigrateArguments.Unpack(encodedSwap)
	if err != nil {
		return MaticMigrate{}, err
	}

	var swap MaticMigrate
	if err = MaticMigrateArguments.Copy(&swap, unpacked); err != nil {
		return MaticMigrate{}, err
	}

	return swap, nil
}

func buildMaticMigrate(swap types.EncodingSwap) (MaticMigrate, error) {
	byteData, err := json.Marshal(swap.Extra)
	if err != nil {
		return MaticMigrate{}, errors.Wrapf(
			ErrMarshalFailed,
			"[buildMaticMigrate] err :[%v]",
			err,
		)
	}

	var extra struct {
		IsMigrate bool `json:"isMigrate"`
	}

	if err = json.Unmarshal(byteData, &extra); err != nil {
		return MaticMigrate{}, errors.Wrapf(
			ErrUnmarshalFailed,
			"[buildMaticMigrate] err :[%v]",
			err,
		)
	}

	return MaticMigrate{
		Pool:      common.HexToAddress(swap.Pool),
		TokenIn:   common.HexToAddress(swap.TokenIn),
		TokenOut:  common.HexToAddress(swap.TokenOut),
		Amount:    swap.SwapAmount,
		Recipient: common.HexToAddress(swap.Recipient),
		IsMigrate: extra.IsMigrate,
	}, nil
}

func packMaticMigrate(swap MaticMigrate) ([]byte, error) {
	return MaticMigrateArguments.Pack(
		swap.Pool,
		swap.TokenIn,
		swap.TokenOut,
		swap.Amount,
		swap.Recipient,
		swap.IsMigrate,
	)
}
