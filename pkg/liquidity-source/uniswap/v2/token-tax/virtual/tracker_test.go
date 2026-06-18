package virtual

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
)

func TestTrackerState(t *testing.T) {
	t.Parallel()

	t.Run("registered pool", func(t *testing.T) {
		tracker := tracker{
			tokenAddress: "0xagent", isLiquidityPool: true,
			buyTaxBps: big.NewInt(100), sellTaxBps: big.NewInt(1000),
		}
		result := tracker.TaxResult()
		assert.Equal(t, tokentax.Result{
			Protocol:     Protocol,
			TokenAddress: "0xagent",
			BuyTaxBps:    uint256.NewInt(100),
			SellTaxBps:   uint256.NewInt(1000),
			Checked:      true,
		}, result)
	})

	t.Run("unregistered pool keeps protocol and token", func(t *testing.T) {
		tracker := tracker{
			tokenAddress: "0xagent", buyTaxBps: big.NewInt(100), sellTaxBps: big.NewInt(1000),
		}
		result := tracker.TaxResult()
		assert.Equal(t, tokentax.Result{
			Protocol:     Protocol,
			TokenAddress: "0xagent",
			Checked:      true,
		}, result)
	})

	t.Run("reverted reads are unsupported", func(t *testing.T) {
		result := (&tracker{tokenAddress: "0xagent"}).TaxResult()
		assert.Equal(t, tokentax.Result{Checked: true}, result)
	})
}

func TestNewTracker(t *testing.T) {
	t.Parallel()

	pool := entity.Pool{Tokens: []*entity.PoolToken{
		{Address: "0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b"},
		{Address: "0xAgent"},
	}}
	assert.IsType(t, &tracker{}, NewTracker(pool))
	assert.True(t, SupportsFactory(factoryBase))
	assert.False(t, SupportsFactory("0xother"))
}
