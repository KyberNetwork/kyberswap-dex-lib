package erc4626

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0xd9a442856c234a39a81a089c06451ebaa4306a72","exchange":"erc4626","type":"erc4626","timestamp":1757342856,"reserves":["0","32876264515566662491485"],"tokens":[{"address":"0xd9a442856c234a39a81a089c06451ebaa4306a72","symbol":"pufETH","decimals":18,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"g\":{\"d\":115994,\"r\":135937},\"sT\":3,\"mR\":\"1200300400200300400\",\"dR\":[\"5\",\"945745537757\",\"945745537757005980\",\"945745537757005980110202\",\"945745537757005980110202556790\"],\"rR\":[\"5\",\"1046793202268\",\"1046793202268710558\",\"1046793202268710559026277\",\"1046793202268710559026277755099\"]}","blockNumber":23319067}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
	tokens  = entityPool.Tokens
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                    string
		tokenInIdx, tokenOutIdx int
		amountIn                *big.Int
		expectedAmountOut       *big.Int
		expectedError           assert.ErrorAssertionFunc
	}{
		{
			name:              "0->1 ok",
			tokenInIdx:        0,
			tokenOutIdx:       1,
			amountIn:          big.NewInt(1.2e18),
			expectedAmountOut: big.NewInt(1256151842722452669),
			expectedError:     assert.NoError,
		},
		{
			name:              "0->1 ok",
			tokenInIdx:        0,
			tokenOutIdx:       1,
			amountIn:          big.NewInt(1.3e18),
			expectedAmountOut: big.NewInt(1360831162949323725),
			expectedError:     assert.NoError,
		},
		{
			name:              "1->0 ok",
			tokenInIdx:        1,
			tokenOutIdx:       0,
			amountIn:          big.NewInt(1.1e18),
			expectedAmountOut: big.NewInt(1040320091532706578),
			expectedError:     assert.NoError,
		},
	}

	poolSim := poolSim.CloneState()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokenAmountIn := pool.TokenAmount{
				Token:  tokens[tc.tokenInIdx].Address,
				Amount: tc.amountIn,
			}
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: tokenAmountIn,
					TokenOut:      tokens[tc.tokenOutIdx].Address,
				})
			})
			tc.expectedError(t, err)
			if err == nil {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
				poolSim.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  tokenAmountIn,
					TokenAmountOut: *result.TokenAmountOut,
					Fee:            *result.Fee,
					SwapInfo:       result.SwapInfo,
				})
			}
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, poolSim)
}
