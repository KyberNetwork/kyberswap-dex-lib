package reafund

import (
	"github.com/ethereum/go-ethereum/common"
)

// HookAddresses is REA.fund's PonsV2MemeHook on Robinhood Chain (4663), shared by every graduated
// REA.fund pool. It was built from Pons' source and constructed with REA.fund's fee escrow
// 0x11a21c0639f38375f70B663d73fA2085e30d20c2; Sourcify reports an exact match.
var HookAddresses = []common.Address{
	common.HexToAddress("0xe6dA98Faca06d7db50a42f93cA79Fd6D1fFBa044"),
}
