package cmeth

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var entityPoolData = entity.Pool{
	Address:  "0xb6f7d38e3eabbf69210afc2212fe82e0f1912b0",
	Exchange: DexType,
	Type:     DexType,
	Reserves: entity.PoolReserves{defaultReserves, defaultReserves},
	Tokens: []*entity.PoolToken{
		{Address: "0xe6829d9a7ee3040e1276fa75293bde931859e8fa", Decimals: 18, Swappable: true, Symbol: "cmETH"},
		{Address: "0xd5f7838f5c461feff7fe49ea5ebaf7728bb0adfa", Decimals: 18, Swappable: true, Symbol: "mETH"},
	},
	Extra:       `{"isTellerPaused":false,"assets":{"0xd5f7838f5c461feff7fe49ea5ebaf7728bb0adfa":{"allowDeposits":true},"0xe6829d9a7ee3040e1276fa75293bde931859e8fa":{"allowDeposits":false}},"accountantState":{"exchangeRate":1000000000000000000,"isPaused":false},"rateProviders":{"0xd5f7838f5c461feff7fe49ea5ebaf7728bb0adfa":{"isPeggedToBase":true,"rateProvider":"0x0000000000000000000000000000000000000000"},"0xe6829d9a7ee3040e1276fa75293bde931859e8fa":{"isPeggedToBase":false,"rateProvider":"0x0000000000000000000000000000000000000000"}}}`,
	StaticExtra: `{"accountant":"0x6049bd892f14669a4466e46981eced75d610a2ec","base":"0xd5f7838f5c461feff7fe49ea5ebaf7728bb0adfa","decimals":18}`,
}

func TestPoolSimulator_CalcAmountOut_mETH_to_cmETH(t *testing.T) {
	t.Parallel()

	p, err := NewPoolSimulator(entityPoolData)
	assert.NoError(t, err)

	amountIn := big.NewInt(1_000000000000000000)
	result, err := p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xd5f7838f5c461feff7fe49ea5ebaf7728bb0adfa", Amount: amountIn},
		TokenOut:      "0xe6829d9a7ee3040e1276fa75293bde931859e8fa",
	})
	assert.NoError(t, err)
	assert.Equal(t, amountIn.String(), result.TokenAmountOut.Amount.String())
}

func TestPoolSimulator_CalcAmountOut_cmETH_not_supported(t *testing.T) {
	t.Parallel()

	p, err := NewPoolSimulator(entityPoolData)
	assert.NoError(t, err)

	_, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xe6829d9a7ee3040e1276fa75293bde931859e8fa", Amount: big.NewInt(1)},
		TokenOut:      "0xd5f7838f5c461feff7fe49ea5ebaf7728bb0adfa",
	})
	assert.ErrorIs(t, err, ErrTellerAssetNotSupported)
}
