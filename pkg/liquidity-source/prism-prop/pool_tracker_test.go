package prismprop

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	weth = "0x4200000000000000000000000000000000000006"
	usdc = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
)

// realOrderBookHex is the raw return of Base's getOrderBookWithParams(WETH,
// USDC, {msgSender: executor}) at block 52369426. The expected outputs below
// are getAmountOutWithParams(executor) at the same block.
const realOrderBookHex = "00000000000000000000000000000000000000000000000000000000000000200000000000000000000000004200000000000000000000000000000000000006000000000000000000000000833589fcd6edb6e08f4c7c32d4f71b54bda0291300000000000000000000000000000000000000000000000000000000031f181200000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000026000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000077ded76ac0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000119799812dea11197f100000000000000000000000000000000000000000000000000000000031f1814000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000002c6349c36976600000000000000000000000000000000000000000000000000000000001dc4653c0000000000000000000000000000000000000000000000000ddf070be6ef0c000000000000000000000000000000000000000000000000000000000094d5c9580000000000000000000000000000000000000000000000000b1c0de0292cbd7e000000000000000000000000000000000000000000000000000000007732201c0000000000000000000000000000000000000000000000001bbad2a96d7a1a82000000000000000000000000000000000000000000000000000000012983defa00000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000048ab4843ef183a6f0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000fffffffffffffffffffffffffffe00000000000000000000000000000000000000000000000000000000031f1814000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000002c620bd65137df6000000000000000000000000000000000000000000000000000000001dc54d420000000000000000000000000000000000000000000000000ddea3b1aff401de0000000000000000000000000000000000000000000000000000000094dab3080000000000000000000000000000000000000000000000000b1b2d628f7b5717000000000000000000000000000000000000000000000000000000007733f40e0000000000000000000000000000000000000000000000001bba9cfa011e93d100000000000000000000000000000000000000000000000000000001299dcff6"

func newSim(t testing.TB) *PoolSimulator {
	t.Helper()
	data, err := hex.DecodeString(realOrderBookHex)
	require.NoError(t, err)
	var res getOrderBookResult
	require.NoError(t, routerABI.UnpackIntoInterface(&res, methodGetOrderBookWithParams, data))

	extra, err := json.Marshal(orderbook.Extra{LevelsFrom: [2][]orderbook.Level{
		toLevels(res.Book.Bids, true, 18, 6),
		toLevels(res.Book.Asks, false, 6, 18),
	}})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(poolpkg.FactoryParams{EntityPool: entity.Pool{
		Address:  "prismprop_" + weth + "_" + usdc,
		Exchange: string(DexType),
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: weth, Decimals: 18, Swappable: true},
			{Address: usdc, Decimals: 6, Swappable: true},
		},
		Reserves:    entity.PoolReserves{"0", "0"},
		Extra:       string(extra),
		StaticExtra: `{"router":"0x290e41f7b52dbd2dc1e3009fe19e6a824f296a42"}`,
		Timestamp:   time.Now().Unix(),
	}})
	require.NoError(t, err)
	return sim
}

func calc(sim *PoolSimulator, tokenIn, tokenOut, amountIn string) (*big.Int, error) {
	in, _ := new(big.Int).SetString(amountIn, 10)
	res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: tokenIn, Amount: in},
		TokenOut:      tokenOut,
	})
	if err != nil {
		return nil, err
	}
	return res.TokenAmountOut.Amount, nil
}

// TestSimulator_MatchesGetAmountOutWithParams guards the bids/asks mapping: bids
// are the base->quote ladder and asks the quote->base ladder. Merging both
// sides into one direction, as the old tracker did, quotes the wrong prices
// and doubles the depth.
func TestSimulator_MatchesGetAmountOutWithParams(t *testing.T) {
	sim := newSim(t)

	for _, tc := range []struct{ in, out, amountIn, want string }{
		{weth, usdc, "100000000000000000", "249823685"},
		{weth, usdc, "1200000000000000000", "2997871639"}, // spans 3 levels
		{weth, usdc, "3900000000000000000", "9742634518"},
		{usdc, weth, "100000000", "40019095791767453"},
		{usdc, weth, "3000000000", "1200567785819060204"},
		{usdc, weth, "9000000000", "3601543954878352599"},
	} {
		got, err := calc(sim, tc.in, tc.out, tc.amountIn)
		require.NoError(t, err)
		want, _ := new(big.Int).SetString(tc.want, 10)
		// the order-book simulator walks the ladder in float64
		assert.InEpsilonf(t, want.Uint64(), got.Uint64(), 1e-9, "%s -> %s %s", tc.in, tc.out, tc.amountIn)
	}
}

// TestSimulator_RejectsBeyondDepth: the venue has no partial fills, one unit
// past the deepest level quotes 0 (here 4.1 WETH and 10500 USDC).
func TestSimulator_RejectsBeyondDepth(t *testing.T) {
	sim := newSim(t)

	_, err := calc(sim, weth, usdc, "4100000000000000000")
	assert.ErrorIs(t, err, orderbook.ErrInsufficientLiquidity)
	_, err = calc(sim, usdc, weth, "10500000000")
	assert.ErrorIs(t, err, orderbook.ErrInsufficientLiquidity)
}

// TestSimulator_SmallTradeNotRejected pins the zero-size sentinel: without it
// order-book reads minTrade from the best level's own size and rejects
// trades smaller than that level.
func TestSimulator_SmallTradeNotRejected(t *testing.T) {
	_, err := calc(newSim(t), weth, usdc, "1000000000000")
	assert.NoError(t, err)
}

func TestToLevels_EmptySide(t *testing.T) {
	assert.Nil(t, toLevels(BookSide{}, true, 18, 6))
}

func BenchmarkCalcAmountOut(b *testing.B) {
	sim := newSim(b)
	in := big.NewInt(1_200_000_000_000_000_000)
	params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: weth, Amount: in},
		TokenOut:      usdc,
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := sim.CalcAmountOut(params); err != nil {
			b.Fatal(err)
		}
	}
}
