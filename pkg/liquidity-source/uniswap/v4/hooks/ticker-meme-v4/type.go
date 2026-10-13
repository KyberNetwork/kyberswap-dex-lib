package tickermemev4

import "github.com/ethereum/go-ethereum/common"

type Extra struct {
	// Old snapshots contain a combined rate; require a refresh before using two rates.
	ImplementationHash common.Hash `json:"implementationV2"`
	Registered         bool        `json:"r"`
	HookFeeBps         uint16      `json:"f"`
	CreatorTaxBps      uint16      `json:"t"`
}
