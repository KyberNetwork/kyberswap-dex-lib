package flywheelfun

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func splitStrings(s Split) []string {
	return []string{s.Net.Dec(), s.Rewards.Dec(), s.Burn.Dec(), s.Liquidity.Dec(), s.Platform.Dec()}
}
func TestSolidityFeeReference(t *testing.T) {
	data, err := os.ReadFile("testdata/fee-reference.json")
	require.NoError(t, err)
	var fixture struct {
		SourceSHA256 string
		Cases        []struct {
			Amount         string
			Protocol       uint32
			Platform       string
			Buy, Sell, ETH []string
			Reverts        bool
		}
	}
	require.NoError(t, json.Unmarshal(data, &fixture))
	require.Len(t, fixture.Cases, 390)
	for _, c := range fixture.Cases {
		x, err := uint256.FromDecimal(c.Amount)
		require.NoError(t, err)
		platform, e1 := BuyPlatform(x, c.Protocol)
		buy, e2 := BuyQuote(x, c.Protocol)
		sell, e3 := SellQuote(x, c.Protocol)
		eth, e4 := SellETH(&sell.Net, x, c.Protocol)
		if c.Reverts {
			require.True(t, e1 != nil || e2 != nil || e3 != nil || e4 != nil, "expected Solidity revert: %s/%d", c.Amount, c.Protocol)
			continue
		}
		require.NoError(t, e1)
		require.NoError(t, e2)
		require.NoError(t, e3)
		require.NoError(t, e4)
		require.Equal(t, c.Platform, platform.Dec())
		require.Equal(t, c.Buy, splitStrings(buy))
		require.Equal(t, c.Sell, splitStrings(sell))
		require.Equal(t, c.ETH, splitStrings(eth))
	}
}
func TestArithmeticGuards(t *testing.T) {
	_, err := BuyPlatform(uint256.NewInt(100), 1001)
	require.ErrorIs(t, err, ErrProtocol)
	_, err = BuyQuote(uint256.NewInt(100), 1001)
	require.ErrorIs(t, err, ErrProtocol)
	_, err = SellQuote(uint256.NewInt(100), 1001)
	require.ErrorIs(t, err, ErrProtocol)
	_, err = SellETH(uint256.NewInt(1), new(uint256.Int), 0)
	require.ErrorIs(t, err, ErrAmount)
	max := new(uint256.Int).SetAllOne()
	_, err = mulDiv(max, max, uint256.NewInt(1), false)
	require.ErrorIs(t, err, ErrMath)
	_, err = mulDiv(max, max, new(uint256.Int), false)
	require.ErrorIs(t, err, ErrMath)
}
func testCurve() CurveState {
	return CurveState{QuoteReserve: *uint256.NewInt(100000), TokenReserve: *uint256.NewInt(100000000), VirtualQuote: *uint256.NewInt(50000), Threshold: *uint256.NewInt(1000000), Invariant: *uint256.NewInt(10000000000000)}
}
func TestCurveRoundingAndLimits(t *testing.T) {
	c := testCurve()
	out, next, err := CurveTrade(c, uint256.NewInt(999), true)
	require.NoError(t, err)
	// Independent arbitrary-precision reference, including ceil rounding.
	numerator := new(big.Int).Mul(c.QuoteReserve.ToBig(), c.TokenReserve.ToBig())
	denom := new(big.Int).Add(c.QuoteReserve.ToBig(), big.NewInt(999))
	expected := new(big.Int).Sub(c.TokenReserve.ToBig(), new(big.Int).Div(new(big.Int).Add(numerator, new(big.Int).Sub(denom, big.NewInt(1))), denom))
	require.Equal(t, expected, out.ToBig())
	require.Equal(t, testCurve(), c)
	returned, _, err := CurveTrade(next, &out, false)
	require.NoError(t, err)
	require.LessOrEqual(t, returned.Uint64(), uint64(999))
	_, _, err = CurveTrade(c, uint256.NewInt(950000), true)
	require.ErrorIs(t, err, ErrPartial)
	_, _, err = CurveTrade(c, uint256.NewInt(200000000), false)
	require.ErrorIs(t, err, ErrCurve)
	_, _, err = CurveTrade(c, new(uint256.Int), true)
	require.ErrorIs(t, err, ErrCurve)
	c.Graduated = true
	_, _, err = CurveTrade(c, uint256.NewInt(100), true)
	require.ErrorIs(t, err, ErrCurve)
}
func testEntity() entity.Pool {
	c := testCurve()
	extra, _ := json.Marshal(Extra{Curve: c, Valid: true})
	static, _ := json.Marshal(StaticExtra{Factory: Factory, Settlement: Settlement, Quote: WETH})
	token := "0x1111111111111111111111111111111111111111"
	return entity.Pool{Address: token, Type: DexType, Exchange: DexType, BlockNumber: 76791655, Extra: string(extra), StaticExtra: string(static), Tokens: []*entity.PoolToken{{Address: WETH}, {Address: token}}, Reserves: entity.PoolReserves{"50000", "100000000"}}
}
func TestSimulatorPurityCloneAndStateUpdate(t *testing.T) {
	s, err := NewPoolSimulator(testEntity())
	require.NoError(t, err)
	p := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: big.NewInt(1000)}, TokenOut: s.Info.Address}
	first, err := s.CalcAmountOut(p)
	require.NoError(t, err)
	second, err := s.CalcAmountOut(p)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, testCurve(), s.Curve)
	clone := s.CloneState().(*PoolSimulator)
	s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
	require.NotEqual(t, clone.Curve, s.Curve)
	require.Equal(t, testCurve(), clone.Curve)
	s.Info.Reserves[0].SetInt64(1)
	require.NotEqual(t, s.Info.Reserves, clone.Info.Reserves)
	sell, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *first.TokenAmountOut, TokenOut: WETH})
	require.NoError(t, err)
	require.True(t, sell.TokenAmountOut.Amount.Cmp(big.NewInt(1000)) < 0)
	s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
	_, err = s.CalcAmountOut(p)
	require.Error(t, err)
}
func TestUnsupportedSnapshotsFailClosed(t *testing.T) {
	p := testEntity()
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	extra.Curve.Graduated = true
	b, _ := json.Marshal(extra)
	p.Extra = string(b)
	_, err := NewPoolSimulator(p)
	require.ErrorIs(t, err, ErrUnsupported)
	p = testEntity()
	p.StaticExtra = `{"factory":"` + Factory + `","settlement":"` + Settlement + `","quote":"0x73c2de14c7fa0a57cc2d9722b959ea70b881ffe4"}`
	_, err = NewPoolSimulator(p)
	require.ErrorIs(t, err, ErrUnsupported)
	p = testEntity()
	p.BlockNumber = 0
	_, err = NewPoolSimulator(p)
	require.ErrorIs(t, err, ErrState)
}
func TestMalformedSwapInputs(t *testing.T) {
	s, err := NewPoolSimulator(testEntity())
	require.NoError(t, err)
	for _, amount := range []*big.Int{nil, big.NewInt(0), big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 256)} {
		_, err = s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: amount}, TokenOut: s.Info.Address})
		require.Error(t, err)
	}
	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: big.NewInt(100)}, TokenOut: WETH})
	require.Error(t, err)
	require.Equal(t, testCurve(), s.Curve)
}
