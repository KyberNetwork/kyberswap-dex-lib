package hyeth

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	hyethABI              abi.ABI
	poolABI               abi.ABI
	issuanceModuleABI     abi.ABI
	hyethComponent4626ABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&hyethABI, hyethABIData},
		{&poolABI, poolABIData},
		{&issuanceModuleABI, issuanceModuleABIData},
		{&hyethComponent4626ABI, hyethComponent4626ABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
