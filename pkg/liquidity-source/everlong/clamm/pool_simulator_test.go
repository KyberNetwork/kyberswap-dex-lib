package everlongclamm

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testKAI  = "0x0f26bbb8962d73bc891327f14db5162d5279899f" // currency0, 18 dec
	testWBTC = "0x0913da6da4b42f538b445599b46bb4622342cf52" // currency1, 8 dec
)

// fixture is a pinned-block capture from live Katana (block 39284109): pool state read
// via CLPoolManager.getSlot0/getLiquidity + ClammALM.getRungs/poolFee, and expected
// outputs from the wei-exact reference port that was parity-validated on-chain.
type fixture struct {
	Block           int64    `json:"block"`
	SqrtPriceX96    string   `json:"sqrtPriceX96"`
	Tick            int      `json:"tick"`
	Liquidity       string   `json:"liquidity"`
	TickSpacing     int      `json:"tickSpacing"`
	PoolFee0For1Wad string   `json:"poolFee0For1Wad"`
	PoolFee1For0Wad string   `json:"poolFee1For0Wad"`
	RungLowers      []int    `json:"rungLowers"`
	RungUppers      []int    `json:"rungUppers"`
	RungLiquidities []string `json:"rungLiquidities"`
	Cases           []struct {
		ZeroForOne bool   `json:"zeroForOne"`
		AmountIn   string `json:"amountIn"`
		Gross      string `json:"gross"`
		Net        string `json:"net"`
	} `json:"cases"`
}

func loadFixture(t *testing.T) fixture {
	raw, err := os.ReadFile("testdata/katana_block_39284109.json")
	require.NoError(t, err)
	var f fixture
	require.NoError(t, json.Unmarshal(raw, &f))
	return f
}

func newTestPoolSimulator(t *testing.T) *PoolSimulator {
	f := loadFixture(t)

	lowers := make([]*big.Int, len(f.RungLowers))
	uppers := make([]*big.Int, len(f.RungUppers))
	liqs := make([]*big.Int, len(f.RungLiquidities))
	for i := range f.RungLowers {
		lowers[i] = big.NewInt(int64(f.RungLowers[i]))
		uppers[i] = big.NewInt(int64(f.RungUppers[i]))
		var ok bool
		liqs[i], ok = new(big.Int).SetString(f.RungLiquidities[i], 10)
		require.True(t, ok)
	}
	ticks, err := rungsToTicks(lowers, uppers, liqs)
	require.NoError(t, err)

	tick := f.Tick
	extraBytes, err := json.Marshal(Extra{
		ExtraTickU256: uniswapv3.ExtraTickU256{
			Liquidity:    uint256.MustFromDecimal(f.Liquidity),
			SqrtPriceX96: uint256.MustFromDecimal(f.SqrtPriceX96),
			TickSpacing:  uint64(f.TickSpacing),
			Tick:         &tick,
			Ticks:        ticks,
		},
		PoolFee0For1Wad: uint256.MustFromDecimal(f.PoolFee0For1Wad),
		PoolFee1For0Wad: uint256.MustFromDecimal(f.PoolFee1For0Wad),
	})
	require.NoError(t, err)

	sqrtPrice := uint256.MustFromDecimal(f.SqrtPriceX96)
	reserve0, reserve1, err := ladderReserves(sqrtPrice, f.Tick, ticks)
	require.NoError(t, err)

	staticExtraBytes, err := json.Marshal(StaticExtra{
		PoolManager: "0xf8058204eddfb48f5fbc7490f4f2871815c1adb4",
		ALM:         "0xa601e3669d76cd7838c04b90dbcedace66454fa9",
		Router:      "0x3067a938ac493cb15ef239e87dcc1888a596f657",
		Hook:        "0x4391b5bda5f882e38a5dd630d5054744aa94991b",
		Fee:         0,
		Parameters:  "0x00000000000000000000000000000000000000000000000000000000000a09d4",
		TickSpacing: f.TickSpacing,
	})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0x7d33a563f58217c160759dbb0809e73933c3189c4f1790b29f3ebc1e69555814",
		Exchange: "everlong-clamm",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: testKAI, Decimals: 18, Swappable: true},
			{Address: testWBTC, Decimals: 8, Swappable: true},
		},
		Reserves:    entity.PoolReserves{reserve0.String(), reserve1.String()},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}, 747474)
	require.NoError(t, err)
	return sim
}

