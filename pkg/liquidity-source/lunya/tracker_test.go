package lunya

import (
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// The Lunya DEX on Arc testnet, and the three pools its factory has listed: a CP pool whose fee is
// charged in token1 (USDC), a CL pool on the default plugin's dynamic fee, and a STABLE pool of mUSDB
// (18 decimals) against mUSDA (6).
const (
	arcTestnetRPC        = "https://rpc.testnet.arc.network"
	arcTestnetMulticall3 = "0xcA11bde05977b3631167028862be2a173976CA11"
	arcTestnetFactory    = "0xf96727d1c01a724e3bab7b00c024786373f0907b"
	arcTestnetQuoter     = "0xc602bc22e2c6bc08ea43b57f45842ff158ddd384"
	arcTestnetUSDC       = "0x3600000000000000000000000000000000000000"

	testnetCPPool      = "0xae4718880f1fec8617de099dfc579fc7b5945b6e"
	testnetCPPoolBlock = 61812247
	testnetCLPool      = "0x0b74ff3c703804a69a9727041ff54012eb649725"
	testnetCLPoolBlock = 61812277
	testnetCLToken1    = "0x89b50855aa3be2f677cd6303cec089b5f319d72a"

	testnetStablePool      = "0xc9ffe87a8f59d4ac9294b4b28df7779637d00bd2"
	testnetStablePoolBlock = 62225954
)

func testnetConfig() *Config {
	return &Config{DexID: DexType, Factory: arcTestnetFactory}
}

func newTestnetClient() *ethrpc.Client {
	return ethrpc.New(arcTestnetRPC).SetMulticallContract(common.HexToAddress(arcTestnetMulticall3))
}

// discoverTestnetPool runs the backfill pool-service drives over the block a pool was created in.
func discoverTestnetPool(t *testing.T, startBlock uint64, address string) entity.Pool {
	t.Helper()

	backfiller, err := poolfactory.NewFilterLogsBackfiller(NewPoolFactoryDecoder(testnetConfig()),
		newTestnetClient(), poolfactory.BackfillConfig{
			FactoryAddresses:     []common.Address{common.HexToAddress(arcTestnetFactory)},
			StartBlock:           startBlock,
			MaxBlockRangePerScan: 1,
			Exchange:             DexType,
		})
	require.NoError(t, err)

	pools, _, _, err := backfiller.Backfill(t.Context(), nil)
	require.NoError(t, err)

	p, found := lo.Find(pools, func(p entity.Pool) bool { return p.Address == address })
	require.True(t, found, "pool %s not discovered", address)
	return p
}

func trackTestnetPool(t *testing.T, p entity.Pool, logs ...ethtypes.Log) (entity.Pool, Extra) {
	t.Helper()

	tracker, err := NewPoolTracker(testnetConfig(), newTestnetClient())
	require.NoError(t, err)

	p, err = tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{Logs: logs})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))

	return p, extra
}

func assertTicksConsistent(t *testing.T, extra Extra) {
	t.Helper()

	var netSum int256.Int
	for i, tick := range extra.Ticks {
		if i > 0 {
			assert.Less(t, extra.Ticks[i-1].Index, tick.Index)
		}
		assert.False(t, tick.LiquidityGross.IsZero())
		netSum.Add(&netSum, tick.LiquidityNet)
	}
	assert.True(t, netSum.IsZero(), "liquidityNet sums to %s", netSum.Dec())
}

