package nadswap

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// commonHex parses a hex string into a common.Address.
// Note: Test fixtures use real-looking placeholder addresses (0x0a / 0x0b) so that
// they survive `common.HexToAddress` normalization — the original plan used `0xT0`
// / `0xT1` which both collapse to the zero address, making buy/sell dispatch
// indistinguishable.
func commonHex(s string) common.Address {
	if s == "" {
		return common.Address{}
	}
	return common.HexToAddress(s)
}

// tokenAddr returns the canonical hex form for a placeholder token id so that the
// entity.Pool tokens and the StaticExtra quoteToken normalize to the same value.
func tokenAddr(id string) string {
	return common.HexToAddress(id).Hex()
}

func buildPool(t *testing.T, isMeme bool, quote, token0, token1 string, r0, r1 string, feeRate uint16) *PoolSimulator {
	t.Helper()
	extra := Extra{Reserve0: u(r0), Reserve1: u(r1)}
	se := StaticExtra{
		IsMemePair:         isMeme,
		QuoteToken:         commonHex(quote),
		CreatorFeeRate:     0,
		DexProtocolFeeRate: feeRate, // put whole rate on dexProtocolFeeRate for simplicity
	}
	extraBytes, _ := json.Marshal(extra)
	seBytes, _ := json.Marshal(se)
	p := entity.Pool{
		Address: "0xpair",
		Type:    DexType,
		Tokens: []*entity.PoolToken{
			{Address: tokenAddr(token0), Swappable: true},
			{Address: tokenAddr(token1), Swappable: true},
		},
		Reserves:    []string{r0, r1},
		Extra:       string(extraBytes),
		StaticExtra: string(seBytes),
	}
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	return sim
}

func TestPoolSimulator_CalcAmountOut_GeneralPair(t *testing.T) {
	t.Parallel()
	sim := buildPool(t, false, "", "0x0a", "0x0b", "10000", "10000", 0)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenAddr("0x0a"), Amount: big.NewInt(1000)},
		TokenOut:      tokenAddr("0x0b"),
	})
	require.NoError(t, err)
	assert.Equal(t, "907", res.TokenAmountOut.Amount.String())
}

func TestPoolSimulator_CalcAmountOut_MemeBuy(t *testing.T) {
	t.Parallel()
	sim := buildPool(t, true, "0x0a", "0x0a", "0x0b", "10000", "10000", 100)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenAddr("0x0a"), Amount: big.NewInt(1000)},
		TokenOut:      tokenAddr("0x0b"),
	})
	require.NoError(t, err)
	assert.Equal(t, "898", res.TokenAmountOut.Amount.String())
}

func TestPoolSimulator_CalcAmountOut_MemeSell(t *testing.T) {
	t.Parallel()
	// quote is T1; selling T0 -> T1 means sell direction
	sim := buildPool(t, true, "0x0b", "0x0a", "0x0b", "10000", "10000", 100)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenAddr("0x0a"), Amount: big.NewInt(1000)},
		TokenOut:      tokenAddr("0x0b"),
	})
	require.NoError(t, err)
	assert.Equal(t, "897", res.TokenAmountOut.Amount.String())
}

func TestPoolSimulator_CalcAmountIn_MemeBuy(t *testing.T) {
	t.Parallel()
	sim := buildPool(t, true, "0x0a", "0x0a", "0x0b", "10000", "10000", 100)
	res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: tokenAddr("0x0b"), Amount: big.NewInt(898)},
		TokenIn:        tokenAddr("0x0a"),
	})
	require.NoError(t, err)
	// Round-trip target was 1000, allow 1 unit ceiling slack.
	got := res.TokenAmountIn.Amount.Int64()
	assert.GreaterOrEqual(t, got, int64(1000))
	assert.LessOrEqual(t, got, int64(1001))
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()
	sim := buildPool(t, false, "", "0x0a", "0x0b", "10000", "10000", 0)
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenAddr("0x0a"), Amount: big.NewInt(1000)},
		TokenOut:      tokenAddr("0x0b"),
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokenAddr("0x0a"), Amount: big.NewInt(1000)},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})
	// After update, reserves should reflect: r0 = 10000 + 1000 = 11000; r1 = 10000 - 907 = 9093
	assert.Equal(t, "11000", sim.reserve0.Dec())
	assert.Equal(t, "9093", sim.reserve1.Dec())
}

func TestPoolSimulator_CloneState(t *testing.T) {
	t.Parallel()
	sim := buildPool(t, false, "", "0x0a", "0x0b", "10000", "10000", 0)
	clone := sim.CloneState().(*PoolSimulator)
	// Mutate clone, original must be untouched.
	clone.reserve0.SetUint64(1)
	assert.Equal(t, "10000", sim.reserve0.Dec())
}
