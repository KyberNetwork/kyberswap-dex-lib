package integral

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	oracleABI  abi.ABI
	pairABI    abi.ABI
	factoryABI abi.ABI
	relayerABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&pairABI, twapPairJSON,
		},
		{
			&factoryABI, twapFactoryJSON,
		},
		{
			&oracleABI, twapOracleJSON,
		},
		{
			&relayerABI, twapRelayerJSON,
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
