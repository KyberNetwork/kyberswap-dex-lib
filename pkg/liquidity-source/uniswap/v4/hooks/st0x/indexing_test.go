package st0x

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The price is valid as of the snapshot the tracker wrote. Indexing must quote at that time;
// route finding, which quotes now, must still refuse the same state as stale.
func TestIndexingQuotesAgedState(t *testing.T) {
	const token0 = "0x0000000000000000000000000000000000000001"
	const token1 = "0x0000000000000000000000000000000000000002"
	ts := time.Now().Add(-24 * time.Hour).Unix()

	p := entity.Pool{
		Address:   "0xe26507ae709887aa01dab0e959ee4b3a8a1c025e5c29e1b2644b62ea950a4253",
		Timestamp: ts,
		Reserves:  entity.PoolReserves{"1000", "2000"},
		Tokens:    []*entity.PoolToken{{Address: token0, Decimals: 18}, {Address: token1, Decimals: 18}},
		Extra: fmt.Sprintf(`{"liquidity":0,"sqrtPriceX96":79228162514264337593543950336,`+
			`"tick":0,"tickSpacing":1,"ticks":[],"hX":{"p":1000000000000000000,"u":%d,"ms":60}}`, ts),
		StaticExtra: `{"hooks":"0x9af1a97021b92c47219d2fb24deba51c248a2a88","fee":0,"tS":1}`,
	}
	quote := func(sim pool.IPoolSimulator) error {
		_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(10)},
			TokenOut:      token1,
		})
		return err
	}

	indexed, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainID(8453))
	require.NoError(t, err)
	require.NoError(t, quote(indexed))

	routed, err := pool.Factory(uniswapv4.DexType)(pool.FactoryParams{
		EntityPool: p, Opts: pool.FactoryOpts{StaleCheck: true},
	})
	require.NoError(t, err)
	require.ErrorIs(t, quote(routed), ErrStalePrice)
}
