package prmfun

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	// Desk side is listed as wrapped native; the executor unwraps before the call.
	testDeskToken = "0x0bd7d308f8e1639fab988df18a8011f41eacad73"
	memeToken     = "0x54c1fa485f182a17b3ba9190e7ed481b6b60109e"
)

// newTestSimulator builds a PoolSimulator from the same live on-chain snapshot
// (block 57379101) used in math_test.go's live-vector tests.
func newTestSimulator(t *testing.T) *PoolSimulator {
	t.Helper()
	ep := entity.Pool{
		Address:  "0xaea1eaf948e97581fbe9d8dea2951c327669af57",
		Exchange: DexType,
		Type:     DexType,
		Tokens:   []*entity.PoolToken{{Address: testDeskToken}, {Address: memeToken}},
		Reserves: []string{"0", "0"},
		StaticExtra: `{"rA":"0x08A59435c8359A45F4F5dC8D91DF893Cc33DaF29",` +
			`"cA":"0xaea1eaf948e97581fbe9d8dea2951c327669af57","mT":"` + memeToken + `",` +
			`"gD":"4200000000000000000","nQ":true}`,
		Extra: `{"ph":0,"vM":"1062330437710576052235807612","vD":"1405714531301211559",` +
			`"mS":"4336228956090614430859054","dR":"5714531301211559"}`,
	}
	s, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return s
}

func TestPoolSimulator_CalcAmountOut_Buy_MatchesLiveVector(t *testing.T) {
	t.Parallel()
	s := newTestSimulator(t)

	result, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testDeskToken, Amount: big.NewInt(1_000_000_000_000_000)},
		TokenOut:      memeToken,
	})
	require.NoError(t, err)
	require.Equal(t, "747638974590231692804552", result.TokenAmountOut.Amount.String())
	require.Equal(t, "10000000000000", result.Fee.Amount.String())
	require.Nil(t, result.RemainingTokenAmountIn)
}

func TestPoolSimulator_CalcAmountOut_Sell_MatchesLiveVector(t *testing.T) {
	t.Parallel()
	s := newTestSimulator(t)

	memeIn, _ := new(big.Int).SetString("1000000000000000000000", 10) // 1000 MEME
	result, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: memeToken, Amount: memeIn},
		TokenOut:      testDeskToken,
	})
	require.NoError(t, err)
	require.Equal(t, "1310003014677", result.TokenAmountOut.Amount.String())
	require.Equal(t, "13232353684", result.Fee.Amount.String())
}

func TestPoolSimulator_CalcAmountOut_InvalidToken(t *testing.T) {
	t.Parallel()
	s := newTestSimulator(t)

	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xnotapooltoken", Amount: big.NewInt(1)},
		TokenOut:      memeToken,
	})
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestPoolSimulator_CalcAmountOut_ZeroAmount(t *testing.T) {
	t.Parallel()
	s := newTestSimulator(t)

	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testDeskToken, Amount: big.NewInt(0)},
		TokenOut:      memeToken,
	})
	require.ErrorIs(t, err, ErrZeroAmount)
}

func TestPoolSimulator_CalcAmountOut_NotTrading(t *testing.T) {
	t.Parallel()
	s := newTestSimulator(t)
	s.phase = PhaseGraduated

	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testDeskToken, Amount: big.NewInt(1_000_000_000_000_000)},
		TokenOut:      memeToken,
	})
	require.ErrorIs(t, err, ErrPoolNotTrading)
}

func TestPoolSimulator_UpdateBalance_BuyThenSell_RoundTripsReserves(t *testing.T) {
	t.Parallel()
	s := newTestSimulator(t)

	buyResult, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testDeskToken, Amount: big.NewInt(1_000_000_000_000_000)},
		TokenOut:      memeToken,
	})
	require.NoError(t, err)

	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testDeskToken, Amount: big.NewInt(1_000_000_000_000_000)},
		TokenAmountOut: *buyResult.TokenAmountOut,
		Fee:            *buyResult.Fee,
		SwapInfo:       buyResult.SwapInfo,
	})

	require.Equal(t, "5083867930680846123663606", s.memeSold.String()) // memeSold(before) + memeOut
}

func TestPoolSimulator_CloneState_IsIndependent(t *testing.T) {
	t.Parallel()
	s := newTestSimulator(t)
	cloned := s.CloneState().(*PoolSimulator)

	cloned.virtualMeme.AddUint64(cloned.virtualMeme, 1)
	require.NotEqual(t, s.virtualMeme.String(), cloned.virtualMeme.String())
}

func TestPoolSimulator_NativeSwapSupport(t *testing.T) {
	s := newTestSimulator(t)

	t.Run("buy unwraps: desk in is native", func(t *testing.T) {
		require.True(t, s.SwapReceiveNativeIn(testDeskToken, memeToken, valueobject.ChainIDRobinhood))
		require.False(t, s.SwapReturnNativeOut(testDeskToken, memeToken, valueobject.ChainIDRobinhood))
	})

	t.Run("sell wraps: desk out is native", func(t *testing.T) {
		require.False(t, s.SwapReceiveNativeIn(memeToken, testDeskToken, valueobject.ChainIDRobinhood))
		require.True(t, s.SwapReturnNativeOut(memeToken, testDeskToken, valueobject.ChainIDRobinhood))
	})

	t.Run("wrong chain: desk token is not that chain's wrapped native", func(t *testing.T) {
		require.False(t, s.SwapReceiveNativeIn(testDeskToken, memeToken, valueobject.ChainIDEthereum))
		require.False(t, s.SwapReturnNativeOut(memeToken, testDeskToken, valueobject.ChainIDEthereum))
	})

	t.Run("unknown token is neither side", func(t *testing.T) {
		const unknown = "0x1111111111111111111111111111111111111111"
		require.False(t, s.SwapReceiveNativeIn(unknown, memeToken, valueobject.ChainIDRobinhood))
		require.False(t, s.SwapReturnNativeOut(memeToken, unknown, valueobject.ChainIDRobinhood))
	})
}

