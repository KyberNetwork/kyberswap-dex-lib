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
	// executeGeneric -> teller.deposit on hyperevm LHYPE, steady state on an anvil fork (approve
	// excluded): WHYPE 106413, stHYPE 154853. Integer average (106413+154853)/2.
	tellerDirectGas int64 = 130633
)
