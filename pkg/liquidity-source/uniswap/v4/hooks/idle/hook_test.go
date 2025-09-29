package idle

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0xae00f378721d06d5742b3144a76b7d574582b2733695b77c6f838376ba56b257","swapFee":3000,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1759125675,"reserves":["78573167530114264154","4454446867830008384973491"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x3a115f568c4b3d0c6e239b2e8f3d4cda3798f536","symbol":"IDLE","decimals":18,"swappable":true}],"extra":"{\"liquidity\":18708286933869706927918,\"sqrtPriceX96\":18864241370847553870267045580110,\"tickSpacing\":60,\"tick\":109459,\"ticks\":[{\"index\":-887220,\"liquidityGross\":18708286933869706927918,\"liquidityNet\":18708286933869706927918},{\"index\":887220,\"liquidityGross\":18708286933869706927918,\"liquidityNet\":-18708286933869706927918}]}","staticExtra":"{\"0x0\":[true,false],\"fee\":3000,\"tS\":60,\"hooks\":\"0xb69ec3f073ac2eb8d81fa0585523ca026124c0cc\",\"uR\":\"0x6ff5693b99212da76ad316178a184ab56d299b43\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":36168163}
`),
		&entityPool)
	poolSim = lo.Must(uniswapv4.NewPoolSimulator(entityPool, 8453))
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		0: {
			1: {
				"198168063968":      "10864758562295945",
				"11616794234322811": "636811092586386185100",
			},
		},
		1: {
			0: {
				"11616794234322811":     "198168063968",
				"636811092586386185100": "10861657250609829",
			},
		},
	})
}
