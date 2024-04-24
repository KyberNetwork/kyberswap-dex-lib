package stableng

import (
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	pools := []string{
		// https://arbiscan.io/address/0xdc40d14accd5629bbfa65d057f175871628d13c7#readContract
		"{\"address\":\"0xdc40d14accd5629bbfa65d057f175871628d13c7\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1709285278,\"reserves\":[\"50980\",\"75958\",\"100000000000000\"],\"tokens\":[{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"symbol\":\"USDT\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"symbol\":\"USDC.e\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"20000\\\",\\\"FutureA\\\":\\\"20000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"20000000000\\\"}\",\"blockNumber\":185969597}",

		// https://arbiscan.io/address/0x3adf984c937fa6846e5a24e0a68521bdaf767ce1#readContract
		"{\"address\":\"0x3adf984c937fa6846e5a24e0a68521bdaf767ce1\",\"exchange\":\"curve-stable-ng\",\"type\":\"curve-stable-ng\",\"timestamp\":1709287180,\"reserves\":[\"8994725349517509957774712\",\"1568153728639\",\"10550045569550900254909685\"],\"tokens\":[{\"address\":\"0x498bf2b1e120fed3ad3d42ea2165e9b73f99c1e5\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"symbol\":\"USDC.e\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"100000\\\",\\\"FutureA\\\":\\\"100000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1000000000000000000000000000000\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"OffpegFeeMultiplier\\\":\\\"50000000000\\\"}\",\"blockNumber\":185977087}",
	}
	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		require.Nil(t, err)
		p, err := NewPoolSimulator(poolEntity)
		require.Nil(t, err)
		return p
	})
	for _, pool := range sims {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(PoolSimulator)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(PoolSimulator{})...))
	}
}
