package pooltypes

import (
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	_ "github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestFactory(t *testing.T) {
	excludedPoolTypes := []string{
		"curve-lending", // not implemented
		"maverick-v2",   // aevm
		"uniswap-v4",    // aevm
	}
	var poolTypesMap map[string]string
	assert.NoError(t, mapstructure.Decode(PoolTypes, &poolTypesMap))
	poolTypes := lo.OmitByValues(poolTypesMap, excludedPoolTypes)

	for _, poolType := range poolTypes {
		t.Run(poolType, func(t *testing.T) {
			got := pool.Factory(poolType)
			assert.NotNil(t, got)
		})
	}
}
