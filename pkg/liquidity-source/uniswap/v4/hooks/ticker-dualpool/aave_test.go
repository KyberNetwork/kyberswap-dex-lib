package tickerdualpool

import (
	"encoding/json"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
)

func TestAaveRoundTripMatchesForkInventory(t *testing.T) {
	raw := []byte(capturedAaveInventory)
	var inv []struct{ Before, After map[string]string }
	require.NoError(t, json.Unmarshal(raw, &inv))
	raw = []byte(capturedMonadQuotes)
	var quotes []map[string]string
	require.NoError(t, json.Unmarshal(raw, &quotes))
	for i, f := range quotes {
		s := fixtureState()
		s.Pool.Balance0 = uint256.MustFromDecimal(f["before0"])
		s.Total0 = s.Pool.Balance0.Clone()
		s.Pool.Balance1 = uint256.MustFromDecimal(f["before1"])
		s.Total1 = s.Pool.Balance1.Clone()
		s.Pool.SqrtPriceX96 = uint256.MustFromDecimal(f["initialPrice"])
		tick, err := strconv.Atoi(f["initialTick"])
		require.NoError(t, err)
		s.Pool.Tick = int32(tick)
		s.Pool.Buckets = []dualpool.Bucket{{TickLower: 312380, TickUpper: 332380, WeightBps: 10000}}
		q, err := QuoteExactInput(s, f["direction"] == "true", uint256.MustFromDecimal(f["input"]))
		require.NoError(t, err)
		b := inv[i].Before
		v := AaveInventory{uint256.MustFromDecimal(b["raw"]), uint256.MustFromDecimal(b["claims"]), uint256.MustFromDecimal(b["poolShares"]), uint256.MustFromDecimal(b["supply"]), uint256.MustFromDecimal(b["idle"]), uint256.MustFromDecimal(b["strategyShares"]), uint256.MustFromDecimal(b["rate"]), uint256.MustFromDecimal(b["pendingYield"])}
		next, err := v.AfterJIT(q.Swap.Mint0, q.Swap.Burn0)
		require.NoError(t, err)
		a := inv[i].After
		require.Equal(t, a["raw"], next.Raw.Dec(), "swap %d raw", i)
		require.Equal(t, a["claims"], next.Claims.Dec(), "swap %d claims", i)
		require.Equal(t, a["poolShares"], next.PoolShares.Dec(), "swap %d pool shares", i)
		require.Equal(t, a["supply"], next.Supply.Dec(), "swap %d supply", i)
		require.Equal(t, a["strategyShares"], next.StrategyShares.Dec(), "swap %d strategy", i)
		require.Equal(t, a["idle"], next.Idle.Dec())
		require.Equal(t, a["rate"], next.Rate.Dec())
		require.Equal(t, a["pendingYield"], next.PendingYield.Dec())
	}
}

func TestAaveSequentialHookUpdatesMatchFork(t *testing.T) {
	raw := []byte(capturedAaveInventory)
	var inventories []struct{ Before, After map[string]string }
	require.NoError(t, json.Unmarshal(raw, &inventories))
	raw = []byte(capturedMonadQuotes)
	var quotes []map[string]string
	require.NoError(t, json.Unmarshal(raw, &quotes))
	f := quotes[0]
	b := inventories[0].Before
	state := fixtureState()
	state.Pool.Balance0 = uint256.MustFromDecimal(f["before0"])
	state.Total0 = state.Pool.Balance0.Clone()
	state.Pool.Balance1 = uint256.MustFromDecimal(f["before1"])
	state.Total1 = state.Pool.Balance1.Clone()
	state.Pool.SqrtPriceX96 = uint256.MustFromDecimal(f["initialPrice"])
	tick, err := strconv.Atoi(f["initialTick"])
	require.NoError(t, err)
	state.Pool.Tick = int32(tick)
	state.Pool.Buckets = []dualpool.Bucket{{TickLower: 312380, TickUpper: 332380, WeightBps: 10000}}
	h := &Hook{Snapshot: Snapshot{State: state}}
	h.Vaults[0] = common.HexToAddress("0x1234")
	h.Aave[0] = &AaveInventory{uint256.MustFromDecimal(b["raw"]), uint256.MustFromDecimal(b["claims"]), uint256.MustFromDecimal(b["poolShares"]), uint256.MustFromDecimal(b["supply"]), uint256.MustFromDecimal(b["idle"]), uint256.MustFromDecimal(b["strategyShares"]), uint256.MustFromDecimal(b["rate"]), uint256.MustFromDecimal(b["pendingYield"])}
	for i, f := range quotes {
		amount, ok := new(big.Int).SetString(f["input"], 10)
		require.True(t, ok)
		result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: f["direction"] == "true", AmountSpecified: amount})
		require.NoError(t, err)
		q := result.SwapInfo.(Quote)
		require.Equal(t, f["net"], q.NetOutput.Dec())
		require.Equal(t, f["price"], q.Swap.SqrtPriceX96.Dec())
		h.UpdateBalance(q)
		require.False(t, h.RefreshRequired)
		require.Equal(t, inventories[i].After["poolShares"], h.Aave[0].PoolShares.Dec())
		require.Equal(t, inventories[i].After["strategyShares"], h.Aave[0].StrategyShares.Dec())
		if i+1 < len(quotes) {
			require.Equal(t, quotes[i+1]["before0"], h.Pool.Balance0.Dec())
			require.Equal(t, quotes[i+1]["before1"], h.Pool.Balance1.Dec())
		}
	}
}

