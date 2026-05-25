package stable

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCloneState(t *testing.T) {
	t.Parallel()
	p, err := NewPoolSimulator(entity.Pool{
		Address:  "0xf5448fc2beb9324900d08225fe4530ba3bbf654f",
		Exchange: "lista-stable",
		Type:     "lista-stable",
		Reserves: entity.PoolReserves{
			"48615751411650460241085692",
			"11206579925899312237017692",
			"59769030327001165128372730",
		},
		Tokens: []*entity.PoolToken{
			{Address: "0x55d398326f99059ff775485246999027b3197955", Symbol: "USDT", Decimals: 18},
			{Address: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", Symbol: "USDC", Decimals: 18},
		},
		Extra:       `{"initialA":"500000","futureA":"500000","initialATime":0,"futureATime":0,"swapFee":"100000","adminFee":"2000000000","oraclePrices":[998996790000000000,999662870000000000],"priceDiffThreshold":[50000000000000000,50000000000000000]}`,
		StaticExtra: `{"lpToken":"0xF6136d7e72446C724ecAeef514AE7B2ab4dbb60B","aPrecision":"100","precisionMultipliers":["1","1"],"rates":["1000000000000000000","1000000000000000000"],"isNativeCoins":[false,false]}`,
	})
	require.NoError(t, err)

	testutil.TestCloneState(t, p, poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "0x55d398326f99059ff775485246999027b3197955",
			Amount: new(big.Int).Mul(big.NewInt(1000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
		},
		TokenOut: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d",
	}, nil)
}
