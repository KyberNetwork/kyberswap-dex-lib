package coocoo

import (
	"github.com/ethereum/go-ethereum/common"
)

// HookAddresses lists CooCoo's PonsV2MemeHook deployments, one per chain: the singleton
// hook shared by every graduated CooCoo pool on that chain. Each is the runtime code of
// Pons' own hook (hooks/pons-v2) apart from the three immutable references to the
// deployment's fee escrow and the compiler metadata hash; TestBytecodeParity_Live pins
// that against the chain. Verified on Blockscout (Robinhood) as PonsV2MemeHook.
var HookAddresses = []common.Address{
	common.HexToAddress("0x37966005085dD9F7Fb7F9d51E65daf5FeA552044"), // Robinhood Chain (4663), escrow 0xB1703f631084fec914cE2834F29EBEb8e07d3fb4
	common.HexToAddress("0x915bF3CB732Bb692D63EcA08f962D62221CFE044"), // Arc (5042), escrow 0x21524873EFaB3Af236A35eA287099eD1fBe4c4cD
}
