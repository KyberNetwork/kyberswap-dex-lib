package spireprop

import (
	"math"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

const baseToken = "0x4200000000000000000000000000000000000006"
const quoteToken = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"

type QuoteCase struct {
	IndexIn    int
	AmountIn   string
	AmountOut  string
	Executable bool
}
type QuoteFixture struct {
	Pool   entity.Pool
	Quotes []QuoteCase
}

func flat(t *testing.T) *PoolSimulator {
	t.Helper()
	extra := Extra{Seq: 1, LastUpdateAt: 1000, BlockTimestamp: 1000, ValidUntil: 1008, TTL: 8,
		Mid: *uint256.NewInt(2_500_000_000), QUnit: *uint256.NewInt(10_000_000_000), CUnit: *uint256.NewInt(1),
		Ask: Side{DepthBps: 10000, Knots: []Knot{{Q: *uint256.NewInt(1_000_000)}}},
		Bid: Side{DepthBps: 10000, Knots: []Knot{{Q: *uint256.NewInt(1_000_000)}}}}
	raw, err := json.Marshal(&extra)
	require.NoError(t, err)
	static, err := json.Marshal(StaticExtra{Entrypoint: "0x98c1d9e102eb2806d902b13186bdc7892ac4ffba", CurveBook: "0x604d9b9eb1e1571c78661a6c1088427ec9c8c6e5", Custodian: "0xaac48feb93c5c97e0fb3c7c57e1633922a4acda3"})
	require.NoError(t, err)
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{Address: "0x0000000000000000000000000000000000000001", Exchange: DexType, Type: DexType, BlockNumber: 100,
		Tokens: []*entity.PoolToken{{Address: baseToken}, {Address: quoteToken}}, Reserves: entity.PoolReserves{"100000000000000000000", "1000000000000000"}, Extra: string(raw), StaticExtra: string(static)}})
	require.NoError(t, err)
	return s
}

func quoteParams(side int, amount string) pool.CalcAmountOutParams {
	value, _ := new(big.Int).SetString(amount, 10)
	tokens := []string{baseToken, quoteToken}
	return pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[side], Amount: value}, TokenOut: tokens[1-side]}
}

func TestFlatPricesAndDepth(t *testing.T) {
	s := flat(t)
	for _, tc := range []struct {
		side    int
		in, out string
	}{{1, "25000000", "10000000000000000"}, {1, "12500000", "5000000000000000"}, {0, "10000000000000000", "25000000"}} {
		result, err := s.CalcAmountOut(quoteParams(tc.side, tc.in))
		require.NoError(t, err)
		require.Equal(t, tc.out, result.TokenAmountOut.Amount.String())
	}
	s.Extra.Ask.DepthBps = 5000
	s.Extra.Bid.DepthBps = 0
	_, err := s.CalcAmountOut(quoteParams(1, "12500001"))
	require.ErrorIs(t, err, ErrDepth)
	_, err = s.CalcAmountOut(quoteParams(0, "1"))
	require.ErrorIs(t, err, ErrDepth)
}

func TestSignedSpreadAndCursorConsumption(t *testing.T) {
	s := flat(t)
	s.Extra.Ask.SpreadBps = 100
	s.Extra.Bid.SpreadBps = -100
	result, err := s.CalcAmountOut(quoteParams(0, "10000000000000000"))
	require.NoError(t, err)
	require.Equal(t, "24750000", result.TokenAmountOut.Amount.String())
	result, err = s.CalcAmountOut(quoteParams(1, "12500000"))
	require.NoError(t, err)
	before, err := json.Marshal(s)
	require.NoError(t, err)
	again, err := s.CalcAmountOut(quoteParams(1, "12500000"))
	require.NoError(t, err)
	require.Equal(t, result, again)
	after, err := json.Marshal(s)
	require.NoError(t, err)
	require.Equal(t, before, after)
	clone := s.CloneState().(*PoolSimulator)
	// Consumption uses SwapInfo even when the caller's other fields are empty.
	clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})
	require.Equal(t, uint64(1), clone.Extra.FillSeq)
	require.Equal(t, uint64(0), s.Extra.FillSeq)
	require.False(t, clone.Extra.Ask.Filled.IsZero())
	require.True(t, s.Extra.Ask.Filled.IsZero())
	clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: result.SwapInfo})
	require.Equal(t, uint64(1), clone.Extra.FillSeq)
	_, err = clone.CalcAmountOut(quoteParams(1, "25000000"))
	require.Error(t, err)
}

