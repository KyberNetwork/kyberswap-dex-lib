package hyeth

import (
	"math/big"
	"strings"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestAddSaturating(t *testing.T) {
	assert.Equal(t, uint256.NewInt(5), addSaturating(uint256.NewInt(2), uint256.NewInt(3)))
	assert.Equal(t, uint256.NewInt(2), addSaturating(uint256.NewInt(2), uint256.NewInt(0)))
	assert.Equal(t, number.MaxU256, addSaturating(number.MaxU256, uint256.NewInt(1)))
	assert.Equal(t, number.MaxU256, addSaturating(number.MaxU256, number.MaxU256))
	a := uint256.NewInt(7)
	addSaturating(a, uint256.NewInt(1))
	assert.Equal(t, uint256.NewInt(7), a, "the arguments are left unchanged")
}

func TestSubSaturating(t *testing.T) {
	assert.Equal(t, uint256.NewInt(1), subSaturating(uint256.NewInt(3), uint256.NewInt(2)))
	assert.True(t, subSaturating(uint256.NewInt(2), uint256.NewInt(2)).IsZero())
	assert.True(t, subSaturating(uint256.NewInt(2), uint256.NewInt(3)).IsZero())
	assert.Equal(t, number.MaxU256, subSaturating(number.MaxU256, uint256.NewInt(0)))
}

// A vault without a deposit limit is stored with the largest maxDeposit. A redeem then an issue on
// the same simulator must still quote the issue: the redeem used to wrap the limit to the redeem's
// output less 1, which refused any larger issue.
func TestRedeemKeepsAnUnlimitedDepositUnlimited(t *testing.T) {
	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(getPoolJSON), &p))
	var extra map[string]any
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	extra["maxDeposit"] = number.MaxU256.Dec()
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(b)
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	hyeth, weth := p.Tokens[0].Address, p.Tokens[1].Address
	swap := func(tokenIn, tokenOut string, amount *big.Int) {
		t.Helper()
		in := pool.TokenAmount{Token: tokenIn, Amount: amount}
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: in, TokenOut: tokenOut})
		require.NoError(t, err, "%s -> %s", tokenIn, tokenOut)
		sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in, TokenAmountOut: *res.TokenAmountOut,
			Fee: *res.Fee, SwapInfo: res.SwapInfo})
	}
	oneEth := big.NewInt(1e18)
	swap(hyeth, weth, oneEth)
	require.True(t, strings.EqualFold(hyeth, hyethToken))
	assert.Equal(t, number.MaxU256, sim.maxDeposit, "a redeem leaves an unlimited deposit unlimited")
	swap(weth, hyeth, new(big.Int).Mul(oneEth, big.NewInt(10)))
}
