package gsm4626

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0x535b2f7c20b9c83d70e519cf9991578ef9816b7b","exchange":"gsm-4626","type":"gsm-4626","tokens":[{"address":"0x40d16fc0246ad3160ccc09b8d0d3a2cd28ae6c2f","symbol":"GHO","decimals":18,"swappable":true},{"address":"0x7bc3485026ac48b6cf9baf0a377477fff5703af8","symbol":"waEthUSDT","decimals":6,"swappable":true}],"extra":"{\"canSwap\":true,\"buyFee\":\"15\",\"sellFee\":\"0\",\"currentExposure\":\"318074276664\",\"exposureCap\":\"25000000000000\",\"rate\":\"1146698616999179571600457092\",\"ghoLimit\":\"200000000000000000000000\",\"ghoUsed\":\"0\"}","staticExtra":"{\"priceRatio\":\"1000000000000000000\"}","blockNumber":23791585}`), &entityPool)
	poolSim    = lo.Must(NewPoolSimulator(entityPool))
	tokens     = entityPool.Tokens
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                    string
		tokenInIdx, tokenOutIdx int
		rate                    *big.Int
		amountIn                *big.Int
		expectedAmountOut       *big.Int
		expectedError           assert.ErrorAssertionFunc
	}{
		{
			name:              "buy asset",
			tokenInIdx:        0,
			tokenOutIdx:       1,
			rate:              bignumber.NewBig("1146696576337460102970261542"),
			amountIn:          big.NewInt(1000000000000000000),
			expectedAmountOut: big.NewInt(870763),
			expectedError:     assert.NoError,
		},
		{
			name:              "sell asset",
			tokenInIdx:        1,
			tokenOutIdx:       0,
			amountIn:          big.NewInt(100000000000),
			rate:              bignumber.NewBig("1146698616999179571600457092"),
			expectedAmountOut: bignumber.NewBig("114669861699000000000000"),
			expectedError:     assert.NoError,
		},
		{
			name:          "sell asset exceeds gho facilitator limit",
			tokenInIdx:    1,
			tokenOutIdx:   0,
			amountIn:      bignumber.NewBig("200000000000"),
			rate:          bignumber.NewBig("1146698616999179571600457092"),
			expectedError: func(t assert.TestingT, err error, _ ...any) bool { return assert.ErrorIs(t, err, ErrGhoLimitExceeded) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.rate != nil {
				poolSim.Rate.SetFromBig(tc.rate)
			}

			cloned := poolSim.CloneState()
			tokenAmountIn := pool.TokenAmount{
				Token:  tokens[tc.tokenInIdx].Address,
				Amount: tc.amountIn,
			}
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return cloned.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: tokenAmountIn,
					TokenOut:      tokens[tc.tokenOutIdx].Address,
				})
			})
			tc.expectedError(t, err)
			if err == nil {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
				cloned.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  tokenAmountIn,
					TokenAmountOut: *result.TokenAmountOut,
					Fee:            *result.Fee,
					SwapInfo:       result.SwapInfo,
				})
			}
		})
	}
}

// TestGhoUsedRoundTrip guards against underflowing GhoUsed on the buy-asset path when
// GhoUsed starts at zero (the common case: paying GHO in doesn't need to have been
// preceded by a sell within the same simulator instance).
func TestGhoUsedRoundTrip(t *testing.T) {
	t.Parallel()

	cloned := poolSim.CloneState().(*PoolSimulator)
	cloned.Rate.SetFromBig(bignumber.NewBig("1146696576337460102970261542"))

	buyResult, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokens[0].Address, Amount: big.NewInt(1000000000000000000)},
		TokenOut:      tokens[1].Address,
	})
	assert.NoError(t, err)
	cloned.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokens[0].Address, Amount: big.NewInt(1000000000000000000)},
		TokenAmountOut: *buyResult.TokenAmountOut,
		Fee:            *buyResult.Fee,
		SwapInfo:       buyResult.SwapInfo,
	})
	assert.True(t, cloned.GhoUsed.IsZero())

	sellResult, err := cloned.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokens[1].Address, Amount: big.NewInt(100000000000)},
		TokenOut:      tokens[0].Address,
	})
	assert.NoError(t, err)
	assert.True(t, sellResult.TokenAmountOut.Amount.Sign() > 0)
}

// TestNewPoolSimulatorMissingGhoUsage guards against a nil-pointer panic on pools whose
// stale Extra (persisted before ghoLimit/ghoUsed were tracked) omits these fields.
func TestNewPoolSimulatorMissingGhoUsage(t *testing.T) {
	t.Parallel()

	p := entityPool
	p.Extra = `{"canSwap":true,"buyFee":"15","sellFee":"0","currentExposure":"318074276664","exposureCap":"25000000000000","rate":"1146698616999179571600457092"}`

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	assert.NotNil(t, sim.GhoLimit)
	assert.NotNil(t, sim.GhoUsed)

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokens[1].Address, Amount: big.NewInt(100000000000)},
		TokenOut:      tokens[0].Address,
	})
	assert.ErrorIs(t, err, ErrGhoLimitExceeded)
}
