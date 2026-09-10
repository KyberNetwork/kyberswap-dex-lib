package prmfun

import (
	"math/big"
	"testing"

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
			`"gD":"4200000000000000000"}`,
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
