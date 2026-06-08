package whlp

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	vaultAddr  = "0x1359b05241ca5076c9f59605214f4f84114c0de8"
	quoteAddr  = "0xb8ce59fc3717ada4c02eadf9682a9e934f625ebb"
	acctAddr   = "0x470bd109A24f608590d85fc1f5a4B6e625E8bDfF"
	depositAddr = "0x340C9f6159ABc2bdfCC0E2b9Fe91D739006b41c1"
)

func newTestPool(t *testing.T, rate int64) *PoolSimulator {
	t.Helper()

	staticExtra, err := json.Marshal(StaticExtra{
		Accountant: common.HexToAddress(acctAddr),
		Depositor:  common.HexToAddress(depositAddr),
		QuoteAsset: common.HexToAddress(quoteAddr),
	})
	require.NoError(t, err)

	extra, err := json.Marshal(Extra{RateInQuote: big.NewInt(rate)})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:     vaultAddr,
		Exchange:    string(valueobject.ExchangeWhlp),
		Type:        DexType,
		Reserves:    []string{unlimitedReserve, unlimitedReserve},
		StaticExtra: string(staticExtra),
		Extra:       string(extra),
		Tokens: []*entity.PoolToken{
			{Address: vaultAddr, Decimals: 6},
			{Address: quoteAddr, Decimals: 6},
		},
	})
	require.NoError(t, err)
	return sim
}

func TestCalcAmountOutQuoteToShare(t *testing.T) {
	sim := newTestPool(t, 1_136_850)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: quoteAddr, Amount: big.NewInt(1_000_000)},
		TokenOut:      vaultAddr,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(879_623), res.TokenAmountOut.Amount.Int64())
}

func TestCalcAmountOutShareToQuote(t *testing.T) {
	sim := newTestPool(t, 1_136_850)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: vaultAddr, Amount: big.NewInt(1_000_000)},
		TokenOut:      quoteAddr,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1_136_850), res.TokenAmountOut.Amount.Int64())
}
