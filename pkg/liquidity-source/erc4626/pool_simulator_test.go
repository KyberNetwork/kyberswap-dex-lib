package erc4626

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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
			name:          "0->1 too much",
			tokenInIdx:    0,
			tokenOutIdx:   1,
			amountIn:      big.NewInt(1.3e18),
			expectedError: assert.Error,
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

// evaUSDT (0x501ebf66...) reverts deposits below minAmount() (2000 USDT); quoting them would make routes fail.
func TestMinDeposit(t *testing.T) {
	t.Parallel()
	var p entity.Pool
	assert.NoError(t, json.Unmarshal([]byte(`{"address":"0x501ebf66d76a96d4fb26ccead42957653e16b8b8","exchange":"erc4626","type":"erc4626","reserves":["0","0"],"tokens":[{"address":"0x501ebf66d76a96d4fb26ccead42957653e16b8b8","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","decimals":6,"swappable":true}],"extra":"{\"g\":{\"d\":220062},\"minD\":\"2000000000\",\"dR\":[\"1000000\",\"1000000000000\",\"1000000000000000000\",\"1000000000000000000000000\",\"1000000000000000000000000000000\"]}"}`), &p))
	sim := lo.Must(NewPoolSimulator(p))
	share, usdt := p.Tokens[0].Address, p.Tokens[1].Address

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdt, Amount: big.NewInt(1999999999)}, TokenOut: share})
	assert.ErrorIs(t, err, ErrERC4626DepositLessThanMin)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdt, Amount: big.NewInt(2000000000)}, TokenOut: share})
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(2000000000), res.TokenAmountOut.Amount)

	_, err = sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: share, Amount: big.NewInt(1000000000)}, TokenIn: usdt})
	assert.ErrorIs(t, err, ErrERC4626DepositLessThanMin)
}

// The configured method name must pack to the on-chain selector, e.g. evaUSDT's minAmount() = 0x9b2cb5d8.
func TestUint256GetterABI(t *testing.T) {
	t.Parallel()
	a := Uint256GetterABI("minAmount")
	data, err := a.Pack("minAmount")
	assert.NoError(t, err)
	assert.Equal(t, "9b2cb5d8", hex.EncodeToString(data))

	var out *big.Int
	ret := common.LeftPadBytes(big.NewInt(2e9).Bytes(), 32)
	unpacked, err := a.Methods["minAmount"].Outputs.Unpack(ret)
	assert.NoError(t, err)
	assert.NoError(t, a.Methods["minAmount"].Outputs.Copy(&out, unpacked))
	assert.Equal(t, big.NewInt(2e9), out)
}
