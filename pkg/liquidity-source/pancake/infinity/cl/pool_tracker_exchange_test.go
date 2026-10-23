package cl

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestWithStateTakesTheHooksExchange: the hook names the pool's exchange, and the pool withState
// returns, bootstrapped or read through the lens, is under it.
func TestWithStateTakesTheHooksExchange(t *testing.T) {
	sqrtPriceOne, _ := new(big.Int).SetString("79228162514264337593543950336", 10)
	tracker := &PoolTracker{config: &Config{}}
	p := entity.Pool{Address: "0xpool", Exchange: DexType}
	result := &FetchRPCResult{Liquidity: big.NewInt(0), TickSpacing: 10,
		Slot0: Slot0Data{SqrtPriceX96: sqrtPriceOne, Tick: big.NewInt(0)}}

	got, err := tracker.withState(t.Context(), p, result, &BaseHook{Exchange: DexType + "-dynamic"},
		&HookParam{Cfg: tracker.config, Pool: &p}, nil, poolpkg.GetNewPoolStateParams{}, 0, logger.WithFields(nil))
	require.NoError(t, err)
	assert.Equal(t, DexType+"-dynamic", got.Exchange)
}
