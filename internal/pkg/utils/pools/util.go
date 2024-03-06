package pools

import (
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// IsTheSameReserve return true if the two pools inPut got the same reserve
func IsTheSameReserve(pool1, pool2 poolpkg.IPoolSimulator) bool {
	oldR := pool1.GetReserves()
	newR := pool2.GetReserves()
	if len(oldR) != len(newR) {
		return false
	}
	for i := 0; i < len(oldR); i++ {
		if oldR[i].Cmp(newR[i]) != 0 {
			return false
		}
	}
	return true
}
