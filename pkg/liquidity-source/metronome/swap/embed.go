package metronomeswap

import (
	_ "embed"
)

//go:embed abi/pool.json
var poolJson []byte

//go:embed abi/pool_registry.json
var poolRegistryJson []byte

//go:embed abi/fee_provider.json
var feeProviderJson []byte

//go:embed abi/master_oracle.json
var masterOracleJson []byte

//go:embed abi/debt_token.json
var debtTokenJson []byte

//go:embed abi/synthetic_token.json
var syntheticTokenJson []byte
