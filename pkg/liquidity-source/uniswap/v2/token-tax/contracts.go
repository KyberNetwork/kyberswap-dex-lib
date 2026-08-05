package tokentax

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

const methodValidate = "validate"

// amountToBorrow mirrors Uniswap's smart-order-router FeeOnTransferDetector fetcher: large enough
// to avoid bps rounding errors, small enough that most V2 pools hold this many raw token units.
const amountToBorrowRaw = 100000

// detectorInstance is one deployed FeeOnTransferDetector, bound to one specific V2 factory at
// deploy time, so a token only resolves through an instance whose factory has a pair for it.
type detectorInstance struct {
	address string
	basic   bool // true selects detectorBasicABI/tokenFeesBasic instead of detectorABI/tokenFees
}

// uniswapContracts is Uniswap's own FeeOnTransferDetector deployment per chain (5-field TokenFees).
var uniswapContracts = map[valueobject.ChainID]string{
	valueobject.ChainIDEthereum:        "0xbc708b192552e19a088b4c4b8772aeea83bcf760",
	valueobject.ChainIDOptimism:        "0x95adc98a949dcd94645a8cd56830d86e4cf34eff",
	valueobject.ChainIDBSC:             "0xcf6220e4496b091a6b391d48e770f1fbac63e740",
	valueobject.ChainIDPolygon:         "0xc988e19819a63c0e487c6ad8d6668ac773923bf2",
	valueobject.ChainIDBase:            "0xcf6220e4496b091a6b391d48e770f1fbac63e740",
	valueobject.ChainIDArbitrumOne:     "0x37324d81e318260dc4f0fcb68035028efde6f50e",
	valueobject.ChainIDAvalancheCChain: "0x8269d47c4910b8c87789aa0ec128c11a8614dfc8",
	valueobject.ChainIDUnichain:        "0x55e74a5c3310bbccdd0b655ade2309e0d0d25826",
	valueobject.ChainIDLinea:           "0xf025e0fe9e331a0ef05c2ad3c4e9c64b625cda6f",
	valueobject.ChainIDMegaETH:         "0x2103a00792b980dff0509952bead5cb2e3149022",
	valueobject.ChainIDMonad:           "0x5c834b6cac4173bfe288c5722a38e04b9e366e30",
}

// pancakeContracts is PancakeSwap's own FeeOnTransferDetector deployment per chain (2-field
// TokenFees: buyFeeBps, sellFeeBps only).
var pancakeContracts = map[valueobject.ChainID]string{
	valueobject.ChainIDEthereum:    "0xe9200516a475b9e6fd4d1c452858097f345a6760",
	valueobject.ChainIDBSC:         "0x003bd52f589f23346e03fa431209c29cd599d693",
	valueobject.ChainIDBase:        "0xd8b14f915b1b4b1c4ee4bf8321bea018e72e5cf3",
	valueobject.ChainIDArbitrumOne: "0xd8b14f915b1b4b1c4ee4bf8321bea018e72e5cf3",
	valueobject.ChainIDLinea:       "0xd8b14f915b1b4b1c4ee4bf8321bea018e72e5cf3",
}

// detectorsFor returns every detector instance available for chainID. Uniswap's and PancakeSwap's
// deployments are independent and equally valid; a chain may have either, both, or neither.
func detectorsFor(chainID valueobject.ChainID) []detectorInstance {
	var instances []detectorInstance
	if address, ok := uniswapContracts[chainID]; ok {
		instances = append(instances, detectorInstance{address: address})
	}
	if address, ok := pancakeContracts[chainID]; ok {
		instances = append(instances, detectorInstance{address: address, basic: true})
	}
	return instances
}
