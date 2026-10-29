package daikonprop

import (
	"math/big"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testBase  = "0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b" // VIRTUAL, 18 decimals
	testQuote = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913" // USDC, 6 decimals
)

// Base snapshot of the VIRTUAL/USDC pair at block 53240500. Every row of onChainQuotes is
// PAMMRouter.quote on a fork of that block, swapData = abi.encode(2, fee) (empty when fee is 0).
const testExtra = `{"baseDec":18,"quoteDec":6,"price":"608919187500000000","observedAtMs":1793270302118,` +
	`"ttl":15,"s1":25,"s2":40,"buyC1":"1060000000","buyC2":"10000000","sellC1":"1060000000",` +
	`"sellC2":"10000000","vaultBase":"2123923513991954515930","vaultQuote":"1192288307","paused":false}`

func newTestSim(t *testing.T, extra string, staleCheck bool, feePpm uint64) *PoolSimulator {
	t.Helper()
	se, _ := json.Marshal(StaticExtra{Router: "0xe0143476a279fc54119ad2ce14c46b5511e6022c", PartnerID: 2, FeePpm: feePpm})
	sim, err := NewPoolSimulator(pool.FactoryParams{
		EntityPool: entity.Pool{
			Address: "0x8549db95091575f7aa240b8a287860a2a75994e8", Exchange: "daikon-prop", Type: DexType,
			Reserves: entity.PoolReserves{"2123923513991954515930", "1192288307"},
			Tokens:   []*entity.PoolToken{{Address: testBase}, {Address: testQuote}},
			Extra:    extra, StaticExtra: string(se),
		},
		Opts: pool.FactoryOpts{StaleCheck: staleCheck},
	})
	require.NoError(t, err)
	return sim
}

func calc(sim *PoolSimulator, buy bool, amt string) (*pool.CalcAmountOutResult, error) {
	in, out := testBase, testQuote
	if buy {
		in, out = out, in
	}
	a, _ := new(big.Int).SetString(amt, 10)
	return sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: a}, TokenOut: out})
}

// Pins the pair's rounding (ceil spread price, one final floor, ceil fees, mid-price capacity
// split) to on-chain quotes. Empty want means the quote reverts.
func TestQuoteMatchesOnChain(t *testing.T) {
	onChainQuotes := []struct {
		buy    bool
		amount string
		fee    uint64
		want   string
	}{
		{true, "1", 0, "1638158650426"},
		{true, "1000000", 0, "1638158650426769623"},
		{true, "1000000", 2500, "1634063253800702699"},
		{true, "3333337", 2500, "5446882427144960277"},
		{true, "100000000", 0, "163815865042676962332"},
		{true, "100000000", 50000, "155625071790543114215"},
		{true, "100000000", 60000, "155625071790543114215"},
		{true, "1060000000", 0, "1736448169452375800719"},
		{true, "1065000000", 2500, "1740271644387620826047"},
		{true, "1070000000", 0, "1752805281474814411748"},
		{true, "1070000001", 0, ""},
		{true, "2000000000", 0, ""},
		{true, "999999999", 1, "1638157010629960546123"},
		{true, "12345", 777, "20206686953014203"},
		{false, "1", 0, ""},
		{false, "1000000000000000", 0, "607"},
		{false, "1000000000000000000", 0, "607396"},
		{false, "1000000000000000000", 2500, "605877"},
		{false, "7777777777777777777", 2500, "4712387"},
		{false, "100000000000000000000", 0, "60739688"},
		{false, "1000000000000000000000", 50000, "577027044"},
		{false, "1000000000000000000000", 60000, "577027044"},
		{false, "1740000000000000000000", 0, "1056870587"},
		{false, "1750000000000000000000", 0, "1062936143"},
		{false, "1760000000000000000000", 2500, ""},
		{false, "1766000000000000000000", 0, ""},
		{false, "1768000000000000000000", 0, ""},
		{false, "5000000000000000000000", 0, ""},
		{false, "123456789012345678", 999, "74912"},
	}
	for _, c := range onChainQuotes {
		res, err := calc(newTestSim(t, testExtra, false, c.fee), c.buy, c.amount)
		if c.want == "" {
			assert.Error(t, err, "buy=%v amount=%s fee=%d", c.buy, c.amount, c.fee)
			continue
		}
		require.NoError(t, err, "buy=%v amount=%s fee=%d", c.buy, c.amount, c.fee)
		assert.Equal(t, c.want, res.TokenAmountOut.Amount.String(), "buy=%v amount=%s fee=%d", c.buy, c.amount, c.fee)
	}
}