func TestAaveInventoryRejectsMalformedAndOverflowingState(t *testing.T) {
	base := func() AaveInventory {
		return AaveInventory{Raw: uint256.NewInt(0), Claims: uint256.NewInt(0), PoolShares: uint256.NewInt(1000000000), Supply: uint256.NewInt(1000000000), Idle: uint256.NewInt(0), StrategyShares: uint256.NewInt(1000), Rate: ray.Clone(), PendingYield: uint256.NewInt(0)}
	}
	for _, name := range []string{"nil", "zero-rate", "excess-shares", "supply-overflow", "packed-overflow"} {
		t.Run(name, func(t *testing.T) {
			v := base()
			switch name {
			case "nil":
				v.Idle = nil
			case "zero-rate":
				v.Rate.Clear()
			case "excess-shares":
				v.PoolShares.AddUint64(v.Supply, 1)
			case "supply-overflow":
				v.Supply.SetAllOne()
			case "packed-overflow":
				v.Raw.Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 128), uint256.NewInt(1))
				v.Claims.SetUint64(1)
			}
			_, err := v.AfterJIT(uint256.NewInt(100), uint256.NewInt(90))
			require.Error(t, err)
		})
	}
	v := base()
	before, _ := json.Marshal(v)
	_, err := v.AfterJIT(uint256.NewInt(2000), uint256.NewInt(2001))
	require.Error(t, err)
	after, _ := json.Marshal(v)
	require.Equal(t, string(before), string(after), "a failed quote must not mutate the original")
}

func TestAaveZeroShareDepositLeavesCashAndDilutionUnchanged(t *testing.T) {
	v := AaveInventory{Raw: uint256.NewInt(1), Claims: uint256.NewInt(0), PoolShares: uint256.NewInt(1000000000), Supply: uint256.NewInt(1000000000), Idle: uint256.NewInt(0), StrategyShares: uint256.NewInt(1000), Rate: new(uint256.Int).Add(ray, uint256.NewInt(1)), PendingYield: uint256.NewInt(20)}
	next, err := v.AfterJIT(uint256.NewInt(0), uint256.NewInt(0))
	require.NoError(t, err)
	require.Equal(t, "1", next.Raw.Dec())
	require.Equal(t, v.Supply.Dec(), next.Supply.Dec())
	require.Equal(t, v.PendingYield.Dec(), next.PendingYield.Dec(), "failed nested deposit rolls back _accrue")
	require.Equal(t, v.StrategyShares.Dec(), next.StrategyShares.Dec())
}

// Captured contract results; preserved from aave-inventory.json.
const capturedAaveInventory = `[
{"before":{"raw":"0","claims":"0","poolShares":"10000000000000000","supply":"10000000000000000","idle":"0","strategyShares":"9899743234","rate":"1010127208674647155365377435","pendingYield":"0","claimValue":"9999999999"},"after":{"raw":"0","claims":"99999998","poolShares":"9999999999995499","supply":"9999999999995499","idle":"0","strategyShares":"9899743233","rate":"1010127208674647155365377435","pendingYield":"0","claimValue":"9999999998"}},
{"before":{"raw":"0","claims":"99999998","poolShares":"9999999999995499","supply":"9999999999995499","idle":"0","strategyShares":"9899743233","rate":"1010127208674647155365377435","pendingYield":"0","claimValue":"9999999998"},"after":{"raw":"0","claims":"0","poolShares":"10089947394009747","supply":"10089947394009747","idle":"0","strategyShares":"9988788843","rate":"1010127208674647155365377435","pendingYield":"0","claimValue":"10089947392"}}
]`
