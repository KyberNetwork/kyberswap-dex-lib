package honey

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	honeyABI      abi.ABI
	assetVaultABI abi.ABI
	erc20ABI      abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&honeyABI, honeyFactoryABIData},
		{&assetVaultABI, assetVaultABIData},
		{&erc20ABI, erc20ABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
