package parityprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	pmmRegistryABI abi.ABI
	pmmPoolABI     abi.ABI
	pmmOracleABI   abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&pmmRegistryABI, pmmRegistryBytes},
		{&pmmPoolABI, pmmPoolBytes},
		{&pmmOracleABI, pmmOracleBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
