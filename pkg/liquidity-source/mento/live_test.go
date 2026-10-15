package mento

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Live tests against Monad mainnet; skipped when CI is set.
const (
	monadRPC        = "https://rpc.monad.xyz"
	monadMulticall3 = "0xcA11bde05977b3631167028862bE2a173976CA11"
	monadFactory    = "0xa849b475FE5a4B5C9C3280152c7a1945b907613b"
	gbpmUsdmPool    = "0xd0e9c1a718d2a693d41eacd4b2696180403ce081"
)

func liveConfig() *Config {
	return &Config{
		DexID:          string(DexType),
		ChainID:        valueobject.ChainIDMonad,
		FactoryAddress: monadFactory,
		NewPoolLimit:   3,
	}
}

func liveClient() *ethrpc.Client {
	return ethrpc.New(monadRPC).SetMulticallContract(common.HexToAddress(monadMulticall3))
}

// listAll drives GetNewPools through its offset cursor until it returns
// nothing, exactly as pool-service does.
func listAll(t *testing.T, u *PoolsListUpdater) []entity.Pool {
	t.Helper()
	var (
		all      []entity.Pool
		metadata []byte
	)
	for i := 0; i < 10; i++ {
		pools, next, err := u.GetNewPools(context.Background(), metadata)
		require.NoError(t, err)
		if len(pools) == 0 {
			break
		}
		all = append(all, pools...)
		metadata = next
	}
	return all
}

func TestPoolsListUpdater_Live(t *testing.T) {
	test.SkipCI(t)

	pools := listAll(t, NewPoolsListUpdater(liveConfig(), liveClient()))
	require.NotEmpty(t, pools)
	t.Logf("%d pools", len(pools))

	byAddr := map[string]entity.Pool{}
	for _, p := range pools {
		assert.Equal(t, DexType, p.Type)
		assert.Len(t, p.Tokens, 2)
		assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		assert.NotEmpty(t, se.Decimals0)
		byAddr[p.Address] = p
	}
	p, ok := byAddr[usdcUsdmPool]
	require.True(t, ok, "USDC/USDm pool discovered")
	assert.Equal(t, usdc, p.Tokens[0].Address)
	assert.Equal(t, usdm, p.Tokens[1].Address)
	assert.Equal(t, `{"dec0":"1000000","dec1":"1000000000000000000"}`, p.StaticExtra)
}

func seedPool(address, token0, token1 string, staticExtra string) entity.Pool {
	return entity.Pool{
		Address:  address,
		Exchange: string(DexType),
		Type:     DexType,
		Reserves: entity.PoolReserves{"0", "0"},
		Tokens: []*entity.PoolToken{
			{Address: token0, Swappable: true},
			{Address: token1, Swappable: true},
		},
		StaticExtra: staticExtra,
	}
}

