package lido_steth

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {

	tokens := []string{"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "0xae7ab96520de3a18e5e111b5eaab095312d7fe84"}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: []string{"1", "1"},
		Tokens:   lo.Map(tokens, func(adr string, _ int) *entity.PoolToken { return &entity.PoolToken{Address: adr} }),
	}, valueobject.ChainIDEthereum)
	require.Nil(t, err)

	assert.Equal(t, []string{tokens[1]}, p.CanSwapFrom(tokens[0]))
	assert.Equal(t, 0, len(p.CanSwapFrom(tokens[1])))
	assert.Equal(t, []string{tokens[0]}, p.CanSwapTo(tokens[1]))
	assert.Equal(t, 0, len(p.CanSwapTo(tokens[0])))

	eth := bignumber.TenPowInt(18)
	testamount := []*big.Int{
		eth,
		new(big.Int).Mul(big.NewInt(2), eth),
		new(big.Int).Mul(big.NewInt(30), eth),
		new(big.Int).Mul(big.NewInt(100), eth),
	}

	for _, amountIn := range testamount {
		t.Run(fmt.Sprintf("deposit %v ETH in should get %v stETH out", amountIn, amountIn), func(t *testing.T) {
			tokAmountIn := pool.TokenAmount{Token: tokens[0], Amount: amountIn}
			got, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: tokAmountIn,
					TokenOut:      tokens[1],
					Limit:         nil,
				})
			})

			require.Nil(t, err)
			assert.Equal(t, amountIn, got.TokenAmountOut.Amount)
			assert.Equal(t, tokens[1], got.TokenAmountOut.Token)
			assert.Equal(t, big.NewInt(0), got.Fee.Amount)
			assert.Equal(t, int64(60000), got.Gas)

			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  tokAmountIn,
				TokenAmountOut: *got.TokenAmountOut,
				Fee:            *got.Fee,
				SwapInfo:       nil,
			})
		})
	}
}

func TestPoolSimulator_WrongChain(t *testing.T) {

	tokens := []string{"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "0xae7ab96520de3a18e5e111b5eaab095312d7fe84"}
	_, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: []string{"1", "1"},
		Tokens:   lo.Map(tokens, func(adr string, _ int) *entity.PoolToken { return &entity.PoolToken{Address: adr} }),
	}, valueobject.ChainIDAvalancheCChain)
	require.NotNil(t, err)
}
