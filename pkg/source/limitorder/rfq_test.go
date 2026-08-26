package limitorder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// opSigServer answers /orders/operator-signature with a signature for every requested id
// present in signed, mimicking the service dropping ids that are no longer active.
func opSigServer(t *testing.T, signed map[int64]bool) *httptest.Server {
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

	srv := opSigServer(t, signed)
	t.Cleanup(srv.Close)

	cfg := &Config{LimitOrderHTTPUrl: srv.URL, ChainID: 1}
	h := &RFQHandler{config: cfg, client: NewHTTPClientWithConfig(cfg)}

	return h.RFQ(t.Context(), pool.RFQParams{
		Recipient: "0x2222222222222222222222222222222222222222",
		SwapInfo:  SwapInfo{AmountIn: "1000", FilledOrders: orders},
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

// A fallback order is spare depth the fill does not need. When the limit-order service stops
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

	t.Run("dead required order still fails", func(t *testing.T) {
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
