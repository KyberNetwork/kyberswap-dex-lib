package limitorder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// signedOrdersSigServer answers /orders/operator-signature with a signature for every
// requested id present in signed, mimicking the service dropping ids that are no longer
// active. Distinct from http_client_test.go's opSigServer type, which records requests.
func signedOrdersSigServer(t *testing.T, signed map[int64]bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sigs []map[string]any
		for _, raw := range r.URL.Query()["orderIds"] {
			var id int64
			if _, err := fmt.Sscan(raw, &id); err != nil || !signed[id] {
				continue
			}
			sigs = append(sigs, map[string]any{
				"id":                         id,
				"chainId":                    "1",
				"operatorSignature":          "aabb",
				"operatorSignatureExpiredAt": 1,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"orders": sigs},
		})
	}))
}

func filledOrder(id int64, isFallback bool) *FilledOrderInfo {
	return &FilledOrderInfo{
		OrderID:    id,
		OrderHash:  fmt.Sprintf("0x%d", id),
		Maker:      "0x1111111111111111111111111111111111111111",
		Receiver:   "0x1111111111111111111111111111111111111111",
		IsFallBack: isFallback,
	}
}

func rfqWith(t *testing.T, signed map[int64]bool, orders []*FilledOrderInfo) (*pool.RFQResult, error) {
	t.Helper()

	return rfqWithSwapInfo(t, signed, SwapInfo{AmountIn: "1000", FilledOrders: orders})
}

func rfqWithSwapInfo(t *testing.T, signed map[int64]bool, swapInfo SwapInfo) (*pool.RFQResult, error) {
	t.Helper()

	srv := signedOrdersSigServer(t, signed)
	t.Cleanup(srv.Close)

	cfg := &Config{LimitOrderHTTPUrl: srv.URL, ChainID: 1}
	h := &RFQHandler{config: cfg, client: NewHTTPClientWithConfig(cfg)}

	return h.RFQ(t.Context(), pool.RFQParams{
		Recipient: "0x2222222222222222222222222222222222222222",
		SwapInfo:  swapInfo,
	})
}

func keptOrderIDs(t *testing.T, res *pool.RFQResult) []int64 {
	t.Helper()

	extra, ok := res.Extra.(OpSignatureExtra)
	require.True(t, ok, "extra should be OpSignatureExtra")

	ids := make([]int64, 0, len(extra.FilledOrders))
	for _, o := range extra.FilledOrders {
		ids = append(ids, o.OrderID)
	}

	return ids
}

// A fallback order is spare depth the fill does not need. When the operator-signature API stops
// signing it — filled, cancelled or expired since the pool snapshot — dropping it keeps the
// encode alive, where before the encoder failed the whole route over it.
func TestRFQ_DropsUnsignedFallbackOrders(t *testing.T) {
	t.Run("dead fallback order is dropped, the rest survive", func(t *testing.T) {
		res, err := rfqWith(t,
			map[int64]bool{1: true, 2: true},
			[]*FilledOrderInfo{filledOrder(1, false), filledOrder(2, true), filledOrder(3, true)},
		)
		require.NoError(t, err)
		assert.Equal(t, []int64{1, 2}, keptOrderIDs(t, res))
	})

	t.Run("dead required order in a quote without recorded room still fails", func(t *testing.T) {
		_, err := rfqWith(t,
			map[int64]bool{2: true},
			[]*FilledOrderInfo{filledOrder(1, false), filledOrder(2, true)},
		)
		require.ErrorIs(t, err, ErrMissingOperatorSignature)
	})

	t.Run("every order signed keeps the route intact", func(t *testing.T) {
		res, err := rfqWith(t,
			map[int64]bool{1: true, 2: true, 3: true},
			[]*FilledOrderInfo{filledOrder(1, false), filledOrder(2, true), filledOrder(3, true)},
		)
		require.NoError(t, err)
		assert.Equal(t, []int64{1, 2, 3}, keptOrderIDs(t, res))
		assert.Nil(t, res.NewAmountOut, "quoted amount out stands")
	})

	t.Run("all fallbacks dead leaves only the required orders", func(t *testing.T) {
		res, err := rfqWith(t,
			map[int64]bool{1: true},
			[]*FilledOrderInfo{filledOrder(1, false), filledOrder(2, true), filledOrder(3, true)},
		)
		require.NoError(t, err)
		assert.Equal(t, []int64{1}, keptOrderIDs(t, res))
	})
}

func TestRFQ_RejectsRecipientEqualToMaker(t *testing.T) {
	order := filledOrder(1, false)
	order.Receiver = ""
	order.Maker = "0x2222222222222222222222222222222222222222"

	_, err := rfqWith(t, map[int64]bool{1: true}, []*FilledOrderInfo{order})
	require.ErrorIs(t, err, ErrSameSenderMaker)
	assert.True(t, strings.Contains(err.Error(), "recipient"))
}

