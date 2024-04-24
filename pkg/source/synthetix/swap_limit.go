//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple AtomicLimits
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package synthetix

import (
	"math/big"
	"sync"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// AtomicLimits implement pool.SwapLimit for synthetic
// key is blockTimestamp, and the limit is its balance
// The balance is stored WITHOUT decimals
// DONOT directly modify it, use UpdateLimit if needed
type AtomicLimits struct {
	lock   *sync.RWMutex `msg:"-"`
	Limits map[string]*big.Int
}

func NewLimits(atomicMaxVolumePerBlocks map[string]*big.Int) pool.SwapLimit {
	return &AtomicLimits{
		lock:   &sync.RWMutex{},
		Limits: atomicMaxVolumePerBlocks,
	}
}

// GetLimit returns a copy of balance for the token in Inventory
func (i *AtomicLimits) GetLimit(blockTimeStamp string) *big.Int {
	i.lock.RLock()
	defer i.lock.RUnlock()
	balance, avail := i.Limits[blockTimeStamp]
	if !avail {
		return big.NewInt(0)
	}
	return big.NewInt(0).Set(balance)
}

// UpdateLimit will reduce the limit to reflect the change in inventory
// note this delta is amount without Decimal
func (i *AtomicLimits) UpdateLimit(blockTimeStamp, _ string, decreaseDelta, _ *big.Int) (*big.Int, *big.Int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	volLimit, avail := i.Limits[blockTimeStamp]
	if !avail {
		return big.NewInt(0), big.NewInt(0), pool.ErrTokenNotAvailable
	}
	if volLimit.Cmp(decreaseDelta) < 0 {
		return big.NewInt(0), big.NewInt(0), pool.ErrNotEnoughInventory
	}
	i.Limits[blockTimeStamp] = volLimit.Sub(volLimit, decreaseDelta)

	return big.NewInt(0).Set(i.Limits[blockTimeStamp]), big.NewInt(0), nil
}
