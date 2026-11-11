package ammalgam

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testTokenX = "0x0000000000000000000000000000000000000001"
	testTokenY = "0x0000000000000000000000000000000000000002"
	testPair   = "0x0000000000000000000000000000000000000003"
)

func simulatorFixture(t *testing.T, reserves [2]string, references [2]string, missing [2]string) entity.Pool {
	t.Helper()
	extra := Extra{}
	for i := range 2 {
		extra.ReferenceReserves[i] = *uint256.MustFromDecimal(references[i])
		extra.MissingAssets[i] = *uint256.MustFromDecimal(missing[i])
	}
	encoded, err := json.Marshal(extra)
	require.NoError(t, err)
	return entity.Pool{
		Address: testPair, Exchange: "ammalgam-dlex", Type: DexType, BlockNumber: 1234,
		Tokens:   []*entity.PoolToken{{Address: testTokenX, Swappable: true}, {Address: testTokenY, Swappable: true}},
		Reserves: entity.PoolReserves{reserves[0], reserves[1]}, Extra: string(encoded),
	}
}

func newTestSimulator(t *testing.T) *PoolSimulator {
	t.Helper()
	ep := simulatorFixture(t,
		[2]string{"1000000000000000000000", "1000000000000000000000"},
		[2]string{"1000000000000000000000", "1000000000000000000000"},
		[2]string{"0", "0"})
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
	require.NoError(t, err)
	return s
}

func TestPoolSimulatorQuotesBothDirectionsAndExactOutMinimality(t *testing.T) {
	for _, direction := range [][2]string{{testTokenX, testTokenY}, {testTokenY, testTokenX}} {
		t.Run(direction[0], func(t *testing.T) {
			s := newTestSimulator(t)
			amountIn := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
			out, err := s.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: direction[0], Amount: amountIn}, TokenOut: direction[1],
			})
			require.NoError(t, err)
			require.Positive(t, out.TokenAmountOut.Amount.Sign())
			require.Positive(t, out.Fee.Amount.Sign())
			require.Equal(t, "200000000000000", out.Fee.Amount.String())
			require.Equal(t, direction[0], out.Fee.Token)
			require.Equal(t, "1000000000000000000000", s.Info.Reserves[0].String(), "quoting must not mutate state")

			exactOut, err := s.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: *out.TokenAmountOut, TokenIn: direction[0],
			})
			require.NoError(t, err)
			require.Positive(t, exactOut.TokenAmountIn.Amount.Sign())
			atMinimum, err := s.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: *exactOut.TokenAmountIn, TokenOut: direction[1],
			})
			require.NoError(t, err)
			require.GreaterOrEqual(t, atMinimum.TokenAmountOut.Amount.Cmp(out.TokenAmountOut.Amount), 0)
			previous := new(big.Int).Sub(exactOut.TokenAmountIn.Amount, big.NewInt(1))
			if previous.Sign() > 0 {
				before, err := s.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: direction[0], Amount: previous}, TokenOut: direction[1],
				})
				if err == nil {
					require.Less(t, before.TokenAmountOut.Amount.Cmp(out.TokenAmountOut.Amount), 0)
				}
			}
		})
	}
}

func TestPoolSimulatorDepletionAndStateIsolation(t *testing.T) {
	ep := simulatorFixture(t,
		[2]string{"1000000000000000000000", "1000000000000000000000"},
		[2]string{"1000000000000000000000", "1000000000000000000000"},
		[2]string{"100000000000000000000", "100000000000000000000"})
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
	require.NoError(t, err)
	clone := s.CloneState().(*PoolSimulator)
	out, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testTokenX, Amount: big.NewInt(1e18)}, TokenOut: testTokenY,
	})
	require.NoError(t, err)
	info := out.SwapInfo.(SwapInfo)
	require.Equal(t, "1001000000000000000000", info.Reserves[0].String())
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testTokenX, Amount: big.NewInt(1e18)},
		TokenAmountOut: *out.TokenAmountOut, SwapInfo: out.SwapInfo,
	})
	require.Equal(t, info.Reserves[0].String(), s.Info.Reserves[0].String())
	require.Equal(t, "1000000000000000000000", clone.Info.Reserves[0].String())
	require.NotEqual(t, s.Info.Reserves[1].String(), clone.Info.Reserves[1].String())
	info.Reserves[0].SetUint64(1)
	require.Equal(t, "1001000000000000000000", s.Info.Reserves[0].String(), "SwapInfo must not alias simulator state")
	s.extra.ReferenceReserves[0].SetUint64(7)
	require.Equal(t, "1000000000000000000000", clone.extra.ReferenceReserves[0].String())
}

