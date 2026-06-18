package fourmeme

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
)

func TestTrackerState(t *testing.T) {
	t.Parallel()
	poolAddress := "0x9053a8607902b8a3e971f2fae2562c4e2aa64b05"

	t.Run("canonical pair", func(t *testing.T) {
		tracker := tracker{
			poolAddress:  poolAddress,
			tokenAddress: "0xagent",
			pairAddress:  common.HexToAddress(poolAddress),
			buyTaxPct:    big.NewInt(1),
			sellTaxPct:   big.NewInt(10),
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

	t.Run("different pair is unsupported", func(t *testing.T) {
		tracker := tracker{poolAddress: poolAddress, pairAddress: common.HexToAddress("0xdead")}
		result := tracker.TaxResult()
		assert.Equal(t, tokentax.Result{Checked: true}, result)
	})
}

func TestNewTracker(t *testing.T) {
	t.Parallel()

	pool := entity.Pool{Tokens: []*entity.PoolToken{
		{Address: "0xAgent"},
		{Address: "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"},
	}}
	assert.IsType(t, &tracker{}, NewTracker(pool, tokentax.Result{}))
	assert.True(t, SupportsFactory(factory))
	assert.False(t, SupportsFactory("0xother"))

	previous := tokentax.Result{TokenAddress: "0xagent", BuyTaxBps: uint256.NewInt(100), Checked: true}
	cachedTracker := NewTracker(pool, previous)
	cached := cachedTracker.TaxResult()
	assert.Equal(t, Protocol, cached.Protocol)
	assert.Equal(t, previous.TokenAddress, cached.TokenAddress)
	assert.Equal(t, previous.BuyTaxBps, cached.BuyTaxBps)
}
