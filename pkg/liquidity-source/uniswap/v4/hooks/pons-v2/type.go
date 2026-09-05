package ponsv2

import "github.com/ethereum/go-ethereum/common"

// launchRaw is the ethrpc.Call.AddCall decode target for PonsV2MemeHook.sol's
// `launches` public mapping getter, field order matching hookABI's outputs.
// Only registered/hookFeeBps/creatorTaxBps drive the swap-visible fee; the
// remaining fields govern fee distribution at sweep time, not pricing, so
// they aren't decoded (abi.Unpack fills struct fields positionally and skips
// absent ones, so every output must still be kept in the struct).
type launchRaw struct {
	Registered                bool
	MemecoinIsCurrency0       bool
	Memecoin                  common.Address
	QuoteToken                common.Address
	Creator                   common.Address
	BuybackCreatorRecipient   common.Address
	ProtocolFeeRecipient      common.Address
	CreatorTaxBps             uint16
	ProtocolFeeShareBps       uint16
	BuybackBurnBps            uint16
	HookFeeBps                uint16
	MaxInternalPriceImpactBps uint16
	BuybackEnabled            bool
}
