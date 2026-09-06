package ponsv2

import (
	"github.com/ethereum/go-ethereum/common"
)

// HookAddresses is the singleton PonsV2MemeHook shared by every graduated pons-v2
// pool on Robinhood chain. Verified via Sourcify: PonsV2MemeHook.sol.
var HookAddresses = []common.Address{
	common.HexToAddress("0xE5e702641Ea86F4ae6cC3cDaeD2B886f976Be044"),
}

const basisPoints = 10_000