func TestPoolTracker_Live(t *testing.T) {
	test.SkipCI(t)

	tracker := NewPoolTracker(liveConfig(), liveClient())
	seed := seedPool(usdcUsdmPool, usdc, usdm, `{"dec0":"1000000","dec1":"1000000000000000000"}`)
	p, err := tracker.GetNewPoolState(context.Background(), seed, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	assert.Positive(t, p.BlockNumber)
	assert.NotEqual(t, "0", p.Reserves[0])
	assert.NotEqual(t, "0", p.Reserves[1])

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	t.Logf("extra: %s", p.Extra)
	assert.False(t, extra.Unquoteable)
	assert.Equal(t, uint64(3), extra.LpFee)
	assert.Equal(t, uint64(2), extra.ProtocolFee)
	assert.False(t, extra.RateNumerator.IsZero())
	assert.Equal(t, "1000000000000000000", extra.RateDenominator.Dec())
	assert.Positive(t, extra.RateTimestamp)
	assert.Equal(t, uint64(3720), extra.RateExpiry)
	assert.False(t, extra.EnforceMarketHours, "stablecoin pool breaker is always open")
	assert.Equal(t, uint8(6), extra.Limits[0].Decimals)
	assert.Equal(t, uint8(18), extra.Limits[1].Decimals)
	assert.Equal(t, "2500000000000000000000", extra.Limits[0].Limit0.Dec())
	assert.Positive(t, extra.Limits[0].LastUpdated0)

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	require.NotNil(t, sim)

	// GBPm/USDm sits behind the real FX-hours breaker; take its token
	// addresses from discovery rather than hardcoding them.
	var gbpm entity.Pool
	for _, lp := range listAll(t, NewPoolsListUpdater(liveConfig(), liveClient())) {
		if lp.Address == gbpmUsdmPool {
			gbpm = lp
		}
	}
	require.NotEmpty(t, gbpm.Address, "GBPm/USDm pool discovered")
	p, err = tracker.GetNewPoolState(context.Background(), gbpm, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	t.Logf("gbpm extra: %s", p.Extra)
	assert.True(t, extra.EnforceMarketHours, "FX pool breaker enforces market hours")
	assert.Equal(t, uint64(10), extra.LpFee)
	assert.Equal(t, uint64(360), extra.RateExpiry)
}

// TestQuoteParity_Live runs the production lister + tracker over every pool,
// then compares the simulator against FPMM.getAmountOut() at the tracker's
// block for a range of sizes in both directions. An on-chain revert (stale
// rate, closed market, suspended feed) must map to a simulator error.
func TestQuoteParity_Live(t *testing.T) {
	test.SkipCI(t)

	client := liveClient()
	cfg := liveConfig()
	pools := listAll(t, NewPoolsListUpdater(cfg, client))
	require.NotEmpty(t, pools)
	tracker := NewPoolTracker(cfg, client)

	sizes := []string{"1", "1000000", "1000000000", "1000000000000", "1000000000000000000", "1000000000000000000000",
		"100000000000000000000000"}

	for _, seed := range pools {
		seed := seed
		t.Run(seed.Address, func(t *testing.T) {
			p, err := tracker.GetNewPoolState(context.Background(), seed, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			sim, err := NewPoolSimulator(p)
			require.NoError(t, err)

			blockNumber := new(big.Int).SetUint64(p.BlockNumber)
			for dir := 0; dir < 2; dir++ {
				tokenIn, tokenOut := p.Tokens[dir].Address, p.Tokens[1-dir].Address
				onchain := make([]*big.Int, len(sizes))
				req := client.NewRequest().SetContext(context.Background()).SetBlockNumber(blockNumber)
				for i, size := range sizes {
					req.AddCall(&ethrpc.Call{
						ABI: poolABI, Target: p.Address, Method: methodGetAmountOut,
						Params: []any{bignumber.NewBig10(size), common.HexToAddress(tokenIn)},
					}, []any{&onchain[i]})
				}
				resp, err := req.TryAggregate()
				require.NoError(t, err)

				for i, size := range sizes {
					res, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: bignumber.NewBig10(size)},
						TokenOut:      tokenOut,
					})
					switch {
					case !resp.Result[i]:
						// getAmountOut reverted: the oracle is unusable right now.
						assert.Errorf(t, simErr, "%s %s->%s: on-chain reverted but simulator quoted", size, tokenIn, tokenOut)
					case onchain[i].Sign() == 0:
						assert.ErrorIs(t, simErr, ErrZeroAmountOut)
					case onchain[i].Cmp(bignumber.NewBig10(p.Reserves[1-dir])) >= 0:
						// getAmountOut is a pure quote; swap() would revert on reserves.
						assert.ErrorIs(t, simErr, ErrInsufficientLiquidity)
					case simErr != nil:
						// remaining simulator-only rejections are the trading limits
						assert.Contains(t, []error{ErrL0LimitExceeded, ErrL1LimitExceeded}, simErr,
							"%s %s->%s: unexpected simulator error", size, tokenIn, tokenOut)
					default:
						assert.Equalf(t, onchain[i].String(), res.TokenAmountOut.Amount.String(),
							"%s %s->%s", size, tokenIn, tokenOut)
					}
				}
			}
		})
	}
}
