package cloberob

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob/libraries"
)

type IBookManager interface {
	GetDepth(tick cloberlib.Tick) (uint64, error)
	MaxLessThan(tick cloberlib.Tick) (cloberlib.Tick, error)
}
