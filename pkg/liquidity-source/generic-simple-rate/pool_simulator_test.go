package generic_simple_rate

import (
	"math/big"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var testPool = entity.Pool{
	Address:  "0xbdcfca946b6cdd965f99a839e4435bcdc1bc470b",
	Exchange: "mkr-sky",
	Type:     "mkr-sky",
	Reserves: []string{
		"0",
		"0",
	},
	Tokens: []*entity.PoolToken{
		{
			Address:   "0x56072c95faa701256059aa122697b133aded9279",
			Swappable: true,
		},
		{
			Address:   "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
			Swappable: true,
		},
	},
	Extra: "{\"rate\":24000,\"rateUnit\":1,\"isRateInversed\":true}",
}

func TestGetAmountOut(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		entityPool        entity.Pool
		tokenAmountIn     poolPkg.TokenAmount
		tokenOut          string
		expectedAmountOut *poolPkg.TokenAmount
		expectedErr       error
	}{
		{
			name:       "test token0 as tokenIn",
			entityPool: testPool,
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x56072c95faa701256059aa122697b133aded9279",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			tokenOut: "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
				Amount: bignumber.NewBig("41666666666666"),
			},
			expectedErr: nil,
		}, {
			name:       "test token1 as tokenIn",
			entityPool: testPool,
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
				Amount: bignumber.NewBig("1000000000000000000"),
			},
			tokenOut: "0x56072c95faa701256059aa122697b133aded9279",
			expectedAmountOut: &poolPkg.TokenAmount{
				Token:  "0x56072c95faa701256059aa122697b133aded9279",
				Amount: bignumber.NewBig("24000000000000000000000"),
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewPoolSimulator(tc.entityPool)
			assert.Nil(t, err)
			calcAmountOutResult, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
				return pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         nil,
				})
			})

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedAmountOut, calcAmountOutResult.TokenAmountOut)
		})
	}
}

func TestCalcAmountIn(t *testing.T) {
	t.Parallel()
	const t0, t1 = "0x56072c95faa701256059aa122697b133aded9279", "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2"
	mk := func(extra string) entity.Pool {
		p := testPool
		p.Extra = extra
		return p
	}
	inv := mk(`{"rate":24000,"rateUnit":1,"isRateInversed":true,"isBidirectional":true}`)
	fwd := mk(`{"rate":3,"rateUnit":7,"isBidirectional":true}`)
	// rate=7 unit=3 not inversed: token0->token1 is the Div-then-Mul branch.
	fwd2 := mk(`{"rate":7,"rateUnit":3,"isBidirectional":true}`)
	huge := mk(`{"rate":"115792089237316195423570985008687907853269984665640564039457584007913129639935","rateUnit":1,"isBidirectional":true}`)

	cases := []struct {
		name    string
		pool    entity.Pool
		tokenIn string
		out     string
		wantIn  string
		wantErr error
	}{
		{"inversed t0->t1 exact", inv, t0, "41666666666666", "999999999999984000", nil},
		{"inversed t0->t1 small", inv, t0, "1", "24000", nil},
		{"inversed t1->t0", inv, t1, "24000", "1", nil},
		{"inversed t1->t0 ceil of rate", inv, t1, "24001", "2", nil},
		{"not inversed t0->t1 mul-branch", fwd, t0, "10", "28", nil},
		{"not inversed t1->t0 ceil", fwd, t1, "10", "5", nil},
		{"div-then-mul branch", fwd2, t0, "15", "9", nil}, // ceil(15/7)=3 *3
		{"zero out", inv, t0, "0", "", ErrZeroAmount},
		{"overflow", huge, t1, "115792089237316195423570985008687907853269984665640564039457584007913129639935", "", ErrOverflow},
		{"paused", mk(`{"rate":1,"rateUnit":1,"paused":true}`), t0, "1", "", ErrPoolPaused},
		{"unidirectional reverse", mk(`{"rate":2,"rateUnit":1}`), t1, "1", "", ErrUnidirectional},
		{"unidirectional forward", mk(`{"rate":2,"rateUnit":1}`), t0, "4", "2", nil},
		{"zero rate", mk(`{"rate":0,"rateUnit":1,"isBidirectional":true}`), t0, "1", "", ErrZeroAmount},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sim, err := NewPoolSimulator(tc.pool)
			assert.NoError(t, err)
			tokenOut := lo.Ternary(tc.tokenIn == t0, t1, t0)
			res, err := sim.CalcAmountIn(poolPkg.CalcAmountInParams{
				TokenAmountOut: poolPkg.TokenAmount{Token: tokenOut, Amount: bignumber.NewBig(tc.out)},
				TokenIn:        tc.tokenIn,
			})
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantIn, res.TokenAmountIn.Amount.String())

			// Round trip: amountIn gives >= out, and amountIn-1 gives < out (minimality).
			outRes, err := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{TokenAmountIn: *res.TokenAmountIn, TokenOut: tokenOut})
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, outRes.TokenAmountOut.Amount.Cmp(bignumber.NewBig(tc.out)), 0)
			less := poolPkg.TokenAmount{Token: tc.tokenIn, Amount: new(big.Int).Sub(res.TokenAmountIn.Amount, big1)}
			outRes, err = sim.CalcAmountOut(poolPkg.CalcAmountOutParams{TokenAmountIn: less, TokenOut: tokenOut})
			assert.NoError(t, err)
			assert.Less(t, outRes.TokenAmountOut.Amount.Cmp(bignumber.NewBig(tc.out)), 0)
		})
	}
}

var big1 = big.NewInt(1)

// onchain-price-service derives its sell pool types from CanCalcAmountIn.
func TestRegisteredAsExactOut(t *testing.T) {
	assert.Contains(t, poolPkg.CanCalcAmountIn, DexType)
}
