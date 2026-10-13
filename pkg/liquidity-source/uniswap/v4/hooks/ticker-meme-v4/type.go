package tickermemev4

import "github.com/ethereum/go-ethereum/common"

type Extra struct {
	// Old snapshots contain a combined rate; require a refresh before using two rates.
	ImplementationHash common.Hash `json:"implementationV2"`
	Registered         bool        `json:"r"`
	HookFeeBps         uint16      `json:"f"`
	CreatorTaxBps      uint16      `json:"t"`
}

// launchInfo matches all thirteen outputs of launches(bytes32), in declaration order.
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
