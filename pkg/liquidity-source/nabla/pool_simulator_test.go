package nabla

import (
	"testing"
	"time"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func Test_sell(t *testing.T) {
	p0Meta := NablaPoolMeta{
		CurveBeta:   int256.NewInt(5000000000000000),
		CurveC:      int256.MustFromDec("17075887234393789126"),
		BackstopFee: int256.NewInt(300),
		ProtocolFee: int256.NewInt(100),
		LpFee:       int256.NewInt(200),
	}

	p1Meta := NablaPoolMeta{
		CurveBeta:   int256.NewInt(5000000000000000),
		CurveC:      int256.MustFromDec("17075887234393789126"),
		BackstopFee: int256.NewInt(300),
		ProtocolFee: int256.NewInt(100),
		LpFee:       int256.NewInt(200),
	}

	p0StateBalanced := NablaPoolState{
		Reserve:             int256.MustFromDec("1000000000000000000000"),
		ReserveWithSlippage: int256.MustFromDec("1000000000000000000000"),
		TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
		Price:               int256.MustFromDec("100000000"),
	}

	p1StateBalanced := NablaPoolState{
		Reserve:             int256.MustFromDec("1000000000000000000000"),
		ReserveWithSlippage: int256.MustFromDec("1000000000000000000000"),
		TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
		Price:               int256.MustFromDec("100000000"),
	}

	p0StateImbalanced := NablaPoolState{
		Reserve:             int256.MustFromDec("1099997249253573525485"),
		ReserveWithSlippage: int256.MustFromDec("1100000000000000000000"),
		TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
		Price:               int256.MustFromDec("100000000"),
	}

	p1StateImbalanced := NablaPoolState{
		Reserve:             int256.MustFromDec("900052749371053261277"),
		ReserveWithSlippage: int256.MustFromDec("900055528992381704580"),
		TotalLiabilities:    int256.MustFromDec("1000019999449850714705"),
		Price:               int256.MustFromDec("100000000"),
	}

	tests := []struct {
		name     string
		p0State  NablaPoolState
		p1State  NablaPoolState
		fromIdx  int
		toIdx    int
		amountIn *int256.Int
		want     *int256.Int
	}{
		{
			name:     "swap with balanced base pools - p0 to p1",
			p0State:  p0StateBalanced,
			p1State:  p1StateBalanced,
			fromIdx:  0,
			toIdx:    1,
			amountIn: int256.MustFromDec("1000000000000000000"),
			want:     int256.MustFromDec("999399447164390453"),
		},
		{
			name:     "swap with balanced base pools - p1 to p0",
			p0State:  p0StateBalanced,
			p1State:  p1StateBalanced,
			fromIdx:  1,
			toIdx:    0,
			amountIn: int256.MustFromDec("1000000000000000000"),
			want:     int256.MustFromDec("999399447164390453"),
		},
		{
			name:     "swap with imbalanced pools - p0 to p1",
			p0State:  p0StateImbalanced,
			p1State:  p1StateImbalanced,
			fromIdx:  0,
			toIdx:    1,
			amountIn: int256.MustFromDec("1000000000000000000"),
			want:     int256.MustFromDec("999288878655122808"),
		},
		{
			name:     "swap with imbalanced pools - p1 to p0",
			p0State:  p0StateImbalanced,
			p1State:  p1StateImbalanced,
			fromIdx:  1,
			toIdx:    0,
			amountIn: int256.MustFromDec("100000000000000000000"),
			want:     int256.MustFromDec("99945528699304486120"),
		},
		{
			name: "swap with imbalanced pools - p0 to p1 with price update (case 1)",
			p0State: NablaPoolState{
				Reserve:             int256.MustFromDec("1099997249253573525485"),
				ReserveWithSlippage: int256.MustFromDec("1100000000000000000000"),
				TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
				Price:               int256.MustFromDec("200000000"),
			},
			p1State:  p1StateImbalanced,
			fromIdx:  0,
			toIdx:    1,
			amountIn: int256.MustFromDec("1000000000000000000"),
			want:     int256.MustFromDec("1998577195258299942"),
		},
		{
			name: "swap with imbalanced pools - p0 to p1 with price update (case 2)",
			p0State: NablaPoolState{
				Reserve:             int256.MustFromDec("1099997249253573525485"),
				ReserveWithSlippage: int256.MustFromDec("1100000000000000000000"),
				TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
				Price:               int256.MustFromDec("200000000"),
			},
			p1State:  p1StateImbalanced,
			fromIdx:  0,
			toIdx:    1,
			amountIn: int256.MustFromDec("294110300000000000000"),
			want:     int256.MustFromDec("587655876076133933446"),
		},
		{
			name: "swap with imbalanced pools - p1 to p0 with price update (case 1)",
			p0State: NablaPoolState{
				Reserve:             int256.MustFromDec("1099997249253573525485"),
				ReserveWithSlippage: int256.MustFromDec("1100000000000000000000"),
				TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
				Price:               int256.MustFromDec("200000000"),
			},
			p1State:  p1StateImbalanced,
			fromIdx:  1,
			toIdx:    0,
			amountIn: int256.MustFromDec("100000000000000000000"),
			want:     int256.MustFromDec("49973449671535241267"),
		},
		{
			name: "swap with imbalanced pools - p1 to p0 with price update (case 2)",
			p0State: NablaPoolState{
				Reserve:             int256.MustFromDec("1099997249253573525485"),
				ReserveWithSlippage: int256.MustFromDec("1100000000000000000000"),
				TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
				Price:               int256.MustFromDec("200000000"),
			},
			p1State:  p1StateImbalanced,
			fromIdx:  1,
			toIdx:    0,
			amountIn: int256.MustFromDec("73248000000000000000"),
			want:     int256.MustFromDec("36604959022922925425"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p0 := NablaPool{
				Meta:  p0Meta,
				State: tt.p0State,
			}
			p1 := NablaPool{
				Meta:  p1Meta,
				State: tt.p1State,
			}

			var fr, to NablaPool
			if tt.fromIdx == 0 {
				fr = p0
				to = p1
			} else {
				fr = p1
				to = p0
			}

			got, _, err := sell(fr, to, tt.amountIn, 18, 18)
			require.NoError(t, err)
			require.Equal(t, tt.want.Dec(), got.Dec())
		})
	}
}

type chainSnapshot struct {
	name       string
	frDecimals uint8
	toDecimals uint8
	frPool     NablaPool
	toPool     NablaPool
	cases      []chainSnapshotCase
}

type chainSnapshotCase struct {
	name      string
	amountIn  string
	want      string
	expectErr bool
}

// baseSnapshot — Base USDC -> EURC
func baseSnapshot() chainSnapshot {
	meta := NablaPoolMeta{
		CurveBeta:                 int256.MustFromDec("1000000000000000"),
		CurveC:                    int256.MustFromDec("33638584039112749"),
		LpFee:                     int256.NewInt(25),
		BackstopFee:               int256.NewInt(12),
		ProtocolFee:               int256.NewInt(12),
		MaxCoverageRatioForSwapIn: int256.NewInt(200),
	}
	usdc := NablaPool{
		Meta: meta,
		State: NablaPoolState{
			Reserve:             int256.MustFromDec("6668027803"),
			ReserveWithSlippage: int256.MustFromDec("6668068923"),
			TotalLiabilities:    int256.MustFromDec("6136347818"),
			Price:               int256.MustFromDec("1000000000000000000"),
		},
	}
	eurc := NablaPool{
		Meta: meta,
		State: NablaPoolState{
			Reserve:             int256.MustFromDec("4818674874"),
			ReserveWithSlippage: int256.MustFromDec("4818713498"),
			TotalLiabilities:    int256.MustFromDec("5257931086"),
			Price:               int256.MustFromDec("1176482790000000000"),
		},
	}
	return chainSnapshot{
		name:       "base",
		frDecimals: 6,
		toDecimals: 6,
		frPool:     usdc,
		toPool:     eurc,
		cases: []chainSnapshotCase{
			{"1 USDC -> EURC", "1000000", "849667", false},
			{"100 USDC -> EURC", "100000000", "84963938", false},
			{"5600 USDC -> EURC (snapshot OK)", "5600000000", "4643936953", false},
			{"5610 USDC -> EURC (snapshot reverts)", "5610000000", "", true},
		},
	}
}

// monadSnapshot — Monad WMON -> USDC
func monadSnapshot() chainSnapshot {
	wmon := NablaPool{
		Meta: NablaPoolMeta{
			CurveBeta:                 int256.MustFromDec("10000000000000000"),
			CurveC:                    int256.MustFromDec("16110498756211208902"),
			LpFee:                     int256.NewInt(200),
			BackstopFee:               int256.NewInt(1500),
			ProtocolFee:               int256.NewInt(300),
			MaxCoverageRatioForSwapIn: int256.NewInt(200),
		},
		State: NablaPoolState{
			Reserve:             int256.MustFromDec("725325240298071518380205"),
			ReserveWithSlippage: int256.MustFromDec("725338516452075133857542"),
			TotalLiabilities:    int256.MustFromDec("607213562903825489532767"),
			Price:               int256.MustFromDec("32538620000000000"),
		},
	}
	usdc := NablaPool{
		Meta: NablaPoolMeta{
			CurveBeta:                 int256.MustFromDec("5000000000000000"),
			CurveC:                    int256.MustFromDec("17075887234393789126"),
			LpFee:                     int256.NewInt(200),
			BackstopFee:               int256.NewInt(1500),
			ProtocolFee:               int256.NewInt(300),
			MaxCoverageRatioForSwapIn: int256.NewInt(200),
		},
		State: NablaPoolState{
			Reserve:             int256.MustFromDec("12691554632"),
			ReserveWithSlippage: int256.MustFromDec("12691697065"),
			TotalLiabilities:    int256.MustFromDec("15502707444"),
			Price:               int256.MustFromDec("999789830000000000"),
		},
	}
	return chainSnapshot{
		name:       "monad",
		frDecimals: 18,
		toDecimals: 6,
		frPool:     wmon,
		toPool:     usdc,
		cases: []chainSnapshotCase{
			{"0.1 WMON -> USDC", "100000000000000000", "3249", false},
			{"1 WMON -> USDC", "1000000000000000000", "32472", false},
		},
	}
}

// hyperEVMSnapshot — HyperEVM WHYPE -> USDT0 at block 32997812
func hyperEVMSnapshot() chainSnapshot {
	whype := NablaPool{
		Meta: NablaPoolMeta{
			CurveBeta:                 int256.MustFromDec("10000000000000000"),
			CurveC:                    int256.MustFromDec("16110498756211208902"),
			LpFee:                     int256.NewInt(200),
			BackstopFee:               int256.NewInt(1200),
			ProtocolFee:               int256.NewInt(600),
			MaxCoverageRatioForSwapIn: int256.NewInt(200),
		},
		State: NablaPoolState{
			Reserve:             int256.MustFromDec("1976767052165232922392"),
			ReserveWithSlippage: int256.MustFromDec("1976834228813870806396"),
			TotalLiabilities:    int256.MustFromDec("2510611626662252234913"),
			Price:               int256.MustFromDec("41439000000000000000"),
		},
	}
	usdt0 := NablaPool{
		Meta: NablaPoolMeta{
			CurveBeta:                 int256.MustFromDec("5000000000000000"),
			CurveC:                    int256.MustFromDec("17075887234393789126"),
			LpFee:                     int256.NewInt(200),
			BackstopFee:               int256.NewInt(1200),
			ProtocolFee:               int256.NewInt(600),
			MaxCoverageRatioForSwapIn: int256.NewInt(200),
		},
		State: NablaPoolState{
			Reserve:             int256.MustFromDec("18507072090"),
			ReserveWithSlippage: int256.MustFromDec("18507556268"),
			TotalLiabilities:    int256.MustFromDec("25085212745"),
			Price:               int256.MustFromDec("1000400000000000000"),
		},
	}
	return chainSnapshot{
		name:       "hyperEVM",
		frDecimals: 18,
		toDecimals: 6,
		frPool:     whype,
		toPool:     usdt0,
		cases: []chainSnapshotCase{
			{"0.1 WHYPE -> USDT0", "100000000000000000", "4134395", false},
			{"1 WHYPE -> USDT0", "1000000000000000000", "41343897", false},
			{"10 WHYPE -> USDT0", "10000000000000000000", "413436283", false},
		},
	}
}

// berachainSnapshot — Berachain WETH -> USDC.e at block 19848523
func berachainSnapshot() chainSnapshot {
	meta := NablaPoolMeta{
		CurveBeta:                 int256.MustFromDec("5000000000000000"),
		CurveC:                    int256.MustFromDec("17075887234393789126"),
		LpFee:                     int256.NewInt(200),
		BackstopFee:               int256.NewInt(1400),
		ProtocolFee:               int256.NewInt(400),
		MaxCoverageRatioForSwapIn: int256.NewInt(200),
	}
	weth := NablaPool{
		Meta: meta,
		State: NablaPoolState{
			Reserve:             int256.MustFromDec("6975460637948466212"),
			ReserveWithSlippage: int256.MustFromDec("6976286631402226543"),
			TotalLiabilities:    int256.MustFromDec("13162831213446480519"),
			Price:               int256.MustFromDec("2334964999990000000000"),
		},
	}
	usdce := NablaPool{
		Meta: meta,
		State: NablaPoolState{
			Reserve:             int256.MustFromDec("735024087"),
			ReserveWithSlippage: int256.MustFromDec("737875754"),
			TotalLiabilities:    int256.MustFromDec("11198185779"),
			Price:               int256.MustFromDec("999849120000000000"),
		},
	}
	return chainSnapshot{
		name:       "berachain",
		frDecimals: 18,
		toDecimals: 6,
		frPool:     weth,
		toPool:     usdce,
		cases: []chainSnapshotCase{
			{"0.01 WETH -> USDC.e", "10000000000000000", "23299701", false},
			{"0.05 WETH -> USDC.e", "50000000000000000", "116498073", false},
			{"0.1 WETH -> USDC.e", "100000000000000000", "232995090", false},
			{"0.2 WETH -> USDC.e", "200000000000000000", "465985959", false},
		},
	}
}

func Test_sell_chainSnapshots(t *testing.T) {
	snapshots := []chainSnapshot{
		baseSnapshot(),
		monadSnapshot(),
		hyperEVMSnapshot(),
		berachainSnapshot(),
	}

	for _, s := range snapshots {
		t.Run(s.name, func(t *testing.T) {
			for _, c := range s.cases {
				t.Run(c.name, func(t *testing.T) {
					got, _, err := sell(s.frPool, s.toPool, int256.MustFromDec(c.amountIn), s.frDecimals, s.toDecimals)
					if c.expectErr {
						require.Error(t, err)
						return
					}
					require.NoError(t, err)
					require.Equal(t, c.want, got.Dec())
				})
			}
		})
	}
}

func Test_sell_baseSnapshotSwapInfo(t *testing.T) {
	s := baseSnapshot()
	meta := s.frPool.Meta
	usdcState := s.frPool.State
	eurcState := s.toPool.State
	amountIn := int256.MustFromDec("1000000000")

	amountOut, swapInfo, err := sell(s.frPool, s.toPool, amountIn, s.frDecimals, s.toDecimals)
	require.NoError(t, err)

	curveIn := NewCurve(meta.CurveBeta, meta.CurveC)
	curveOut := NewCurve(meta.CurveBeta, meta.CurveC)

	effectiveAmountIn := curveIn.InverseHorizontal(
		usdcState.Reserve,
		usdcState.TotalLiabilities,
		new(int256.Int).Add(usdcState.ReserveWithSlippage, amountIn),
		int64(s.frDecimals),
	)
	require.Equal(t,
		new(int256.Int).Add(usdcState.Reserve, effectiveAmountIn).Dec(),
		swapInfo.frPoolNewState.Reserve.Dec(),
	)
	require.Equal(t,
		new(int256.Int).Add(usdcState.ReserveWithSlippage, amountIn).Dec(),
		swapInfo.frPoolNewState.ReserveWithSlippage.Dec(),
	)

	rawAmountOut := new(int256.Int).Mul(effectiveAmountIn, usdcState.Price)
	rawAmountOut.Quo(rawAmountOut, eurcState.Price)

	backstopFeeAmount := new(int256.Int).Mul(rawAmountOut, meta.BackstopFee)
	backstopFeeAmount.Quo(backstopFeeAmount, feePrecision)

	protocolFeeAmount := new(int256.Int).Mul(rawAmountOut, meta.ProtocolFee)
	protocolFeeAmount.Quo(protocolFeeAmount, feePrecision)

	maxLpFeeAmount := new(int256.Int).Mul(rawAmountOut, meta.LpFee)
	maxLpFeeAmount.Quo(maxLpFeeAmount, feePrecision)

	reducedReserveOut := new(int256.Int).Sub(eurcState.Reserve, rawAmountOut)
	reducedReserveOut.Add(reducedReserveOut, backstopFeeAmount)
	reducedReserveOut.Add(reducedReserveOut, protocolFeeAmount)

	actualLpFeeAmount := curveOut.InverseDiagonal(
		reducedReserveOut,
		eurcState.TotalLiabilities,
		eurcState.ReserveWithSlippage,
		int64(s.toDecimals),
	)
	if actualLpFeeAmount.Gt(maxLpFeeAmount) {
		actualLpFeeAmount = maxLpFeeAmount
	}

	actualReducedReserveOut := new(int256.Int).Add(reducedReserveOut, actualLpFeeAmount)
	actualTotalLiabilitiesOut := new(int256.Int).Add(eurcState.TotalLiabilities, actualLpFeeAmount)
	reserveWithSlippageAfterAmountOut := curveOut.Psi(actualReducedReserveOut, actualTotalLiabilitiesOut, int64(s.toDecimals))
	if reserveWithSlippageAfterAmountOut.Gt(eurcState.ReserveWithSlippage) {
		reserveWithSlippageAfterAmountOut = eurcState.ReserveWithSlippage
	}

	expectedOutputReserve := new(int256.Int).Sub(actualReducedReserveOut, protocolFeeAmount)
	expectedOutputReserveWithSlippage := curveOut.Psi(expectedOutputReserve, actualTotalLiabilitiesOut, int64(s.toDecimals))
	if expectedOutputReserveWithSlippage.Gt(reserveWithSlippageAfterAmountOut) {
		expectedOutputReserveWithSlippage = reserveWithSlippageAfterAmountOut
	}

	require.Equal(t,
		new(int256.Int).Sub(eurcState.ReserveWithSlippage, reserveWithSlippageAfterAmountOut).Dec(),
		amountOut.Dec(),
	)
	require.Equal(t, expectedOutputReserve.Dec(), swapInfo.toPoolNewState.Reserve.Dec())
	require.Equal(t, expectedOutputReserveWithSlippage.Dec(), swapInfo.toPoolNewState.ReserveWithSlippage.Dec())
	require.Equal(t, actualTotalLiabilitiesOut.Dec(), swapInfo.toPoolNewState.TotalLiabilities.Dec())
}

// Route finding (StaleCheck) must drop pools whose prices stopped refreshing: the on-chain oracle reverts
// once its own max age passes, but our last-read price would keep quoting. Indexing must still load them.
func TestNewPoolSimulator_StaleCheck(t *testing.T) {
	newEntity := func(priceTs int64) entity.Pool {
		extra, _ := json.Marshal(Extra{PriceTimestamp: priceTs})
		return entity.Pool{Exchange: DexType, Type: DexType, Extra: string(extra)}
	}
	staleCheck := pool.FactoryOpts{StaleCheck: true}
	now := time.Now().Unix()

	_, err := pool.Factory(DexType)(pool.FactoryParams{EntityPool: newEntity(now - 5), Opts: staleCheck})
	require.NoError(t, err)

	_, err = pool.Factory(DexType)(pool.FactoryParams{EntityPool: newEntity(now - int64(MaxAge.Seconds()) - 5), Opts: staleCheck})
	require.ErrorIs(t, err, ErrStalePrice)

	// Pools written before priceTs existed count as stale until the tracker reads prices again.
	_, err = pool.Factory(DexType)(pool.FactoryParams{EntityPool: newEntity(0), Opts: staleCheck})
	require.ErrorIs(t, err, ErrStalePrice)

	_, err = pool.Factory(DexType)(pool.FactoryParams{EntityPool: newEntity(0)})
	require.NoError(t, err)
}

// Surge fees (nabla-integration-reference ts/swapLogic.test.ts): the in-fee cuts amountIn before the input
// curve, the out-fee cuts the final output; both floor-divide by 1e6 and only apply on their own leg.
func Test_sell_surgeFees(t *testing.T) {
	const dynInFee, dynOutFee = 2_000, 1_000
	balanced := func(inFee, outFee uint64) NablaPool {
		return NablaPool{
			Meta: NablaPoolMeta{
				CurveBeta:   int256.NewInt(5000000000000000),
				CurveC:      int256.MustFromDec("17075887234393789126"),
				BackstopFee: int256.NewInt(300),
				ProtocolFee: int256.NewInt(100),
				LpFee:       int256.NewInt(200),
				SwapInFee:   inFee,
				SwapOutFee:  outFee,
			},
			State: NablaPoolState{
				Reserve:             int256.MustFromDec("1000000000000000000000"),
				ReserveWithSlippage: int256.MustFromDec("1000000000000000000000"),
				TotalLiabilities:    int256.MustFromDec("1000000000000000000000"),
				Price:               int256.NewInt(1e8),
			},
		}
	}
	quote := func(fr, to NablaPool, amountIn *int256.Int) (*int256.Int, *SwapInfo) {
		out, info, err := sell(fr, to, amountIn, 18, 18)
		require.NoError(t, err)
		return out, info
	}
	floorFee := func(x *int256.Int, fee int64) *int256.Int {
		f := new(int256.Int).Mul(x, int256.NewInt(fee))
		return f.Quo(f, feePrecision)
	}
	oneEth := int256.MustFromDec("1000000000000000000")
	baseline, _ := quote(balanced(0, 0), balanced(0, 0), oneEth)
	require.Equal(t, "999399447164390453", baseline.Dec())

	t.Run("in and out fees compose in contract order", func(t *testing.T) {
		reducedIn := new(int256.Int).Sub(oneEth, floorFee(oneEth, dynInFee))
		curveOut, _ := quote(balanced(0, 0), balanced(0, 0), reducedIn)
		want := new(int256.Int).Sub(curveOut, floorFee(curveOut, dynOutFee))

		got, info := quote(balanced(dynInFee, 0), balanced(0, dynOutFee), oneEth)
		require.Equal(t, want.Dec(), got.Dec())
		require.True(t, got.Lt(baseline))

		// The input pool is credited only the reduced amount; the out-fee stays in the out pool untouched.
		_, noOutFeeInfo := quote(balanced(dynInFee, 0), balanced(0, 0), oneEth)
		require.Equal(t, "1000998000000000000000", info.frPoolNewState.ReserveWithSlippage.Dec())
		require.Equal(t, noOutFeeInfo.toPoolNewState, info.toPoolNewState)
	})

	t.Run("fees floor-divide", func(t *testing.T) {
		got, _ := quote(balanced(dynInFee, 0), balanced(0, 0), int256.NewInt(3)) // 3*2000/1e6 = 0
		want, _ := quote(balanced(0, 0), balanced(0, 0), int256.NewInt(3))
		require.Equal(t, want.Dec(), got.Dec())

		got, _ = quote(balanced(0, 0), balanced(0, 333333), oneEth)
		require.Equal(t, new(int256.Int).Sub(baseline, floorFee(baseline, 333333)).Dec(), got.Dec())
	})

	t.Run("max uint16 fees", func(t *testing.T) {
		reducedIn := new(int256.Int).Sub(oneEth, floorFee(oneEth, 65535))
		curveOut, _ := quote(balanced(0, 0), balanced(0, 0), reducedIn)
		got, _ := quote(balanced(65535, 0), balanced(0, 65535), oneEth)
		require.Equal(t, new(int256.Int).Sub(curveOut, floorFee(curveOut, 65535)).Dec(), got.Dec())
	})

	t.Run("wrong-leg fees are ignored", func(t *testing.T) {
		got, _ := quote(balanced(0, dynOutFee), balanced(dynInFee, 0), oneEth)
		require.Equal(t, baseline.Dec(), got.Dec())
	})

	t.Run("caller amountIn is not mutated", func(t *testing.T) {
		amountIn := new(int256.Int).Set(oneEth)
		quote(balanced(dynInFee, 0), balanced(0, 0), amountIn)
		require.Equal(t, oneEth.Dec(), amountIn.Dec())
	})
}