func TestPoolTracker_ArcTestnet(t *testing.T) {
	test.SkipCI(t)

	t.Run("CP pool", func(t *testing.T) {
		p, extra := trackTestnetPool(t, discoverTestnetPool(t, testnetCPPoolBlock, testnetCPPool))

		assert.EqualValues(t, 10_000, extra.Fee, "DYNAMIC_FEE is off: slot0.fee")
		assert.EqualValues(t, feeTokenToken1, extra.FeeToken)
		assert.False(t, extra.Halted)
		assert.Positive(t, p.BlockNumber)

		// A CP pool holds every position at [MIN_TICK, MAX_TICK], whatever its tick spacing, so there
		// are two initialized ticks - the tree's first and last leaf words - and all liquidity is active.
		require.Len(t, extra.Ticks, 2)
		assert.Equal(t, -887272, extra.Ticks[0].Index)
		assert.Equal(t, 887272, extra.Ticks[1].Index)
		assert.Equal(t, extra.Liquidity.Dec(), extra.Ticks[0].LiquidityNet.Dec())
		assertTicksConsistent(t, extra)
	})

	t.Run("CL pool", func(t *testing.T) {
		p, extra := trackTestnetPool(t, discoverTestnetPool(t, testnetCLPoolBlock, testnetCLPool))

		assert.Positive(t, extra.Fee)
		assert.EqualValues(t, feeTokenPaid, extra.FeeToken)
		assert.False(t, extra.Halted)
		assert.NotEmpty(t, extra.Ticks)
		assertTicksConsistent(t, extra)

		// A later pass re-reads only the ticks the logs name - including one that is not initialized -
		// and must land on the same list as the full scan.
		logs := lo.Map(extra.Ticks, func(tick Tick, _ int) ethtypes.Log {
			return mintLog(p.Address, tick.Index, tick.Index)
		})
		logs = append(logs, burnLog(p.Address, extra.Ticks[0].Index+1, extra.Ticks[0].Index+2))

		_, again := trackTestnetPool(t, p, logs...)
		assert.Equal(t, extra.Ticks, again.Ticks)
	})

	t.Run("STABLE pool", func(t *testing.T) {
		p, extra := trackTestnetPool(t, discoverTestnetPool(t, testnetStablePoolBlock, testnetStablePool))

		assert.Positive(t, extra.Fee)
		assert.False(t, extra.Halted)
		assert.False(t, extra.Liquidity.IsZero())
		assert.Empty(t, extra.Ticks, "a STABLE pool has no ticks")

		// the rates are 10**(18 - decimals), and the price scale their square-root ratio in Q64.96
		var priceScale uint256.Int
		require.NoError(t, mulDiv(&priceScale, extra.Rate1, q192, extra.Rate0))
		assert.Equal(t, priceScale.Sqrt(&priceScale).Dec(), extra.PriceScaleSqrtQ96.Dec())

		// the curve holds no more than the balances, which also carry the fees
		for i, curve := range []*uint256.Int{extra.CurveReserve0, extra.CurveReserve1} {
			assert.LessOrEqual(t, curve.ToBig().Cmp(bignumber.NewBig10(p.Reserves[i])), 0)
		}
	})
}

func tickTopic(tick int) common.Hash {
	v := uint256.NewInt(uint64(max(tick, -tick)))
	if tick < 0 {
		v.Neg(v)
	}
	return v.Bytes32()
}

func mintLog(poolAddress string, tickLower, tickUpper int) ethtypes.Log {
	return ethtypes.Log{
		Address: common.HexToAddress(poolAddress),
		Topics:  []common.Hash{mintEvent.ID, {}, tickTopic(tickLower), tickTopic(tickUpper)},
	}
}

func burnLog(poolAddress string, tickLower, tickUpper int) ethtypes.Log {
	return ethtypes.Log{
		Address: common.HexToAddress(poolAddress),
		Topics:  []common.Hash{burnEvent.ID, {}, tickTopic(tickLower), tickTopic(tickUpper)},
	}
}

func TestInt24FromTopic(t *testing.T) {
	t.Parallel()

	for _, tick := range []int{-887272, -65536, -1, 0, 1, 60, 887272} {
		assert.Equal(t, tick, int24FromTopic(tickTopic(tick)))
	}
}

func TestTouchedTicks(t *testing.T) {
	t.Parallel()

	swap := mintLog(testnetCLPool, 1, 2)
	swap.Topics[0] = common.HexToHash("0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67")

	logs := []ethtypes.Log{
		mintLog(testnetCLPool, -120, 60),
		burnLog(testnetCLPool, -120, 180),
		mintLog(testnetCPPool, -300, 300),
		swap,
	}

	assert.ElementsMatch(t, []int{-120, 60, 180}, touchedTicks(testnetCLPool, logs))
}

func TestFetchAllTicksWordRange(t *testing.T) {
	t.Parallel()

	// Root bit 0 covers leaf words [-3466, -3211]; bit 27 starts at 3446 and is cut at MAX_TICK's word.
	assert.Equal(t, minLeafWord, 0<<8-leafOffset)
	assert.Equal(t, 3446, 27<<8-leafOffset)
	assert.Equal(t, maxLeafWord, 887272>>8)
	assert.Equal(t, minLeafWord, -887272>>8)
}
