package makerpsm

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"testing/synctest"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test/ethrpcmock"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var USDX_WAD = bignumber.TenPowInt(6)
var TOLL_ONE_PCT = bignumber.TenPowInt(16)

func TestGetNewPoolStatePinsVatRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		srv, recorder := ethrpcmock.New(t, makerPSMPSM, makerPSMVat)
		client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0x4444444444444444444444444444444444444444"))
		tracker := NewPoolTracker(&Config{DexID: DexTypeMakerPSM}, client)
		p := entity.Pool{Address: "0x7777777777777777777777777777777777777777"}
		overrides := map[common.Address]gethclient.OverrideAccount{common.HexToAddress("0x8888888888888888888888888888888888888888"): {}}
		got, err := tracker.GetNewPoolStateWithOverrides(context.Background(), p, pool.GetNewPoolStateWithOverridesParams{Overrides: overrides})
		require.NoError(t, err)
		require.Equal(t, uint64(ethrpcmock.Block), got.BlockNumber)
		require.Len(t, got.Reserves, 2)
		ethrpcmock.CheckPool(t, "makerpsm_TestGetNewPoolStatePinsVatRead", got)
		ethrpcmock.CheckRequests(t, "makerpsm_TestGetNewPoolStatePinsVatRead", recorder)
		ethrpcmock.RequireRequestsPinned(t, recorder, 2)
	})
}

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
	t.Parallel()
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L166
	pool100 := newPool(t, big.NewInt(100), big.NewInt(0), big.NewInt(0))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), USDX_WAD)}
	out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      DAIAddress,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(100), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)

	// reach dept ceiling
	pool100.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: inAmount, TokenAmountOut: *out.TokenAmountOut, Fee: *out.Fee})
	_, err = testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      DAIAddress,
		})
	})
	require.NotNil(t, err)
	fmt.Println(err)
}

func TestGetAmountOut_sellGemWithFee(t *testing.T) {
	t.Parallel()
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L189
	pool100 := newPool(t, big.NewInt(100), TOLL_ONE_PCT, big.NewInt(0))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), USDX_WAD)}
	out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      DAIAddress,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(99), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)
}

func TestGetAmountOut_swapBothNoFee(t *testing.T) {
	t.Parallel()
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L208
	// sell 100 USDX
	pool100 := newPool(t, big.NewInt(100), big.NewInt(0), big.NewInt(0))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), USDX_WAD)}
	out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      DAIAddress,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(100), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)

	pool100.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: inAmount, TokenAmountOut: *out.TokenAmountOut, Fee: *out.Fee})

	// then buy back with 40 eth
	inAmount = pool.TokenAmount{Token: DAIAddress, Amount: new(big.Int).Mul(big.NewInt(40), bignumber.BONE)}
	out, err = testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      "USDX",
			Limit:         nil,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(40), USDX_WAD), out.TokenAmountOut.Amount)
	assert.Equal(t, "USDX", out.TokenAmountOut.Token)
}

func TestGetAmountOut_swapBothWithFee(t *testing.T) {
	t.Parallel()
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L224
	// sell 100 USDX -> 95 eth
	pool100 := newPool(t, big.NewInt(100), new(big.Int).Mul(big.NewInt(5), TOLL_ONE_PCT), new(big.Int).Mul(big.NewInt(10), TOLL_ONE_PCT))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), USDX_WAD)}
	out, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      DAIAddress,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(95), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)

	pool100.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: inAmount, TokenAmountOut: *out.TokenAmountOut, Fee: *out.Fee})

	// then buy back with 44 eth -> 40 usdx
	inAmount = pool.TokenAmount{Token: DAIAddress, Amount: new(big.Int).Mul(big.NewInt(44), bignumber.BONE)}
	out, err = testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      "USDX",
			Limit:         nil,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(40), USDX_WAD), out.TokenAmountOut.Amount)
	assert.Equal(t, "USDX", out.TokenAmountOut.Token)
}