// TestCalcAmountOutMatchesReference checks the simulator wei-exact against pinned-block
// reference outputs (live Katana, block 39284109), produced by the reference port that
// was parity-validated against the on-chain CLPoolManager.
func TestCalcAmountOutMatchesReference(t *testing.T) {
	f := loadFixture(t)

	for _, tc := range f.Cases {
		tokenIn, tokenOut := testKAI, testWBTC
		if !tc.ZeroForOne {
			tokenIn, tokenOut = testWBTC, testKAI
		}
		sim := newTestPoolSimulator(t)
		amountIn, _ := new(big.Int).SetString(tc.AmountIn, 10)
		wantNet, _ := new(big.Int).SetString(tc.Net, 10)
		wantGross, _ := new(big.Int).SetString(tc.Gross, 10)

		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
			TokenOut:      tokenOut,
		})
		require.NoError(t, err, "amountIn=%s zeroForOne=%v", tc.AmountIn, tc.ZeroForOne)

		got := result.TokenAmountOut.Amount
		gotGross := new(big.Int).Add(got, result.Fee.Amount)

		require.Zero(t, wantNet.Cmp(got),
			"net out must be wei-exact vs the chain-validated reference: amountIn=%s zeroForOne=%v got=%s want=%s",
			tc.AmountIn, tc.ZeroForOne, got, wantNet)
		require.Zero(t, wantGross.Cmp(gotGross),
			"gross out must be wei-exact vs the chain-validated reference: amountIn=%s zeroForOne=%v got=%s want=%s",
			tc.AmountIn, tc.ZeroForOne, gotGross, wantGross)
	}
}

// TestCalcAmountOutIsPureAndDeterministic: repeated quoting must not drift.
func TestCalcAmountOutIsPureAndDeterministic(t *testing.T) {
	sim := newTestPoolSimulator(t)
	amountIn, _ := new(big.Int).SetString("5000000000000000000000", 10)
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testKAI, Amount: amountIn},
		TokenOut:      testWBTC,
	}
	first, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		again, err := sim.CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, first.TokenAmountOut.Amount, again.TokenAmountOut.Amount)
	}
}

// TestUpdateBalanceMovesPrice: after consuming input the next quote must be worse, and a
// clone taken before the update must keep quoting the original amount.
func TestUpdateBalanceMovesPrice(t *testing.T) {
	sim := newTestPoolSimulator(t)
	amountIn, _ := new(big.Int).SetString("5000000000000000000000", 10)
	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testKAI, Amount: amountIn},
		TokenOut:      testWBTC,
	}
	first, err := sim.CalcAmountOut(params)
	require.NoError(t, err)

	backup := sim.CloneState()

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testKAI, Amount: amountIn},
		TokenAmountOut: *first.TokenAmountOut,
		Fee:            *first.Fee,
		SwapInfo:       first.SwapInfo,
	})

	second, err := sim.CalcAmountOut(params)
	require.NoError(t, err)
	require.True(t, second.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount) < 0,
		"price must move against the taker after UpdateBalance")

	fromBackup, err := backup.CalcAmountOut(params)
	require.NoError(t, err)
	require.Equal(t, first.TokenAmountOut.Amount, fromBackup.TokenAmountOut.Amount,
		"CloneState must fully insulate the backup from UpdateBalance")
}

// TestSaturatedInputReturnsRemaining: beyond ladder exhaustion, the unconsumed input must
// be surfaced in RemainingTokenAmountIn.
func TestSaturatedInputReturnsRemaining(t *testing.T) {
	sim := newTestPoolSimulator(t)
	amountIn, _ := new(big.Int).SetString("1000000000", 10) // 10 WBTC >> ladder KAI depth
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testWBTC, Amount: amountIn},
		TokenOut:      testKAI,
	})
	require.NoError(t, err)
	require.NotNil(t, result.RemainingTokenAmountIn)
	require.True(t, result.RemainingTokenAmountIn.Amount.Sign() > 0,
		"a swap exhausting the ladder must return the unconsumed input")
}

// TestExactOutRejected: the ClammHook only allows exact-input swaps.
func TestExactOutRejected(t *testing.T) {
	sim := newTestPoolSimulator(t)
	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testWBTC, Amount: big.NewInt(1000000)},
		TokenIn:        testKAI,
	})
	require.ErrorIs(t, err, ErrExactOutNotSupported)
}
