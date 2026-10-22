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
	// executeWHLP -> Depositor.deposit (hyperevm LHYPE), steady-state on an anvil fork, approve
	// excluded: WHYPE 175999, stHYPE 234397. Integer average (175999+234397)/2.
	simpleTellerGas int64 = 205198
)