// freshOrder is an order with no fills before the quote, so its remaining amounts are the whole order.
func freshOrder(id int64, isFallback bool, maker, making, taking, filledMaking, filledTaking string) *FilledOrderInfo {
	o := filledOrder(id, isFallback)
	o.Maker, o.Receiver = maker, maker
	o.MakingAmount, o.TakingAmount = making, taking
	o.RemainingMakingAmount = making
	o.FilledMakingAmount, o.FilledTakingAmount, o.FeeAmount = filledMaking, filledTaking, "0"

	return o
}

type fill struct{ taking, making string }

func fillsOf(orders []*FilledOrderInfo) map[int64]fill {
	fills := make(map[int64]fill)
	for _, o := range orders {
		if !o.IsFallBack {
			fills[o.OrderID] = fill{o.FilledTakingAmount, o.FilledMakingAmount}
		}
	}

	return fills
}

// A required order the service no longer signs is replaced by the signed orders after it, the
// way the executor skips an order it can't fill on-chain, and the lower amount out is reported.
func TestRFQ_RefillsDroppedRequiredOrder(t *testing.T) {
	const (
		m1 = "0xa000000000000000000000000000000000000001"
		m2 = "0xa000000000000000000000000000000000000002"
		m3 = "0xa000000000000000000000000000000000000003"
	)
	signed := map[int64]bool{2: true, 3: true}

	t.Run("later orders take the dropped order's share", func(t *testing.T) {
		res, err := rfqWith(t, signed, []*FilledOrderInfo{
			freshOrder(1, false, m1, "600", "600", "600", "600"),
			freshOrder(2, false, m2, "700", "700", "400", "400"),
			freshOrder(3, true, m3, "500", "1000", "0", "0"),
		})
		require.NoError(t, err)
		assert.Equal(t, []int64{2, 3}, keptOrderIDs(t, res))
		// order 2 fills its whole 700 room, order 3 takes the last 300 at 0.5
		assert.Equal(t, "850", res.NewAmountOut.String())
		assert.Equal(t, map[int64]fill{2: {"700", "700"}, 3: {"300", "150"}},
			fillsOf(res.Extra.(OpSignatureExtra).FilledOrders))
	})

	t.Run("not enough signed room still fails", func(t *testing.T) {
		_, err := rfqWith(t, signed, []*FilledOrderInfo{
			freshOrder(1, false, m1, "600", "600", "600", "600"),
			freshOrder(2, false, m2, "700", "700", "400", "400"),
			freshOrder(3, true, m3, "100", "200", "0", "0"),
		})
		require.ErrorIs(t, err, ErrMissingOperatorSignature)
	})

	// Orders 2 and 3 share maker m2. Order 2 now fills its whole 700, which leaves order 3
	// whatever of m2's balance is still free.
	sameMaker := func(balance string) []*FilledOrderInfo {
		orders := []*FilledOrderInfo{
			freshOrder(1, false, m1, "600", "600", "600", "600"),
			freshOrder(2, false, m2, "700", "700", "400", "400"),
			freshOrder(3, true, m2, "300", "600", "0", "0"),
		}
		orders[1].MakerBalance, orders[2].MakerBalance = balance, balance
		return orders
	}
	t.Run("a maker's later order gets only the balance its earlier one left", func(t *testing.T) {
		_, err := rfqWith(t, signed, sameMaker("700"))
		require.ErrorIs(t, err, ErrMissingOperatorSignature)
	})
	t.Run("a maker with spare balance keeps its later order", func(t *testing.T) {
		res, err := rfqWith(t, signed, sameMaker("10000"))
		require.NoError(t, err)
		assert.Equal(t, "850", res.NewAmountOut.String())
	})
}

// A dropped order may have been filled on-chain after the quote, taking up to its whole own
// remaining making from its maker. Order 2 (m1) had 800 left but filled only 500 in the quote;
// fallback order 3 (m1) was capped to the 500 of m1's 1000 balance that order 2 left.
func TestRFQ_RefillAssumesDroppedOrderDrainedItsMaker(t *testing.T) {
	const (
		m1 = "0xa000000000000000000000000000000000000001"
		m2 = "0xa000000000000000000000000000000000000002"
	)
	orders := func(m1Balance string) []*FilledOrderInfo {
		o := []*FilledOrderInfo{
			freshOrder(1, false, m2, "500", "500", "500", "500"),
			freshOrder(2, false, m1, "800", "800", "500", "500"),
			freshOrder(3, true, m1, "500", "500", "0", "0"),
		}
		o[1].MakerBalance, o[2].MakerBalance = m1Balance, m1Balance
		return o
	}
	signed := map[int64]bool{1: true, 3: true}

	t.Run("not enough left after the dropped order's whole remaining", func(t *testing.T) {
		// m1 may have only 1000 - 800 = 200 left, short of the 500 order 3 needs to supply
		_, err := rfqWith(t, signed, orders("1000"))
		require.ErrorIs(t, err, ErrMissingOperatorSignature)
	})
	t.Run("enough left even if the dropped order filled entirely", func(t *testing.T) {
		res, err := rfqWith(t, signed, orders("2000"))
		require.NoError(t, err)
		assert.Equal(t, "1000", res.NewAmountOut.String())
		assert.Equal(t, map[int64]fill{1: {"500", "500"}, 3: {"500", "500"}},
			fillsOf(res.Extra.(OpSignatureExtra).FilledOrders))
	})
}