func TestDeadlinesAndMalformedState(t *testing.T) {
	s := flat(t)
	require.False(t, s.expired(1008))
	require.True(t, s.expired(1009))
	s.Extra.TTL = 0
	require.True(t, s.expired(1009))
	s.Extra.TTL = 1
	require.True(t, s.expired(1002))
	s.Extra.TTL = math.MaxUint64
	require.False(t, s.expired(1008))
	require.True(t, s.expired(1009))
	s.StaleCheck = true
	_, err := s.CalcAmountOut(quoteParams(1, "1"))
	require.ErrorIs(t, err, ErrExpired)
	s.Extra.ValidUntil = uint64(time.Now().Unix() + 5)
	s.Extra.TTL = 0
	_, err = s.CalcAmountOut(quoteParams(1, "1"))
	require.NoError(t, err)
	s.Extra.Ask.SpreadBps = -10000
	require.ErrorIs(t, s.validate(), ErrInvalidState)
	s = flat(t)
	s.Extra.Ask.Knots = append(s.Extra.Ask.Knots, s.Extra.Ask.Knots[0])
	require.ErrorIs(t, s.validate(), ErrInvalidState)
}

func TestInventoryAndFailurePurity(t *testing.T) {
	s := flat(t)
	s.Info.Reserves[0] = big.NewInt(1)
	before, err := json.Marshal(s)
	require.NoError(t, err)
	_, err = s.CalcAmountOut(quoteParams(1, "25000000"))
	require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
	after, err := json.Marshal(s)
	require.NoError(t, err)
	require.Equal(t, before, after)
	s = flat(t)
	limit := swaplimit.NewInventory(DexType, s.CalculateLimit())
	params := quoteParams(0, "10000000000000000")
	params.Limit = limit
	quote, err := s.CalcAmountOut(params)
	require.NoError(t, err)
	old := new(big.Int).Set(limit.GetLimit(s.limitKey(quoteToken)))
	s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: quote.SwapInfo, SwapLimit: limit})
	require.Equal(t, new(big.Int).Sub(old, big.NewInt(25000000)), limit.GetLimit(s.limitKey(quoteToken)))
	params = quoteParams(1, "25000000")
	params.Limit = swaplimit.NewInventory(DexType, map[string]*big.Int{})
	_, err = s.CalcAmountOut(params)
	require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
}

func TestWidePartialProductAndCheckedOverflow(t *testing.T) {
	s := flat(t)
	s.Extra.QUnit = *uint256.MustFromDecimal("1000000000000000000000000000000000")
	s.Extra.Ask.Knots[0].Q = *uint256.NewInt((1 << 48) - 1)
	s.Info.Reserves[0] = uint256.MustFromDecimal("1000000000000000000000000000000000000000000000000000000000000").ToBig()
	result, err := s.CalcAmountOut(quoteParams(1, "1000000000000000000000000000000"))
	require.NoError(t, err)
	require.Equal(t, "400000000000000000000000000000000000000", result.TokenAmountOut.Amount.String())
	max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).String()
	_, err = s.CalcAmountOut(quoteParams(0, max))
	require.Error(t, err)
	s.Extra.CUnit = *uint256.MustFromDecimal(max)
	s.Extra.Ask.Knots[0].Extra.SetUint64(2)
	require.ErrorIs(t, s.validate(), ErrOverflow)
}

func TestPinnedDeploymentFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/base-snapshot.json")
	require.NoError(t, err)
	var f QuoteFixture
	require.NoError(t, json.Unmarshal(raw, &f))
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: f.Pool})
	require.NoError(t, err)
	for _, tc := range f.Quotes {
		result, err := s.CalcAmountOut(quoteParams(tc.IndexIn, tc.AmountIn))
		if !tc.Executable {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, tc.AmountOut, result.TokenAmountOut.Amount.String())
	}
	meta := s.GetMetaInfo(baseToken, quoteToken).(PoolMeta)
	require.Equal(t, f.Pool.BlockNumber, meta.BlockNumber)
	require.Equal(t, s.StaticExtra.Entrypoint, s.GetApprovalAddress(baseToken, quoteToken))
}

