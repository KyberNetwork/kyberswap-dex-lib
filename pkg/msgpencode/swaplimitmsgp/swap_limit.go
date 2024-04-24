//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple SwapLimitEnum

package swaplimitmsgp

import (
	"fmt"

	kyberpmm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/kyber-pmm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/synthetix"
)

type SwapLimitEnum struct {
	Inventory    *kyberpmm.Inventory     `msg:",omitempty"`
	AtomicLimits *synthetix.AtomicLimits `msg:",omitempty"`
}

func (l *SwapLimitEnum) get() pool.SwapLimit {
	if l.Inventory != nil {
		return l.Inventory
	}
	if l.AtomicLimits != nil {
		return l.AtomicLimits
	}
	return nil
}

func (l *SwapLimitEnum) set(sl pool.SwapLimit) error {
	switch sl := sl.(type) {
	case *kyberpmm.Inventory:
		l.Inventory = sl
		return nil
	case *synthetix.AtomicLimits:
		l.AtomicLimits = sl
		return nil
	default:
		return fmt.Errorf("invalid pool.SwapLimit concrete type")
	}
}
