package uniswapv2

import (
	"testing"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestExtraWithoutTaxInfo(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(Extra{Fee: 30, FeePrecision: 10000})
	require.NoError(t, err)
	assert.JSONEq(t, `{"fee":30,"feePrecision":10000}`, string(encoded))
}

func TestNewTokenTaxTracker(t *testing.T) {
	t.Parallel()

	const (
		weth  = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
		agent = "0x0000000000000000000000000000000000000001"
	)

	poolWith := func(baseToken string) entity.Pool {
		return entity.Pool{
			Address: "0x0000000000000000000000000000000000000002",
			Tokens: []*entity.PoolToken{
				{Address: baseToken},
				{Address: agent},
			},
		}
	}

	t.Run("unsupported chain is not tracked", func(t *testing.T) {
		tracker, info := newTokenTaxTracker(valueobject.ChainID(999999), poolWith(weth), Extra{})
		assert.Nil(t, tracker)
		assert.Equal(t, tokentax.TaxInfo{}, info)
	})

	t.Run("pool with unusual pairing is still tracked (no base-token allowlist needed)", func(t *testing.T) {
		// The self-pair probe needs no recognized base token on either side: it works for any
		// 2-token pool that lives on the same factory a detector instance is bound to.
		tracker, info := newTokenTaxTracker(valueobject.ChainIDEthereum, entity.Pool{
			Tokens: []*entity.PoolToken{{Address: agent}, {Address: "0x0000000000000000000000000000000000000003"}},
		}, Extra{})
		assert.NotNil(t, tracker)
		assert.Equal(t, tokentax.TaxInfo{}, info)
	})

	t.Run("pool without exactly 2 tokens is not tracked", func(t *testing.T) {
		tracker, info := newTokenTaxTracker(valueobject.ChainIDEthereum, entity.Pool{
			Tokens: []*entity.PoolToken{{Address: agent}},
		}, Extra{})
		assert.Nil(t, tracker)
		assert.Equal(t, tokentax.TaxInfo{}, info)
	})

	t.Run("reuses cached unsupported token once migrated", func(t *testing.T) {
		previous := tokentax.TaxInfo{Checked: true, TaxCheckVersion: currentTaxCheckVersion}
		tracker, info := newTokenTaxTracker(valueobject.ChainIDEthereum, poolWith(weth), Extra{TaxInfo: &previous})
		assert.Nil(t, tracker)
		assert.Equal(t, previous, info)
	})

	t.Run("stale TaxCheckVersion forces one recheck even if marked checked", func(t *testing.T) {
		previous := tokentax.TaxInfo{Checked: true}
		tracker, info := newTokenTaxTracker(valueobject.ChainIDEthereum, poolWith(weth), Extra{TaxInfo: &previous})
		assert.NotNil(t, tracker)
		assert.Equal(t, tokentax.TaxInfo{}, info)
	})

	t.Run("refreshes known mutable tax", func(t *testing.T) {
		previous := tokentax.TaxInfo{Token: agent, Checked: true, TaxCheckVersion: currentTaxCheckVersion}
		tracker, info := newTokenTaxTracker(valueobject.ChainIDEthereum, poolWith(weth), Extra{TaxInfo: &previous})
		assert.NotNil(t, tracker)
		assert.Equal(t, tokentax.TaxInfo{}, info)
	})

	t.Run("probes unchecked state", func(t *testing.T) {
		tracker, info := newTokenTaxTracker(valueobject.ChainIDEthereum, poolWith(weth), Extra{})
		assert.NotNil(t, tracker)
		assert.Equal(t, tokentax.TaxInfo{}, info)
	})
}
