package dpp

import (
	"encoding/json"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	pools := []string{
		// Pool data at block https://arbiscan.io/block/215783877

		// https://arbiscan.io/address/0x8f11519f4f7c498e1f940b9de187d9c390321016#code
		"{\"address\":\"0x8f11519f4f7c498e1f940b9de187d9c390321016\",\"swapFee\":3000000000000000,\"exchange\":\"dodo-dpp\",\"type\":\"dodo-dpp\",\"timestamp\":1716868655,\"reserves\":[\"5682349893627314\",\"18472539\"],\"tokens\":[{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"name\":\"Wrapped Ether\",\"symbol\":\"WETH\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"name\":\"Tether USD\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"1200000000\\\",\\\"K\\\":\\\"1000000000000000000\\\",\\\"B\\\":\\\"5682349893627314\\\",\\\"Q\\\":\\\"18472539\\\",\\\"B0\\\":\\\"10116304445839343\\\",\\\"Q0\\\":\\\"9000000\\\",\\\"R\\\":\\\"1\\\",\\\"mtFeeRate\\\":\\\"0\\\",\\\"lpFeeRate\\\":\\\"3000000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0x8f11519f4f7c498e1f940b9de187d9c390321016\\\",\\\"lpToken\\\":\\\"\\\",\\\"type\\\":\\\"DPP\\\",\\\"tokens\\\":[\\\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\\\",\\\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",

		// https://arbiscan.io/address/0xb7392c0d85676de049121771c1edb31edd446336#code
		"{\"address\":\"0xb7392c0d85676de049121771c1edb31edd446336\",\"swapFee\":500000000000000,\"exchange\":\"dodo-dpp\",\"type\":\"dodo-dpp\",\"timestamp\":1716868655,\"reserves\":[\"900000000000000000\",\"100000\"],\"tokens\":[{\"address\":\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\",\"name\":\"Magic Internet Money\",\"symbol\":\"MIM\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\",\"name\":\"USD Coin\",\"symbol\":\"USDC\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"1000000\\\",\\\"K\\\":\\\"250000000000000\\\",\\\"B\\\":\\\"900000000000000000\\\",\\\"Q\\\":\\\"100000\\\",\\\"B0\\\":\\\"900000000000000000\\\",\\\"Q0\\\":\\\"100000\\\",\\\"R\\\":\\\"0\\\",\\\"mtFeeRate\\\":\\\"0\\\",\\\"lpFeeRate\\\":\\\"500000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xb7392c0d85676de049121771c1edb31edd446336\\\",\\\"lpToken\\\":\\\"\\\",\\\"type\\\":\\\"DPP\\\",\\\"tokens\\\":[\\\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\\\",\\\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",
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