// The observation expired long ago: StaleCheck rejects it, the indexer (StaleCheck=false) does not.
func TestStaleCheck(t *testing.T) {
	_, err := calc(newTestSim(t, testExtra, true, 0), true, "1000000")
	assert.ErrorIs(t, err, ErrExpired)
	_, err = calc(newTestSim(t, testExtra, false, 0), true, "1000000")
	assert.NoError(t, err)
}

func TestNotTrading(t *testing.T) {
	paused := strings.Replace(testExtra, `"paused":false`, `"paused":true`, 1)
	_, err := calc(newTestSim(t, paused, false, 0), true, "1000000")
	assert.ErrorIs(t, err, ErrNotTrading)
	unpublished := strings.Replace(testExtra, `"observedAtMs":1793270302118`, `"observedAtMs":0`, 1)
	_, err = calc(newTestSim(t, unpublished, false, 0), true, "1000000")
	assert.ErrorIs(t, err, ErrNotTrading)
}

func TestVaultInventoryLimitsOutput(t *testing.T) {
	poor := strings.Replace(testExtra, `"vaultQuote":"1192288307"`, `"vaultQuote":"1000000"`, 1)
	_, err := calc(newTestSim(t, poor, false, 0), false, "100000000000000000000") // pays 60.7 USDC
	assert.ErrorIs(t, err, ErrVaultInventory)
}

// Capacity use matches a fork run: buying 50 USDC drops buyC1 by 50e6, and selling 10 VIRTUAL
// drops sellC1 by ceil(10 * mid price) in quote units. A clone must not see the original's spend.
func TestUpdateBalanceSpendsCapacity(t *testing.T) {
	sim := newTestSim(t, testExtra, false, 0)
	clone := sim.CloneState().(*PoolSimulator)

	for _, c := range []struct {
		buy    bool
		amount string
		check  func(s *PoolSimulator) string
		want   string
	}{
		{true, "50000000", func(s *PoolSimulator) string { return s.BuyC1.String() }, "1010000000"},
		{false, "10000000000000000000", func(s *PoolSimulator) string { return s.SellC1.String() }, "1053910808"},
	} {
		res, err := calc(sim, c.buy, c.amount)
		require.NoError(t, err)
		in, _ := new(big.Int).SetString(c.amount, 10)
		tokenIn := testBase
		if c.buy {
			tokenIn = testQuote
		}
		sim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: in},
			TokenAmountOut: *res.TokenAmountOut, SwapInfo: res.SwapInfo,
		})
		assert.Equal(t, c.want, c.check(sim))
	}
	assert.Equal(t, "1060000000", clone.BuyC1.String())
	assert.Equal(t, "1060000000", clone.SellC1.String())
	assert.Equal(t, "2123923513991954515930", clone.VaultBase.Dec())
}

// A per-swap fee (the alpha fee path) must price exactly like the static fee the same on-chain
// rows were taken with, in both directions and above the 5% cap.
func TestCalcAmountOutWithFeeMatchesOnChain(t *testing.T) {
	sim := newTestSim(t, testExtra, false, 0)
	for _, c := range []struct {
		buy         bool
		amount, out string
		fee         uint64
	}{
		{true, "1000000", "1634063253800702699", 2500},
		{true, "12345", "20206686953014203", 777},
		{true, "100000000", "155625071790543114215", 60000},
		{false, "1000000000000000000", "605877", 2500},
		{false, "123456789012345678", "74912", 999},
		{false, "1000000000000000000000", "577027044", 60000},
	} {
		in, out := testBase, testQuote
		if c.buy {
			in, out = out, in
		}
		a, _ := new(big.Int).SetString(c.amount, 10)
		res, err := sim.CalcAmountOutWithFee(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in, Amount: a}, TokenOut: out}, c.fee)
		require.NoError(t, err)
		assert.Equal(t, c.out, res.TokenAmountOut.Amount.String(), "buy=%v amount=%s fee=%d", c.buy, c.amount, c.fee)
	}
}
