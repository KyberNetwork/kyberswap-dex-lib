package prop

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

// Extra holds the probed ladders. Titan state overrides are not stored:
// router-service fetches its own snapshot for build-time simulation.
type Extra struct {
	Ladders [2][]ladder.Point `json:"l"`
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
	BlockNumber     uint64 `json:"blockNumber"`
	RouterAddress   string `json:"routerAddress"`
	ApprovalAddress string `json:"approvalAddress"`
}
