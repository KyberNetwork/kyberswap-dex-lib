package tickermemev4

import "github.com/ethereum/go-ethereum/common"

type Extra struct {
	ImplementationHash common.Hash `json:"impl"`
	Registered         bool        `json:"r"`
	HookFeeBps         uint16      `json:"f"`
	CreatorTaxBps      uint16      `json:"t"`
}

// The getter returns all thirteen LaunchInfo fields in Solidity declaration order.
type launchInfo struct {
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
