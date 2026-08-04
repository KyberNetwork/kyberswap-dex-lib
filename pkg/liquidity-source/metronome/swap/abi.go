package metronomeswap

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI           abi.ABI
	poolRegistryABI   abi.ABI
	feeProviderABI    abi.ABI
	masterOracleABI   abi.ABI
	debtTokenABI      abi.ABI
	syntheticTokenABI abi.ABI
)

// PoolABI is exported for aggregator-encoding's swapdata packer, which calls
// PoolABI.Pack("swap", ...) to build the on-chain calldata directly against the same ABI the
// simulator/tracker parse — no separate copy to drift.
var PoolABI abi.ABI

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&poolABI, poolJson},
		{&poolRegistryABI, poolRegistryJson},
		{&feeProviderABI, feeProviderJson},
		{&masterOracleABI, masterOracleJson},
		{&debtTokenABI, debtTokenJson},
		{&syntheticTokenABI, syntheticTokenJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	PoolABI = poolABI
}