// Integer-sized segments make each rounding decision independently checkable.
func TestMultiKnotRoundingAndConsumedOrigin(t *testing.T) {
	s := flat(t)
	s.Extra.QUnit.SetUint64(1)
	s.Extra.CUnit.SetUint64(1)
	s.Extra.Mid.SetUint64(1_000_000_000_000_000_000)
	knots := []Knot{{Q: *uint256.NewInt(3), Extra: *uint256.NewInt(1)}, {Q: *uint256.NewInt(10), Extra: *uint256.NewInt(5)}}
	s.Extra.Ask.Knots = knots
	s.Extra.Bid.Knots = knots
	for _, tc := range []struct {
		side    int
		in, out string
	}{{1, "4", "3"}, {1, "15", "10"}, {0, "3", "2"}, {0, "4", "2"}, {0, "10", "5"}} {
		result, err := s.CalcAmountOut(quoteParams(tc.side, tc.in))
		require.NoError(t, err)
		require.Equal(t, tc.out, result.TokenAmountOut.Amount.String())
	}
	// One quote unit left after the first knot cannot buy an atomic base unit.
	_, tinyErr := s.CalcAmountOut(quoteParams(1, "5"))
	require.ErrorIs(t, tinyErr, ErrDepth)
	s.Extra.Bid.Filled.SetUint64(3)
	_, err := s.CalcAmountOut(quoteParams(0, "2"))
	require.ErrorIs(t, err, ErrDepth)
	// extra(5)=1+ceil(4*2/7)=3; extra(3)=1, so two units have no net output.
	// Use a three-unit sale: extra(6)=3, mid=3, net=1.
	result, err := s.CalcAmountOut(quoteParams(0, "3"))
	require.NoError(t, err)
	require.Equal(t, "1", result.TokenAmountOut.Amount.String())
	s.Extra.Ask.Filled.SetUint64(3)
	result, err = s.CalcAmountOut(quoteParams(1, "5"))
	require.NoError(t, err)
	require.Equal(t, "3", result.TokenAmountOut.Amount.String())
}

func TestAdditionalBaseScaleAndSharedQuoteInventory(t *testing.T) {
	weth := flat(t)
	other := flat(t)
	other.Info.Tokens[0] = "0x0000000000000000000000000000000000000002"
	// An eight-decimal base at the same 2,500 quote-token price. qUnit and
	// midpoint come from the contract, so no WETH/18-decimal assumption applies.
	other.Extra.QUnit.SetUint64(1)
	other.Extra.Mid = *uint256.MustFromDecimal("25000000000000000000")
	other.Info.Reserves[0] = big.NewInt(100_000_000)
	weth.Info.Reserves[1] = big.NewInt(25_000_000)
	other.Info.Reserves[1] = big.NewInt(25_000_000)
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: quoteToken, Amount: big.NewInt(25_000_000)}, TokenOut: other.Info.Tokens[0]}
	buy, err := other.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, "1000000", buy.TokenAmountOut.Amount.String())
	params = pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: other.Info.Tokens[0], Amount: big.NewInt(1_000_000)}, TokenOut: quoteToken}
	sell, err := other.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, "25000000", sell.TokenAmountOut.Amount.String())

	inventory := weth.CalculateLimit()
	for key, value := range other.CalculateLimit() {
		inventory[key] = value
	}
	limit := swaplimit.NewInventory(DexType, inventory)
	firstParams := quoteParams(0, "10000000000000000")
	firstParams.Limit = limit
	first, err := weth.CalcAmountOut(firstParams)
	require.NoError(t, err)
	weth.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo, SwapLimit: limit})
	require.Zero(t, limit.GetLimit(other.limitKey(quoteToken)).Sign())
	// The second base has its own cursor but must not spend the same USDC twice.
	params.Limit = limit
	_, err = other.CalcAmountOut(params)
	require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
	require.Zero(t, other.Extra.FillSeq)
	require.Equal(t, other.Info.Tokens[0], other.GetMetaInfo("", "").(PoolMeta).Base)
}

func TestSellCursorOverflowMatchesContractDepthFailure(t *testing.T) {
	s := flat(t)
	s.Extra.Bid.Filled.SetUint64(1)
	max := new(uint256.Int).SetAllOne()
	_, arithmeticErr := add(&s.Extra.Bid.Filled, max)
	require.ErrorIs(t, arithmeticErr, ErrOverflow)
	// The contract uses tryAdd and returns Fail.BeyondDepth on this specific
	// overflow, causing quote() to return zero and consume() to reject depth.
	_, err := s.CalcAmountOut(quoteParams(0, max.Dec()))
	require.ErrorIs(t, err, ErrDepth)
	require.Equal(t, uint64(1), s.Extra.Bid.Filled.Uint64())
	require.Zero(t, s.Extra.FillSeq)
}
