package gyro2clp

import (
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	rawPools := []string{
		`{"address":"0xdac42eeb17758daa38caf9a3540c808247527ae3","exchange":"gyroscope-2clp","type":"gyroscope-2clp","timestamp":1702978154,"reserves":["41488841728","42841512988282624073636"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063","weight":1,"swappable":true}],"extra":"{\"swapFeePercentage\":\"0xb5e620f48000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0xdac42eeb17758daa38caf9a3540c808247527ae3000200000000000000000a2b\",\"poolType\":\"Gyro2\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\"],\"sqrtParameters\":[\"0xdd7d21d9fd0cd67\",\"0xde9959a7b067d3c\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51305088}`,
	}
	var err error
	poolEntities := make([]*entity.Pool, len(rawPools))
	for i, rawPool := range rawPools {
		poolEntities[i] = new(entity.Pool)
		err = json.Unmarshal([]byte(rawPool), poolEntities[i])
		require.NoError(t, err)
	}
	pools := make([]*PoolSimulator, len(poolEntities))
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(*poolEntity)
		require.NoError(t, err)
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(PoolSimulator)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(PoolSimulator{})...))
	}
}
