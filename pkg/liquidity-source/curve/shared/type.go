package shared

import (
	"time"

	"github.com/goccy/go-json"
)

type (
	CurveCoin struct {
		Address     string
		IsOrgNative bool // if this is an native coin (we'll convert native to wrapped, so need this to track the original data)
	}

	CurvePool struct {
		Id             string
		Address        string
		Coins          []CurveCoin
		Name           string
		Implementation string
		LpTokenAddress string
		IsMetaPool     bool

		// for meta pool
		BasePoolAddress string
		UnderlyingCoins []CurveCoin
	}

	GetPoolsResult struct {
		Success bool
		Data    struct {
			PoolData []CurvePool
		}
	}

	GetRegistryAddressResult struct {
		Success bool
		Data    struct {
			RegistryAddress string
		}
	}

	CurvePoolWithType struct {
		CurvePool
		PoolType CurvePoolType
	}

	PoolListUpdaterMetadata struct {
		LastRun time.Time
	}

	CurvePoolType   string
	CurveDataSource string
)

func (m PoolListUpdaterMetadata) ToBytes() []byte {
	b, _ := json.Marshal(m)
	return b
}