func TestPoolSimulatorRejectsStaleSwapInfo(t *testing.T) {
	s := newTestSimulator(t)
	stale, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testTokenX, Amount: big.NewInt(1e18)}, TokenOut: testTokenY,
	})
	require.NoError(t, err)
	fresh, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testTokenX, Amount: big.NewInt(2e18)}, TokenOut: testTokenY,
	})
	require.NoError(t, err)

	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testTokenX, Amount: big.NewInt(2e18)},
		TokenAmountOut: *fresh.TokenAmountOut, SwapInfo: fresh.SwapInfo,
	})
	afterFresh := [2]string{s.Info.Reserves[0].String(), s.Info.Reserves[1].String()}
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testTokenX, Amount: big.NewInt(1e18)},
		TokenAmountOut: *stale.TokenAmountOut, SwapInfo: stale.SwapInfo,
	})
	require.Equal(t, afterFresh[0], s.Info.Reserves[0].String())
	require.Equal(t, afterFresh[1], s.Info.Reserves[1].String())
}

func TestPoolSimulatorExactOutUpdatesWithExecutedExactInOutput(t *testing.T) {
	ep := simulatorFixture(t,
		[2]string{"1000000000000000000", "1000000000000000000000000"},
		[2]string{"1000000000000000000", "1000000000000000000000000"},
		[2]string{"0", "0"})
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
	require.NoError(t, err)

	requestedOut := big.NewInt(1)
	quote, err := s.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testTokenY, Amount: requestedOut}, TokenIn: testTokenX,
	})
	require.NoError(t, err)
	info := quote.SwapInfo.(SwapInfo)
	require.Greater(t, info.AmountOut.ToBig().Cmp(requestedOut), 0,
		"the exact-input adapter can execute more output than the exact-out target")

	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  *quote.TokenAmountIn,
		TokenAmountOut: pool.TokenAmount{Token: testTokenY, Amount: requestedOut},
		SwapInfo:       quote.SwapInfo,
	})
	require.Equal(t, info.Reserves[0].String(), s.Info.Reserves[0].String())
	require.Equal(t, info.Reserves[1].String(), s.Info.Reserves[1].String())
}

func TestPoolSimulatorMetaAndApproval(t *testing.T) {
	s := newTestSimulator(t)
	require.Equal(t, pool.RouterMetaInfo{Router: testPair, BlockNumber: 1234}, s.GetMetaInfo(testTokenX, testTokenY))
	require.Empty(t, s.GetApprovalAddress(testTokenX, testTokenY))
	require.Equal(t, DexType, "ammalgam-dlex")
}

func TestPoolSimulatorRejectsInvalidInputsAndStates(t *testing.T) {
	s := newTestSimulator(t)
	invalidAmounts := []*big.Int{nil, big.NewInt(0), big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 256)}
	for _, amount := range invalidAmounts {
		_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: testTokenX, Amount: amount}, TokenOut: testTokenY,
		})
		require.Error(t, err)
		_, err = s.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: testTokenY, Amount: amount}, TokenIn: testTokenX,
		})
		require.Error(t, err)
	}
	for _, pair := range [][2]string{{"bad", testTokenY}, {testTokenX, "bad"}, {testTokenX, testTokenX}} {
		_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: pair[0], Amount: big.NewInt(1e18)}, TokenOut: pair[1],
		})
		require.Error(t, err)
	}
	base := simulatorFixture(t, [2]string{"100", "100"}, [2]string{"100", "100"}, [2]string{"0", "0"})
	for _, reserve := range []string{"-1", "bad", "5192296858534827628530496329220096"} {
		bad := base
		bad.Reserves = entity.PoolReserves{reserve, "100"}
		_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: bad})
		require.Error(t, err)
	}
	bad := base
	bad.Tokens = bad.Tokens[:1]
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: bad})
	require.Error(t, err)
	bad = base
	bad.Extra = `{"referenceReserves":["100","100"],"missingAssets":["100","0"]}`
	_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: bad})
	require.Error(t, err)
}

