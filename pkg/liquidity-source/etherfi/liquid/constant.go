package liquid

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	DexType          = "etherfi-liquid"
	unlimitedReserve = "10000000000000000000000000"
)

var (
	liquidReferAddress = common.HexToAddress("0x1d536713e681b3679f6201f0ad0f83d79eff3ede")

	defaultGas int64 = 204000
	// executeGeneric -> teller.deposit on hyperevm LHYPE, measured on an anvil fork (steady state,
	// executor already holds 1 wei of each token; identical for 3 amounts, approve excluded).
	tellerDirectGas int64 = 106413
)
