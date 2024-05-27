package classical

import (
	"encoding/json"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	pools := []string{
		// Pool data at block https://arbiscan.io/block/214416932

		// https://arbiscan.io/address/0xb42a054d950dafd872808b3c839fbb7afb86e14c#readContract
		"{\"address\":\"0xb42a054d950dafd872808b3c839fbb7afb86e14c\",\"swapFee\":3000000000000000,\"exchange\":\"dodo-classical\",\"type\":\"dodo-classical\",\"timestamp\":1716521335,\"reserves\":[\"5293182\",\"10402621507\"],\"tokens\":[{\"address\":\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\",\"name\":\"Wrapped BTC\",\"symbol\":\"WBTC\",\"decimals\":8,\"weight\":50,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"name\":\"USD Coin (Arb1)\",\"symbol\":\"USDC\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"B\\\":\\\"5293182\\\",\\\"Q\\\":\\\"10402621507\\\",\\\"B0\\\":\\\"5313565\\\",\\\"Q0\\\":\\\"10388770142\\\",\\\"rStatus\\\":1,\\\"oraclePrice\\\":\\\"678741575565600000000\\\",\\\"k\\\":\\\"300000000000000000\\\",\\\"mtFeeRate\\\":\\\"600000000000000\\\",\\\"lpFeeRate\\\":\\\"2400000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xb42a054d950dafd872808b3c839fbb7afb86e14c\\\",\\\"lpToken\\\":\\\"0xb94904bbe8a625709162dc172875fbc51c477abb\\\",\\\"type\\\":\\\"CLASSICAL\\\",\\\"tokens\\\":[\\\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\\\",\\\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",

		// https://arbiscan.io/address/0xe4b2dfc82977dd2dce7e8d37895a6a8f50cbb4fb
		"{\"address\":\"0xe4b2dfc82977dd2dce7e8d37895a6a8f50cbb4fb\",\"swapFee\":10000000000000,\"exchange\":\"dodo-classical\",\"type\":\"dodo-classical\",\"timestamp\":1716521335,\"reserves\":[\"1444873953831\",\"578850766374\"],\"tokens\":[{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"name\":\"Tether USD\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"name\":\"USD Coin (Arb1)\",\"symbol\":\"USDC\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"B\\\":\\\"1444873953831\\\",\\\"Q\\\":\\\"578850766374\\\",\\\"B0\\\":\\\"978121462386\\\",\\\"Q0\\\":\\\"1045528008085\\\",\\\"rStatus\\\":2,\\\"oraclePrice\\\":\\\"1000000000000000000\\\",\\\"k\\\":\\\"200000000000000\\\",\\\"mtFeeRate\\\":\\\"10000000000000\\\",\\\"lpFeeRate\\\":\\\"0\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xe4b2dfc82977dd2dce7e8d37895a6a8f50cbb4fb\\\",\\\"lpToken\\\":\\\"0x82b423848cdd98740fb57f961fa692739f991633\\\",\\\"type\\\":\\\"CLASSICAL\\\",\\\"tokens\\\":[\\\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\\\",\\\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",
	}
	sims := lo.Map(pools, func(poolRedis string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(poolRedis), &poolEntity)
		if err != nil {
			panic(err)
		}
		p, err := NewPoolSimulator(poolEntity)
		if err != nil {
			panic(err)
		}
		return p
	})
	return sims
}
