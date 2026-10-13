package tickermemev4

import (
	"github.com/ethereum/go-ethereum/common"

	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
)

type Extra struct {
	ponsv2.Extra
	// A new key forces snapshots with separate f/t rates to be tracked again.
	ImplementationHash common.Hash `json:"implementation"`
}
