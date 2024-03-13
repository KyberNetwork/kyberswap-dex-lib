package shared

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/KyberNetwork/logger"
)

type (
	CurveCoin struct {
		Address           string
		Decimals          interface{}
		IsBasePoolLpToken bool
		Symbol            string
		IsOrgNative       bool // if this is an native coin (we'll convert native to wrapped, so need this to track the original data)
	}

	CurvePool struct {
		Id             string
		Address        string
		Coins          []CurveCoin
		Name           string
		Implementation string
		LpTokenAddress string
		IsMetaPool     bool

		BasePoolAddress string
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

func (c *CurveCoin) GetDecimals() uint8 {
	switch v := c.Decimals.(type) {
	case float64:
		return uint8(v)
	case string:
		dec, err := strconv.ParseInt(v, 10, 0)
		if err != nil {
			logger.Errorf("curve coin with invalid decimal %v %v", c.Address, c.Decimals)
			return 0
		}
		return uint8(dec)
	default:
		logger.Errorf("curve coin with invalid decimal %v %v", c.Address, c.Decimals)
		return 0
	}
}

func (m PoolListUpdaterMetadata) ToBytes() []byte {
	b, _ := json.Marshal(m)
	return b
}
