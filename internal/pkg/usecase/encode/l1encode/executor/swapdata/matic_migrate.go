package swapdata

import (
	"encoding/json"

	"github.com/KyberNetwork/blockchain-toolkit/account"
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

	var (
		tokenAddress common.Address
		recipient    common.Address
	)
	if extra.IsMigrate {
		tokenAddress = common.HexToAddress(swap.TokenOut)
		recipient = account.ZeroAddress
	} else {
		tokenAddress = common.HexToAddress(swap.TokenIn)
		recipient = common.HexToAddress(swap.Recipient)
	}

	return MaticMigrate{
		Pool:         common.HexToAddress(swap.Pool),
		TokenAddress: tokenAddress,
		Amount:       swap.SwapAmount,
		Recipient:    recipient,
	}, nil
}

func packMaticMigrate(swap MaticMigrate) ([]byte, error) {
	return MaticMigrateArguments.Pack(
		swap.Pool,
		swap.TokenAddress,
		swap.Amount,
		swap.Recipient,
	)
}
