package limitorder

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// A maker's balance in the limit caps what that maker fills. Native CalcAmtOut must match the legacy
// CalcAmountOut under the cap, and the capped maker must fill less than its order's full amount.
func TestPool_CalcAmtOut_BindingMakerBalance(t *testing.T) {
	t.Parallel()
	const (
		maker = "0x1111111111111111111111111111111111111111"
		other = "0x2222222222222222222222222222222222222222"
	)
	newOrder := func(id int64, makerAddr string) *order {
		return &order{
			ID:                    id,
			Maker:                 makerAddr,
			MakerAsset:            "B",
			TakerAsset:            "A",
			MakingAmount:          bignumber.NewBig10("10000000000"),
			TakingAmount:          bignumber.NewBig10("10000000000"),
			AvailableMakingAmount: bignumber.NewBig10("10000000000"),
			MakerBalanceAllowance: bignumber.NewBig10("10000000000"),
		}
	}
	sExtra, err := json.Marshal(Extra{BuyOrders: []*order{newOrder(1001, maker), newOrder(1002, other)}})
	require.NoError(t, err)
	p, err := NewPoolSimulator(entity.Pool{
		Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Reserves:    entity.PoolReserves{"0", "0"},
		StaticExtra: `{"ContractAddress":""}`,
		Extra:       string(sExtra),
	})
	require.NoError(t, err)

	amountIn := uint256.NewInt(3_000_000_000)
	quote := func(makerBalance *big.Int) (*pool.CalcAmtOutResult, *pool.CalcAmountOutResult) {
		t.Helper()
		limit := swaplimit.NewInventory("", swaplimit.ToU256Balances(map[string]*big.Int{
			NewMakerAndAsset(maker, "B"): makerBalance,
			NewMakerAndAsset(other, "B"): big.NewInt(10_000_000_000),
		}))
		native, err := p.CalcAmtOut(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: "A", Amount: *amountIn},
			TokenOut:      "B",
			Limit:         limit,
		})
		require.NoError(t, err)
		legacy, err := p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "A", Amount: amountIn.ToBig()},
			TokenOut:      "B",
			Limit:         pool.AsSwapLimitBig(limit),
		})
		require.NoError(t, err)
		return native, legacy
	}
	filledBy := func(res *pool.CalcAmtOutResult, id int64) string {
		for _, f := range res.SwapInfo.(SwapInfo).FilledOrders {
			if f.OrderID == id {
				return f.FilledMakingAmount
			}
		}
		return ""
	}

	unbound, unboundLegacy := quote(big.NewInt(10_000_000_000))
	capped, cappedLegacy := quote(big.NewInt(2_000_000_000))

	assert.Equal(t, unboundLegacy.TokenAmountOut.Amount.String(), unbound.TokenAmountOut.Amount.Dec())
	assert.Equal(t, cappedLegacy.TokenAmountOut.Amount.String(), capped.TokenAmountOut.Amount.Dec())
	assert.Equal(t, cappedLegacy.Fee.Amount.String(), capped.Fee.Amount.Dec())
	assert.Equal(t, cappedLegacy.Gas, capped.Gas)
	assert.Equal(t, "3000000000", filledBy(unbound, 1001), "unbound maker fills its full input share")
	assert.Equal(t, "2000000000", filledBy(capped, 1001), "the binding balance must cap maker fill at 2e9")
}
