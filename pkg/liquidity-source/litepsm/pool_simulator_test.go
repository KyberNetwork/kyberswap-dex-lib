package litepsm

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCloneState(t *testing.T) {
	t.Parallel()
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(`{
		"address": "0xf6e72db5454dd049d0788e411b06cfaf16853042",
		"exchange": "lite-psm",
		"type": "lite-psm",
		"reserves": ["414303017692629270548465826","3850977525217835"],
		"tokens": [
			{"address": "0x6b175474e89094c44da98b954eedeac495271d0f","symbol": "DAI","decimals": 18,"swappable": true},
			{"address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol": "USDC","decimals": 6,"swappable": true}
		],
		"extra": "{}",
		"staticExtra": "{\"poc\":\"0x37305b1cd40574e4c5ce33f8e8306be057fd7341\"}"
	}`), &ep))

	p, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	// DAI in, USDC out (buyGem): 1 DAI → ~1 USDC
	testutil.TestCloneState(t, p, poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "0x6b175474e89094c44da98b954eedeac495271d0f",
			Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
		},
		TokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	}, nil)
}
