package ezeth

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	RestakeManagerABI abi.ABI
	RenzoOracleABI    abi.ABI
	PriceFeedABI      abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&RestakeManagerABI, restakeManagerABIJson,
		},
		{
			&RenzoOracleABI, renzoOracleABIJson,
		},
		{
			&PriceFeedABI, priceFeedABIJson,
		},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
