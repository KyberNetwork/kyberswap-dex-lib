package everlongflamm

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func TestListerListsEveryConfiguredFactorysPoolsOnce(t *testing.T) {
	w := testWorld()
	two := func(f string) *fakePool {
		return &fakePool{factory: common.HexToAddress(f), listed: true, version: 1,
			tokens: []tokenConfig{tokenCfg(usdc, 6), tokenCfg(usdt, 6)}}
	}
	pY, pZ := common.HexToAddress("0xa2"), common.HexToAddress("0xa3")
	w.add(pY, two(factoryB))
	w.add(common.HexToAddress("0xa4"), &fakePool{factory: common.HexToAddress(factoryA), noCount: true})
	many := two(factoryA)
	for k := range 15 {
		many.tokens = append(many.tokens, tokenCfg(common.BigToAddress(big.NewInt(int64(0x100+k))).Hex(), 18))
	}
	w.add(common.HexToAddress("0xa5"), many) // 17 tokens: past what the source routes
	twin := two(factoryA)
	twin.tokens[1] = twin.tokens[0]
	w.add(common.HexToAddress("0xa6"), twin)                     // a token twice
	w.add(pZ, two("0x00000000000000000000000000000000000000f3")) // a Factory not configured

	cfg := testConfig()
	cfg.Factories = []string{"0x00000000000000000000000000000000000000F1", factoryB}
	n := newNode(t, testBlk, w.respond, w.codeAt)
	u := NewPoolsListUpdater(cfg, n.client())
	pools, md, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, []string{poolX, "0x00000000000000000000000000000000000000a2"},
		lo.Map(pools, func(p entity.Pool, _ int) string { return p.Address }))
	x := pools[0]
	require.Equal(t, DexType, x.Exchange)
	require.Equal(t, DexType, x.Type)
	require.Equal(t, []string{cbBTC, usdc, usdt}, lo.Map(x.Tokens, func(t *entity.PoolToken, _ int) string {
		return t.Address
	}), "the pool's own order, lower case")
	require.True(t, lo.EveryBy(x.Tokens, func(t *entity.PoolToken) bool { return t.Swappable && t.Decimals == 0 }))
	require.Equal(t, entity.PoolReserves{"0", "0", "0"}, x.Reserves)
	require.JSONEq(t, `{"factory":"`+factoryA+`"}`, x.StaticExtra)
	require.JSONEq(t, `{"factory":"`+factoryB+`"}`, pools[1].StaticExtra)

	// the cursor lists nothing twice
	again, md2, err := u.GetNewPools(context.Background(), md)
	require.NoError(t, err)
	require.Empty(t, again)

	// a token added later lists the pool again, with it
	w.pools[pY].tokens = append(w.pools[pY].tokens, tokenCfg(cbBTC, 8))
	again, _, err = u.GetNewPools(context.Background(), md2)
	require.NoError(t, err)
	require.Len(t, again, 1)
	require.Len(t, again[0].Tokens, 3)
	var cur Metadata
	require.NoError(t, json.Unmarshal(md2, &cur))
	require.Equal(t, map[string]int{poolX: 3, "0x00000000000000000000000000000000000000a2": 2}, cur.Pools)

	// an unreadable cursor lists afresh
	afresh, _, err := u.GetNewPools(context.Background(), []byte("x"))
	require.NoError(t, err)
	require.Len(t, afresh, 2)
}

func TestListerListsTheOtherFactoriesWhenOneDoesNotRead(t *testing.T) {
	w := testWorld()
	n := newNode(t, testBlk, w.respond, w.codeAt)
	cfg := testConfig()
	cfg.Factories = append([]string{"0x00000000000000000000000000000000000000f9"}, cfg.Factories...) // no contract
	u := NewPoolsListUpdater(cfg, n.client())
	pools, md, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.JSONEq(t, `{"pools":{"`+poolX+`":3}}`, string(md))

	n.setDown(true)
	pools, md2, err := u.GetNewPools(context.Background(), md)
	require.NoError(t, err)
	require.Empty(t, pools)
	require.JSONEq(t, string(md), string(md2), "the cursor is kept")
}
