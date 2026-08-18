package parityswapprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	pmmRegistryABI abi.ABI
	// PmmPoolABI is exported so aggregator-encoding can decode-verify its
	// packed calldata against the real contract ABI.
	PmmPoolABI   abi.ABI
	pmmOracleABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&pmmRegistryABI, pmmRegistryBytes},
		{&PmmPoolABI, pmmPoolBytes},
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
