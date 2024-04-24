package gyro3clp

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
		`{"address":"0x17f1ef81707811ea15d9ee7c741179bbe2a63887","exchange":"gyroscope-3clp","type":"gyroscope-3clp","timestamp":1703150040,"reserves":["23020440114","1126110825231923552925","19544825382"],"tokens":[{"address":"0x2791bca1f2de4661ed88a30c99a7a9449aa84174","weight":1,"swappable":true},{"address":"0x9c9e5fd8bbc25984b178fdce6117defa39d2db39","weight":1,"swappable":true},{"address":"0xc2132d05d31c914a87c6611c10748aeb04b58e8f","weight":1,"swappable":true}],"extra":"{\"poolTokenInfos\":[{\"cash\":\"0x55c200a32\",\"managed\":\"0x0\",\"lastChangeBlock\":51379111,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x3d0bed552856cc229d\",\"managed\":\"0x0\",\"lastChangeBlock\":51378988,\"assetManager\":\"0x0000000000000000000000000000000000000000\"},{\"cash\":\"0x48cf65e26\",\"managed\":\"0x0\",\"lastChangeBlock\":51379111,\"assetManager\":\"0x0000000000000000000000000000000000000000\"}],\"swapFeePercentage\":\"0x110d9316ec000\",\"paused\":false}","staticExtra":"{\"poolId\":\"0x17f1ef81707811ea15d9ee7c741179bbe2a63887000100000000000000000799\",\"poolType\":\"Gyro3\",\"poolTypeVersion\":0,\"scalingFactors\":[\"0xc9f2c9cd04674edea40000000\",\"0xde0b6b3a7640000\",\"0xc9f2c9cd04674edea40000000\"],\"root3Alpha\":\"0xddeeff45500c000\",\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}","blockNumber":51380313}`,
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