// Route optimizers may pass the originally offered input to UpdateBalance. It
// must apply accepted principal from SwapInfo after a graduation partial fill.
func TestAllPairsFinalBuyUsesAcceptedInput(t *testing.T) {
	for _, pair := range []string{testDeskToken, prmToken, stockToken} {
		t.Run(pair, func(t *testing.T) {
			s := newTestSimulator(t)
			s.Info.Tokens[0] = pair
			s.isNativeQuote = pair == testDeskToken
			offered := new(big.Int).Mul(s.graduationDesk.ToBig(), big.NewInt(2))
			before := s.stateCopy()
			params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: pair, Amount: offered}, TokenOut: memeToken}
			first, err := s.CalcAmountOut(params)
			require.NoError(t, err)
			second, err := s.CalcAmountOut(params)
			require.NoError(t, err)
			require.Equal(t, first, second)
			require.Equal(t, before, s.stateCopy())
			require.NotNil(t, first.RemainingTokenAmountIn)
			require.Positive(t, first.RemainingTokenAmountIn.Amount.Sign())
			require.Equal(t, pair, first.Fee.Token)
			require.Equal(t, pair == testDeskToken, first.SwapInfo.(*SwapInfo).IsNativeQuote)
			require.Greater(t, first.Gas, int64(buyGas))
			snapshot := first.SwapInfo.(*SwapInfo).NewState.DeskRaised.Dec()
			clone := s.CloneState().(*PoolSimulator)
			clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *first.TokenAmountOut, Fee: *first.Fee, SwapInfo: first.SwapInfo})
			require.Equal(t, s.graduationDesk.Dec(), clone.deskRaised.Dec())
			require.Equal(t, uSaleSupply.Dec(), clone.memeSold.Dec())
			require.Equal(t, PhaseGraduated, clone.phase)
			require.Equal(t, before, s.stateCopy())
			require.Equal(t, snapshot, first.SwapInfo.(*SwapInfo).NewState.DeskRaised.Dec())
			// Reusing the immutable quote does not add the input twice.
			clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
			require.Equal(t, s.graduationDesk.Dec(), clone.deskRaised.Dec())
			_, err = clone.CalcAmountOut(params)
			require.ErrorIs(t, err, ErrPoolNotTrading)
		})
	}
}

func TestAllPairsBuySellAndNativeCurrency(t *testing.T) {
	for _, pair := range []string{testDeskToken, prmToken, stockToken} {
		t.Run(pair, func(t *testing.T) {
			s := newTestSimulator(t)
			s.Info.Tokens[0] = pair
			s.isNativeQuote = pair == testDeskToken
			start := s.deskRaised.Clone()
			buy, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: pair, Amount: big.NewInt(1000000000000000)}, TokenOut: memeToken})
			require.NoError(t, err)
			s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: buy.SwapInfo})
			sell, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *buy.TokenAmountOut, TokenOut: pair})
			require.NoError(t, err)
			require.Equal(t, pair, sell.Fee.Token)
			s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: sell.SwapInfo})
			// Conservative rounding leaves at most one unit more tracked principal.
			require.LessOrEqual(t, new(uint256.Int).Sub(s.deskRaised, start).Uint64(), uint64(1))
			require.True(t, sell.TokenAmountOut.Amount.Cmp(big.NewInt(1000000000000000)) < 0)
			require.Equal(t, pair == testDeskToken, s.SwapReceiveNativeIn(pair, memeToken, valueobject.ChainIDRobinhood))
			require.Equal(t, pair == testDeskToken, s.SwapReturnNativeOut(memeToken, pair, valueobject.ChainIDRobinhood))
			require.False(t, s.SwapReceiveNativeIn(pair, pair, valueobject.ChainIDRobinhood))
		})
	}
}

func TestInvalidAmountsAndOverflowDoNotMutate(t *testing.T) {
	for _, amount := range []*big.Int{nil, big.NewInt(-1), big.NewInt(0), new(big.Int).Lsh(big.NewInt(1), 256)} {
		s := newTestSimulator(t)
		before := s.stateCopy()
		_, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: testDeskToken, Amount: amount}, TokenOut: memeToken})
		require.Error(t, err)
		require.Equal(t, before, s.stateCopy())
	}
	s := newTestSimulator(t)
	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: testDeskToken, Amount: big.NewInt(100)}, TokenOut: testDeskToken})
	require.ErrorIs(t, err, ErrInvalidToken)
	s.virtualDesk.SetAllOne()
	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: testDeskToken, Amount: big.NewInt(100)}, TokenOut: memeToken})
	require.ErrorIs(t, err, ErrOverflow)
}

func TestRejectIncompletePersistedState(t *testing.T) {
	p := testPool(t)
	for _, extra := range []string{`{}`, `{"ph":0,"vM":"0","vD":"1","mS":"0","dR":"0"}`, `{"ph":0,"vM":"1","vD":"1","mS":"0","dR":"0"}`} {
		p.Extra = extra
		_, err := NewPoolSimulator(p)
		require.Error(t, err)
	}
	s := newTestSimulator(t)
	encoded, err := json.Marshal(s.stateCopy())
	require.NoError(t, err)
	p.Extra = string(encoded)
	p.Reserves = []string{"invalid", "0"}
	_, err = NewPoolSimulator(p)
	require.ErrorIs(t, err, ErrInvalidState)
}
