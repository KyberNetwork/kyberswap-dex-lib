package makerpsm

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var USDX_WAD = bignumber.TenPowInt(6)
var TOLL_ONE_PCT = bignumber.TenPowInt(16)

func newPool(t *testing.T, eth *big.Int, tIn *big.Int, tOut *big.Int) *PoolSimulator {
	eth = new(big.Int).Mul(eth, bignumber.BONE)
	p, err := NewPoolSimulator(entity.Pool{
		Tokens: []*entity.PoolToken{{Address: "USDX", Decimals: 6}, {Address: DAIAddress}},
		Extra:  fmt.Sprintf("{\"psm\":{\"tIn\":%v,\"tOut\":%v,\"vat\":{\"ilk\":{\"art\":0,\"rate\":1,\"line\":%v},\"debt\":0,\"line\":%v}}}", tIn, tOut, eth, eth),
	})
	require.Nil(t, err)
	assert.Equal(t, []string{DAIAddress}, p.CanSwapTo("USDX"))
	assert.Equal(t, []string{"USDX"}, p.CanSwapTo(DAIAddress))
	return p
}

func TestGetAmountOut_sellGemNoFee(t *testing.T) {
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L166
	pool100 := newPool(t, big.NewInt(100), big.NewInt(0), big.NewInt(0))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), USDX_WAD)}
	out, err := pool100.CalcAmountOut(inAmount, DAIAddress)
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(100), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)

	// reach dept ceiling
	pool100.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: inAmount, TokenAmountOut: *out.TokenAmountOut, Fee: *out.Fee})
	_, err = pool100.CalcAmountOut(inAmount, DAIAddress)
	require.NotNil(t, err)
	fmt.Println(err)
}

func TestGetAmountOut_sellGemWithFee(t *testing.T) {
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L189
	pool100 := newPool(t, big.NewInt(100), TOLL_ONE_PCT, big.NewInt(0))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), USDX_WAD)}
	out, err := pool100.CalcAmountOut(inAmount, DAIAddress)
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(99), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)
}

func TestGetAmountOut_swapBothNoFee(t *testing.T) {
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L208
	// sell 100 USDX
	pool100 := newPool(t, big.NewInt(100), big.NewInt(0), big.NewInt(0))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), USDX_WAD)}
	out, err := pool100.CalcAmountOut(inAmount, DAIAddress)
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(100), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)

	pool100.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: inAmount, TokenAmountOut: *out.TokenAmountOut, Fee: *out.Fee})

	// then buy back with 40 eth
	inAmount = pool.TokenAmount{Token: DAIAddress, Amount: new(big.Int).Mul(big.NewInt(40), bignumber.BONE)}
	out, err = pool100.CalcAmountOut(inAmount, "USDX")
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(40), USDX_WAD), out.TokenAmountOut.Amount)
	assert.Equal(t, "USDX", out.TokenAmountOut.Token)
}

func TestGetAmountOut_swapBothWithFee(t *testing.T) {
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L224
	// sell 100 USDX -> 95 eth
	pool100 := newPool(t, big.NewInt(100), new(big.Int).Mul(big.NewInt(5), TOLL_ONE_PCT), new(big.Int).Mul(big.NewInt(10), TOLL_ONE_PCT))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), USDX_WAD)}
	out, err := pool100.CalcAmountOut(inAmount, DAIAddress)
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(95), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)

	pool100.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: inAmount, TokenAmountOut: *out.TokenAmountOut, Fee: *out.Fee})

	// then buy back with 44 eth -> 40 usdx
	inAmount = pool.TokenAmount{Token: DAIAddress, Amount: new(big.Int).Mul(big.NewInt(44), bignumber.BONE)}
	out, err = pool100.CalcAmountOut(inAmount, "USDX")
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(40), USDX_WAD), out.TokenAmountOut.Amount)
	assert.Equal(t, "USDX", out.TokenAmountOut.Token)
}
