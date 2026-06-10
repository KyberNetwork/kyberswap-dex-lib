package whlp

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	rate := big.NewInt(1136787)
	extraBytes, err := json.Marshal(Extra{RateInQuote: rate})
	require.NoError(t, err)

	staticExtraBytes, err := json.Marshal(StaticExtra{
		Accountant:    accountantAddress,
		Depositor:     depositorAddress,
		CommunityCode: "kyberswap",
	})
	require.NoError(t, err)

	entityPool := entity.Pool{
		Address:  whlpVaultAddress,
		Exchange: "whlp",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: whlpVaultAddress, Decimals: 6},
			{Address: usdt0Address.Hex(), Decimals: 6},
		},
		Reserves:    []string{unlimitedReserve, unlimitedReserve},
		StaticExtra: string(staticExtraBytes),
		Extra:       string(extraBytes),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	amountIn := big.NewInt(1_000_000)
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: usdt0Address.Hex(), Amount: amountIn},
		TokenOut:      whlpVaultAddress,
	})
	require.NoError(t, err)

	expected := new(big.Int)
	bignumber.MulDivDown(expected, amountIn, bignumber.TenPowInt(6), rate)
	assert.Equal(t, 0, result.TokenAmountOut.Amount.Cmp(expected))
	assert.Equal(t, int64(879672), result.TokenAmountOut.Amount.Int64())
}