func TestPoolSimulatorReserveCapAndStrictOutputBound(t *testing.T) {
	const max = "5192296858534827628530496329220095" // 2^112-1
	ep := simulatorFixture(t, [2]string{"5192296858534827628530496329220094", max},
		[2]string{"5192296858534827628530496329220094", max}, [2]string{"0", "0"})
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
	require.NoError(t, err)
	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testTokenX, Amount: big.NewInt(1000)}, TokenOut: testTokenY,
	})
	require.ErrorIs(t, err, ErrReserveLimit)

	ep = simulatorFixture(t, [2]string{"1000000000000000000000", "1000000000000000000000"},
		[2]string{"1000000000000000000000", "1000000000000000000000"},
		[2]string{"0", "999999999999999999999"})
	s, err = NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
	require.NoError(t, err)
	_, err = s.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testTokenY, Amount: big.NewInt(1)}, TokenIn: testTokenX,
	})
	require.Error(t, err, "amountOut must be strictly less than reserveOut minus missingOut")
}

// Pool states and quotes taken at Ethereum block 26181046. Each output was replayed on a mainnet fork against
// the real pair.swap: out passes the K check, out+1 reverts, and the exact-out input is the minimum that works.
// The USDC/USDT pair was past the 1-day lending checkpoint, so its reserves include accrued LP interest.
func TestQuotesMatchOnChainSwapAtBlock26181046(t *testing.T) {
	tests := []struct {
		name                          string
		reserves, references, missing [2]string
		in, out, outIn                [2]string // per direction: X->Y, Y->X
	}{
		{
			name:       "USDC/WETH depleted-side missing assets",
			reserves:   [2]string{"84732207639", "33786949378155639618"},
			references: [2]string{"84870087911", "33666176041955854980"},
			missing:    [2]string{"47666742598", "24054088558545824156"},
			in:         [2]string{"84732207", "33786949378155639"},
			out:        [2]string{"33752858463128903", "84509242"},
			outIn:      [2]string{"84732207", "33786949069799458"},
		},
		{
			name:       "USDC/USDT accrued lending checkpoint",
			reserves:   [2]string{"30088143792", "30061657241"},
			references: [2]string{"30073607076", "30014869562"},
			missing:    [2]string{"21063181895", "22036751344"},
			in:         [2]string{"300881437", "300616572"},
			out:        [2]string{"296993505", "297127656"},
			outIn:      [2]string{"300881437", "300616572"},
		},
	}
	for _, tc := range tests {
		s, err := NewPoolSimulator(pool.FactoryParams{
			EntityPool: simulatorFixture(t, tc.reserves, tc.references, tc.missing),
		})
		require.NoError(t, err, tc.name)
		tokens := [2]string{testTokenX, testTokenY}
		for i := range 2 {
			in, _ := new(big.Int).SetString(tc.in[i], 10)
			res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokens[i], Amount: in}, TokenOut: tokens[1-i],
			})
			require.NoError(t, err, tc.name)
			require.Equal(t, tc.out[i], res.TokenAmountOut.Amount.String(), tc.name)

			out, _ := new(big.Int).SetString(tc.out[i], 10)
			resIn, err := s.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: tokens[1-i], Amount: out}, TokenIn: tokens[i],
			})
			require.NoError(t, err, tc.name)
			require.Equal(t, tc.outIn[i], resIn.TokenAmountIn.Amount.String(), tc.name)
		}
	}
}