// A cancelled dropped order leaves its maker's balance to the maker's worse order, which then
// crowds out a better order after it; a filled one would not. The refill reports the lower.
func TestRFQ_RefillKeepsLowerOfCancelledAndFilledDroppedOrder(t *testing.T) {
	const (
		m1 = "0xa000000000000000000000000000000000000001"
		m2 = "0xa000000000000000000000000000000000000002"
	)
	o := []*FilledOrderInfo{
		freshOrder(1, false, m1, "500", "500", "500", "500"),
		freshOrder(2, false, m1, "500", "1000", "250", "500"),
		freshOrder(3, true, m2, "2000", "1000", "0", "0"),
	}
	o[0].MakerBalance, o[1].MakerBalance = "700", "700"

	// cancelled: order 2 gets m1's 700 and takes all of amountIn for 500; filled: m1 has 200
	// left, order 2 takes 400 for 200 and order 3 the other 600 for 1200
	res, err := rfqWith(t, map[int64]bool{2: true, 3: true}, o)
	require.NoError(t, err)
	assert.Equal(t, "500", res.NewAmountOut.String())
	assert.Equal(t, map[int64]fill{2: {"1000", "500"}}, fillsOf(res.Extra.(OpSignatureExtra).FilledOrders))
}

// Dropping a required order must give the fill the simulator finds on the pool without it, as
// long as that fill stays within the orders the quote carried; otherwise the RFQ still fails.
func TestRFQ_RefillMatchesSimulatorWithoutDroppedOrder(t *testing.T) {
	newOrder := func(id int64, maker, making, taking string, feePct uint32, takerAssetFee bool) *order {
		return &order{
			ID:                    id,
			Maker:                 maker,
			MakingAmount:          bignumber.NewBig10(making),
			TakingAmount:          bignumber.NewBig10(taking),
			AvailableMakingAmount: bignumber.NewBig10(making),
			MakerBalanceAllowance: bignumber.NewBig10("1000000000000"),
			MakerTokenFeePercent:  feePct,
			IsTakerAssetFee:       takerAssetFee,
		}
	}
	// Distinct makers: the same-maker room cap is deliberately conservative (see above).
	orders := []*order{
		newOrder(101, "0xb01", "1000", "1000", 30, false), // maker-asset fee
		newOrder(102, "0xb02", "800", "900", 0, false),
		newOrder(103, "0xb03", "500", "600", 100, true), // taker-asset fee
		newOrder(104, "0xb04", "700", "1000", 0, false),
		newOrder(105, "0xb05", "400", "800", 0, false),
	}
	quote := func(t *testing.T, skip int64, amountIn string) (*pool.CalcAmountOutResult, error) {
		kept := slices.DeleteFunc(slices.Clone(orders), func(o *order) bool { return o.ID == skip })
		extra, err := json.Marshal(Extra{BuyOrders: kept})
		require.NoError(t, err)
		sim, err := NewPoolSimulator(entity.Pool{
			Tokens:      []*entity.PoolToken{{Address: "a"}, {Address: "b"}},
			Reserves:    entity.PoolReserves{"0", "0"},
			StaticExtra: `{"ContractAddress":""}`,
			Extra:       string(extra),
		})
		require.NoError(t, err)

		return sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: "a", Amount: bignumber.NewBig10(amountIn)},
			TokenOut:      "b",
			Limit:         swaplimit.NewInventory("", sim.CalculateLimit()),
		})
	}

	refilled := 0
	for _, amountIn := range []string{"1500", "2400", "2600", "3000"} {
		res, err := quote(t, 0, amountIn)
		require.NoError(t, err)
		quoted := res.SwapInfo.(SwapInfo)

		for _, dropped := range quoted.FilledOrders {
			if dropped.IsFallBack {
				continue
			}
			t.Run(fmt.Sprintf("amountIn %s drops %d", amountIn, dropped.OrderID), func(t *testing.T) {
				signed := make(map[int64]bool, len(quoted.FilledOrders))
				for _, o := range quoted.FilledOrders {
					signed[o.OrderID] = o.OrderID != dropped.OrderID
				}
				got, err := rfqWithSwapInfo(t, signed, quoted)

				want, wantErr := quote(t, dropped.OrderID, amountIn)
				if wantErr != nil || slices.ContainsFunc(want.SwapInfo.(SwapInfo).FilledOrders, func(o *FilledOrderInfo) bool {
					return !o.IsFallBack && !signed[o.OrderID]
				}) {
					require.ErrorIs(t, err, ErrMissingOperatorSignature)
					return
				}
				require.NoError(t, err)
				refilled++
				assert.Equal(t, want.TokenAmountOut.Amount.String(), got.NewAmountOut.String())
				assert.Equal(t, fillsOf(want.SwapInfo.(SwapInfo).FilledOrders),
					fillsOf(got.Extra.(OpSignatureExtra).FilledOrders))
			})
		}
	}
	assert.Positive(t, refilled, "some drop should be refillable")
}
