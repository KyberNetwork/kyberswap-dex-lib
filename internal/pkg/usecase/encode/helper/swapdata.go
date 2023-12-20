package helper

import (
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

// IsUniV3Type returns true if poolType is UniswapV3 or PancakeV3 type
// otherwise, it returns false
// This function should be updated whenever we support new UniV3-like pool
func IsUniV3Type(poolType string) bool {
	switch poolType {
	case constant.PoolTypes.UniV3:
		return true
	case constant.PoolTypes.PancakeV3:
		return true
	case constant.PoolTypes.RamsesV2:
		return true
	case constant.PoolTypes.SolidlyV3:
		return true
	default:
		return false
	}
}
