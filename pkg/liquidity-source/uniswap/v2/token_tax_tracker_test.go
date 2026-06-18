package uniswapv2

import (
	"context"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

const (
	wbnb           = "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c"
	pancakeFactory = "0xca143ce32fe78f1f7019d7d551a6402fc5350c73"
)

func poolWith(tokens ...string) entity.Pool {
	p := entity.Pool{Tokens: make([]*entity.PoolToken, len(tokens))}
	for i, t := range tokens {
		p.Tokens[i] = &entity.PoolToken{Address: t}
	}
	return p
}

// A pool already probed as non-tax must never hit the RPC client again (nil client would panic).
func TestResolveTokenTax_CheckedNonTaxSkipsProbing(t *testing.T) {
	t.Parallel()
	prev := TokenTax{Checked: true}
	tax, err := resolveTokenTax(context.Background(), nil, poolWith("0xa", "0xb"), prev, pancakeFactory, nil)
	require.NoError(t, err)
	assert.Equal(t, prev, tax)
}

// four.meme rates are immutable, so a known four.meme pool reuses cached state without any RPC.
func TestResolveTokenTax_FourMemeCachedReused(t *testing.T) {
	t.Parallel()
	prev := TokenTax{Token: "0xagent", BuyTax: uint256.NewInt(100), Checked: true}
	tax, err := resolveTokenTax(context.Background(), nil, poolWith("0xagent", wbnb), prev, pancakeFactory, nil)
	require.NoError(t, err)
	assert.Equal(t, prev, tax)
}

// A WBNB pool on a non-four.meme factory is not a candidate; it is marked checked, no probing.
func TestResolveTokenTax_WrongFactorySkips(t *testing.T) {
	t.Parallel()
	tax, err := resolveTokenTax(context.Background(), nil, poolWith("0xagent", wbnb), TokenTax{}, "0xotherfactory", nil)
	require.NoError(t, err)
	assert.True(t, tax.Checked)
	assert.Empty(t, tax.Token)
}

// A pool unrelated to any tax protocol is marked checked so later runs skip it.
func TestResolveTokenTax_UnrelatedPoolMarkedChecked(t *testing.T) {
	t.Parallel()
	tax, err := resolveTokenTax(context.Background(), nil, poolWith("0xa", "0xb"), TokenTax{}, pancakeFactory, nil)
	require.NoError(t, err)
	assert.True(t, tax.Checked)
	assert.Empty(t, tax.Token)
}

func TestPairedToken(t *testing.T) {
	t.Parallel()
	virtual := "0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b"
	assert.Equal(t, "0xagent", pairedToken(poolWith(virtual, "0xAgent"), virtualBaseTokens))
	assert.Equal(t, "0xagent", pairedToken(poolWith("0xAgent", wbnb), fourMemeBaseTokens))
	assert.Empty(t, pairedToken(poolWith("0xa", "0xb"), virtualBaseTokens))
	assert.Empty(t, pairedToken(poolWith(virtual), virtualBaseTokens)) // single-token pool
}
