package stablestable

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	gasBeforeSwap int64 = 13445
)

var HookAddresses = []common.Address{
	common.HexToAddress("0x4509b7Eb3F9641226804Fea4976963435d1c6080"),
	common.HexToAddress("0x0000113dCf4ADd69999Fad8F20F2b63F979bfcC0"),
	common.HexToAddress("0x3b64660a35a09AfDe554cE545bca9166D6A23CC0"), // Robinhood Chain (4663)
}

// legacyHookAddress is the only deployment whose feeConfig() still returns an
// explicit on-chain logK; every other (newer) hook address derives it from k
// off-chain (see deriveLogK). Any hook not equal to this is treated as V2.
var legacyHookAddress = HookAddresses[0]
