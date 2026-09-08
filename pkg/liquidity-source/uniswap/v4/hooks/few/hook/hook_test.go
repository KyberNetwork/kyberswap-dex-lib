package hook_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	_ "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/few/hook"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestFewHook_RegisteredAndPassthrough builds the real DAI/fwDAI wrapper pool
// (state cross-checked on-chain: sqrtPriceX96=2^96, liquidity~5.57e22) and
// confirms:
//  1. its hook resolves via uniswapv4.HasSwapPermissions/GetHook to few/hook's
//     registered factory, not the generic auto-calibration fallback; and
//  2. a swap through it is an exact 1:1 passthrough in both directions.
func TestFewHook_RegisteredAndPassthrough(t *testing.T) {
	hookAddr := "0x85b648a64aed6307d5d5ce26e6ae086c17bde888" // DAI/fwDAI FewTokenHook

	entityPool := entity.Pool{
		Address:  "0xf906beb74154ca4d057b7079c90eb1044efaf40ef468e62ec983930cf80a1e2b",
		Exchange: "uniswap-v4",
		Type:     "uniswap-v4",
		Reserves: []string{"100000000000000000000000", "100000000000000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Symbol: "DAI", Decimals: 18, Swappable: true},
			{Address: "0x8a6fe57c08c84e0f4ee97aae68a62e820a37d259", Symbol: "fwDAI", Decimals: 18, Swappable: true},
		},
		StaticExtra: `{"0x0":[false,false],"fee":0,"tS":1,"hooks":"` + hookAddr + `","uR":"0x66a9893cc07d91d95644aedd05d03f95e1dba8af","pm2":"0x000000000022d473030f116ddee9f6b43ac78ba3","mc3":"0xca11bde05977b3631167028862be2a173976ca11"}`,
		Extra: `{"liquidity":55712545988302929134508,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":1,"tick":0,` +
			`"ticks":[{"index":-887272,"liquidityGross":55712545988302929134508,"liquidityNet":55712545988302929134508},` +
			`{"index":887272,"liquidityGross":55712545988302929134508,"liquidityNet":-55712545988302929134508}]}`,
	}

	simulator, err := uniswapv4.NewPoolSimulator(entityPool, valueobject.ChainIDEthereum)
	require.NoError(t, err)

	amountIn := big.NewInt(1_000000000000000000) // 1 DAI
	result, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: entityPool.Tokens[0].Address, Amount: amountIn},
		TokenOut:      entityPool.Tokens[1].Address,
	})
	require.NoError(t, err)
	assert.Equal(t, amountIn, result.TokenAmountOut.Amount, "DAI->fwDAI must be exact 1:1")

	resultRev, err := simulator.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: entityPool.Tokens[1].Address, Amount: amountIn},
		TokenOut:      entityPool.Tokens[0].Address,
	})
	require.NoError(t, err)
	assert.Equal(t, amountIn, resultRev.TokenAmountOut.Amount, "fwDAI->DAI must be exact 1:1")
}
