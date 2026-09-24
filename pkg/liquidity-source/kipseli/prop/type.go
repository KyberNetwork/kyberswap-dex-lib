package prop

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
)

// Extra is ladder.Extra plus SO, the Titan state override the ladder was
// probed against — pamm-variant only. A pool only ever persists non-empty
// ladders for the pamm variant when SO is also non-empty (the venue reverts
// every quote() without a fresh override), so SO's presence doubles as the
// variant discriminator downstream (see PoolMetaInfo) — no separate flag
// needed. ladder.NewPoolSimulatorWith only reads the "l" field, so SO parses
// along for free without any stripping.
type Extra struct {
	Ladders [2][]ladder.Point    `json:"l"`
	SO      titan.StateOverrides `json:"so,omitempty"`
}

type StaticExtra struct {
	RouterAddress string `json:"routerAddress"`
}

// PoolMetaInfo.ApprovalAddress duplicates RouterAddress under the
// "approvalAddress" JSON key: the private repo's kipseli-prop RFQHandler
// reads pool.ApprovalInfo.ApprovalAddress from this same JSON (its own field
// tag), so RouterAddress alone left it unset -- executor approve()/swap()
// calls need the same address either way.
type PoolMetaInfo struct {
	BlockNumber     uint64               `json:"blockNumber"`
	RouterAddress   string               `json:"routerAddress"`
	ApprovalAddress string               `json:"approvalAddress"`
	SO              titan.StateOverrides `json:"so,omitempty"`
}
