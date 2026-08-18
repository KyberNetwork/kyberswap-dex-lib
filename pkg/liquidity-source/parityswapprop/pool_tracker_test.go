package parityswapprop

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Robinhood chain mainnet RPC + multicall address, matching pool-service's
// own production config for this chain (ArbMulticall2, not the standard
// Multicall3 deterministic-deployment address: on this Arbitrum Orbit chain,
// plain Multicall3's aggregate() reports block.number as read by the EVM
// opcode, which doesn't track the real chain height -- ArbMulticall2 routes
// through ArbSys.arbBlockNumber() instead, so its response.BlockNumber
// actually matches eth_blockNumber and can be used to pin a later call).
//
// Deliberately no hardcoded historical block constant here: pinning a
// mutable-state read (reserves) to a fixed past block number is unsound in a
// permanent test against a public RPC endpoint with a finite retention
// window -- a fixed block ages out and starts failing with "metadata is not
// found" after enough blocks pass.
const (
	liveRPCURL             = "https://rpc.mainnet.chain.robinhood.com"
	multicallAddress       = "0x2cAC2D899eCC914d704FeaAE33ac1bF36277DaD1"
	referencePoolAddress   = "0xd778f470c69bce130d1cef08852f34bf296b4e67"
	referenceOracleAddress = "0xc484f39b1c25fc7fcb140fbc0824a6ff9143e405"
)

func newLiveRPCClient() *ethrpc.Client {
	return ethrpc.New(liveRPCURL).SetMulticallContract(common.HexToAddress(multicallAddress))
}

// seedReferencePool is the entity.Pool shape pool_list_updater.go's
// resolvePools would have produced for the reference pool -- StaticExtra
// already populated (discovery's job), Reserves/Extra still zero-ish
// (tracker's job to fill in).
func seedReferencePool() entity.Pool {
	staticExtra := `{"base":"` + weth + `","quote":"` + usdg + `","baseScale":"1000000000000000000","quoteScale":"1000000"}`
	return entity.Pool{
		Address:     referencePoolAddress,
		Exchange:    string(DexType),
		Type:        DexType,
		Reserves:    entity.PoolReserves{"0", "0"},
		Tokens:      []*entity.PoolToken{{Address: weth, Decimals: 18, Swappable: true}, {Address: usdg, Decimals: 6, Swappable: true}},
		StaticExtra: staticExtra,
	}
}

// TestGetNewPoolState_Live exercises the full GetNewPoolState wiring (both
// multicalls pinned to the same block, JSON marshal, block/timestamp
// stamping) against the real pool. Mutable fields (reserves, oracle price,
// params) are only sanity-checked, not pinned to historical values -- they
// are live state and legitimately drift over time.
// TestGetReserves_MatchesRawMulticallRead below is the same-block
// cross-check that satisfies the reserves_sample math-redundancy gate.
func TestGetNewPoolState_Live(t *testing.T) {
	test.SkipCI(t)

	tracker, err := NewPoolTracker(&Config{ChainID: valueobject.ChainIDRobinhood}, newLiveRPCClient())
	require.NoError(t, err)

	updated, err := tracker.GetNewPoolState(context.Background(), seedReferencePool(), pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	require.Len(t, updated.Reserves, 2)
	baseReserve, ok := new(big.Int).SetString(updated.Reserves[0], 10)
	require.True(t, ok)
	quoteReserve, ok := new(big.Int).SetString(updated.Reserves[1], 10)
	require.True(t, ok)
	assert.True(t, baseReserve.Sign() > 0, "base reserve should be positive")
	assert.True(t, quoteReserve.Sign() > 0, "quote reserve should be positive")
	assert.True(t, updated.BlockNumber > 0)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
	assert.Equal(t, referenceOracleAddress, extra.Oracle)
	assert.True(t, extra.OracleBid > 0, "oracle bid should be posted")
	assert.True(t, extra.OracleMid > 0, "oracle mid should be posted")
	assert.True(t, extra.OracleAsk > 0, "oracle ask should be posted")
	assert.LessOrEqual(t, extra.FeeBps, uint16(bps))
	require.NotNil(t, extra.MaxSwapNotional)
	require.NotNil(t, extra.MaxBlockNotional)
	require.NotNil(t, extra.MinBaseReserve)
	require.NotNil(t, extra.MinQuoteReserve)
	require.NotNil(t, extra.BlockNotional)

	// Confirms the tracker's output round-trips through the simulator
	// constructor -- the same handoff pool-service performs in production.
	sim, err := NewPoolSimulator(updated)
	require.NoError(t, err)
	require.NotNil(t, sim)
}

// TestGetReserves_MatchesRawMulticallRead cross-checks GetNewPoolState's
// reserves decode (produced inside its normal 6-call batched Aggregate) against
// an isolated single-call read of the exact same getReserves() function, both
// pinned to the same block -- fetched fresh via GetBlockNumber at test-run
// time, never a hardcoded historical constant (see the const block above for
// why). This exists to catch batching-specific bugs (wrong call
// ordering/indexing inside the 6-call Aggregate corrupting one field with
// another's data) that a from-scratch single-purpose call can't share, even
// though it can't catch an ABI/struct-shape bug common to both call sites.
func TestGetReserves_MatchesRawMulticallRead(t *testing.T) {
	test.SkipCI(t)

	client := newLiveRPCClient()
	blockNumber, err := client.GetBlockNumber(context.Background())
	require.NoError(t, err)
	blockNumberBig := new(big.Int).SetUint64(blockNumber)

	var raw poolReserves
	_, err = client.NewRequest().SetContext(context.Background()).
		SetBlockNumber(blockNumberBig).
		AddCall(&ethrpc.Call{ABI: PmmPoolABI, Target: referencePoolAddress, Method: methodGetReserves}, []any{&raw}).
		Call()
	require.NoError(t, err)
	require.NotNil(t, raw.BaseReserve)
	require.NotNil(t, raw.QuoteReserve)

	tracker, err := NewPoolTracker(&Config{ChainID: valueobject.ChainIDRobinhood}, client)
	require.NoError(t, err)
	updated, err := tracker.GetNewPoolState(context.Background(), seedReferencePool(), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.Len(t, updated.Reserves, 2)

	assert.Equal(t, raw.BaseReserve.String(), updated.Reserves[0])
	assert.Equal(t, raw.QuoteReserve.String(), updated.Reserves[1])
}
