package uniswapv2

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tokentax "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax"
	fourmeme "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2/token-tax/four-meme"
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
		virtualFactory  = "0x8909dc15e40173ff4699343b6eb8132c65e18ec6"
		virtualToken    = "0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b"
		fourMemeFactory = "0xca143ce32fe78f1f7019d7d551a6402fc5350c73"
		wbnb            = "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
		agent           = "0x0000000000000000000000000000000000000001"
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

	t.Run("ordinary factory is not tracked", func(t *testing.T) {
		tracker, info := newTokenTaxTracker("0xother", entity.Pool{}, Extra{})
		assert.Nil(t, tracker)
		assert.Equal(t, tokentax.TaxInfo{}, info)
	})

	t.Run("virtual reuses cached unsupported token", func(t *testing.T) {
		previous := tokentax.TaxInfo{Checked: true}
		p := poolWith(virtualToken)
		tracker, info := newTokenTaxTracker(virtualFactory, p, Extra{TaxInfo: &previous})
		assert.Nil(t, tracker)
		assert.Equal(t, previous, info)
	})

	t.Run("virtual refreshes known mutable tax", func(t *testing.T) {
		previous := tokentax.TaxInfo{Token: agent, BuyTaxBps: uint256.NewInt(100), Checked: true}
		p := poolWith(virtualToken)
		tracker, info := newTokenTaxTracker(virtualFactory, p, Extra{TaxInfo: &previous})
		assert.NotNil(t, tracker)
		assert.Equal(t, tokentax.TaxInfo{}, info)
	})

	t.Run("four meme refreshes tax but reuses canonical pair", func(t *testing.T) {
		previous := tokentax.TaxInfo{
			Protocol: fourmeme.Protocol, Token: agent, BuyTaxBps: uint256.NewInt(100), Checked: true,
		}
		p := poolWith(wbnb)
		tracker, info := newTokenTaxTracker(fourMemeFactory, p, Extra{TaxInfo: &previous})
		assert.NotNil(t, tracker)
		assert.Equal(t, tokentax.TaxInfo{}, info)
	})

	t.Run("four meme reuses cached unsupported pair", func(t *testing.T) {
		previous := tokentax.TaxInfo{Checked: true}
		tracker, info := newTokenTaxTracker(
			fourMemeFactory, poolWith(wbnb), Extra{TaxInfo: &previous},
		)
		assert.Nil(t, tracker)
		assert.Equal(t, previous, info)
	})

	t.Run("four meme probes unchecked state", func(t *testing.T) {
		tracker, info := newTokenTaxTracker(fourMemeFactory, poolWith(wbnb), Extra{})
		assert.NotNil(t, tracker)
		assert.Equal(t, tokentax.TaxInfo{}, info)
	})
}
