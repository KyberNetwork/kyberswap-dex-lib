package ilyrislb

import (
	"os"
	"strconv"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testdata/oracle.json records the real BinPool's quoteExactIn and priceFromId outputs on an
// in-process EVM, so these compare the Go port with the contract, not with another port.
type oracleVector struct {
	BinStepBps            uint32               `json:"binStepBps"`
	SwapFeeBps            uint64               `json:"swapFeeBps"`
	ActiveID              int32                `json:"activeId"`
	DecimalsX             uint8                `json:"decimalsX"`
	DecimalsY             uint8                `json:"decimalsY"`
	VariableFeeControl    uint32               `json:"variableFeeControl"`
	VolatilityAccumulator uint32               `json:"volatilityAccumulator"`
	TotalFeeRate          string               `json:"totalFeeRate"`
	Bins                  [][3]json.RawMessage `json:"bins"`
	Prices                map[string]string    `json:"prices"`
	ExactIn               []struct {
		XForY     bool   `json:"xForY"`
		AmountIn  string `json:"amountIn"`
		AmountOut string `json:"amountOut"`
		FeeAmount string `json:"feeAmount"`
		FinalID   int32  `json:"finalId"`
	} `json:"exactIn"`
}

func loadOracle(t *testing.T) []oracleVector {
	raw, err := os.ReadFile("testdata/oracle.json")
	require.NoError(t, err)
	var vectors []oracleVector
	require.NoError(t, json.Unmarshal(raw, &vectors))
	require.NotEmpty(t, vectors)
	return vectors
}

// oracleSim builds a simulator for a vector. The oracle pool's base fee is swapFeeBps*1e5.
func oracleSim(t *testing.T, v oracleVector) *PoolSimulator {
	p := &PoolSimulator{binStepBps: v.BinStepBps, activeID: v.ActiveID,
		feeRate: min(v.SwapFeeBps*100_000+variableFeeRate(v.VariableFeeControl, v.BinStepBps, v.VolatilityAccumulator), maxFeeRate)}
	decimalFactor(&p.decimalFactor, v.DecimalsX, v.DecimalsY)
	for _, raw := range v.Bins {
		var b Bin
		var x, y string
		require.NoError(t, json.Unmarshal(raw[0], &b.ID))
		require.NoError(t, json.Unmarshal(raw[1], &x))
		require.NoError(t, json.Unmarshal(raw[2], &y))
		require.NoError(t, b.ReserveX.SetFromDecimal(x))
		require.NoError(t, b.ReserveY.SetFromDecimal(y))
		p.bins = append(p.bins, b)
	}
	return p
}

func TestPriceFromIDMatchesContract(t *testing.T) {
	var got uint256.Int
	for _, v := range loadOracle(t) {
		for idStr, want := range v.Prices {
			id, err := strconv.ParseInt(idStr, 10, 32)
			require.NoError(t, err)
			require.NoError(t, priceFromID(&got, v.BinStepBps, int32(id)))
			assert.Equal(t, want, got.Dec(), "binStep %d id %d", v.BinStepBps, id)
		}
	}
	// Live BinPool.priceFromId on Robinhood, far outside the oracle's id range.
	for _, c := range []struct {
		step uint32
		id   int32
		want string
	}{
		{10, 7796, "2421408311338001783650"}, {10, 7877, "2625598433470790466360"},
		{10, 8016, "3016928892677849546775"}, {10, -7796, "412982806459199"},
		{250, 577, "1540530606381114177903360"}, {250, 697, "29821822301846452059491583"},
		{250, -858, "629395297"},
	} {
		require.NoError(t, priceFromID(&got, c.step, c.id))
		assert.Equal(t, c.want, got.Dec(), "binStep %d id %d", c.step, c.id)
	}
}

func TestFeeRateMatchesContract(t *testing.T) {
	for _, v := range loadOracle(t) {
		if v.TotalFeeRate != "" {
			assert.Equal(t, v.TotalFeeRate, strconv.FormatUint(oracleSim(t, v).feeRate, 10))
		}
	}
	// A surcharge past the cap clamps at MAX_FEE_RATE instead of exceeding it.
	f := FeeParams{BaseFactor: 1000, FilterPeriod: 30, VariableFeeControl: 1_000_000, MaxVolatilityAccumulator: 350_000, VolatilityReference: 350_000}
	assert.EqualValues(t, maxFeeRate, f.totalFeeRate(100, 0, 0))
}

func TestQuoteExactInMatchesContract(t *testing.T) {
	for i, v := range loadOracle(t) {
		p := oracleSim(t, v)
		for j, c := range v.ExactIn {
			var out, fee uint256.Int
			finalID, _, err := p.quoteExactIn(c.XForY, uint256.MustFromDecimal(c.AmountIn), &out, &fee)
			require.NoError(t, err, "vector %d case %d", i, j)
			assert.Equal(t, c.AmountOut, out.Dec(), "vector %d case %d amountOut", i, j)
			assert.Equal(t, c.FeeAmount, fee.Dec(), "vector %d case %d fee", i, j)
			assert.Equal(t, c.FinalID, finalID, "vector %d case %d finalId", i, j)
		}
	}
}
