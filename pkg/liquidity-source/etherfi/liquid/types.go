package liquid

import "github.com/ethereum/go-ethereum/common"

type Extra struct {
	LiquidRefer common.Address `json:"liquidRefer"`
	Teller      common.Address `json:"teller"`
}
