package dvm

import (
	"encoding/json"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	pools := []string{
		// Pool data at block https://arbiscan.io/block/215765443

		// https://arbiscan.io/address/0xb627b318a537dff3883fcb7f0bd247ab6201b8d3#code
		"{\"address\":\"0xb627b318a537dff3883fcb7f0bd247ab6201b8d3\",\"swapFee\":100000000000000,\"exchange\":\"dodo-dvm\",\"type\":\"dodo-dvm\",\"timestamp\":1716863956,\"reserves\":[\"1001\",\"0\"],\"tokens\":[{\"address\":\"0x5330467941b3691a2c838769a58ddc5fca22ddec\",\"name\":\"BERD\",\"symbol\":\"BERD\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"name\":\"Wrapped Ether\",\"symbol\":\"WETH\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"10000000\\\",\\\"K\\\":\\\"500000000000000000\\\",\\\"B\\\":\\\"1001\\\",\\\"Q\\\":\\\"0\\\",\\\"B0\\\":\\\"1001\\\",\\\"Q0\\\":\\\"0\\\",\\\"R\\\":\\\"1\\\",\\\"mtFeeRate\\\":\\\"20000000000000\\\",\\\"lpFeeRate\\\":\\\"80000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xb627b318a537dff3883fcb7f0bd247ab6201b8d3\\\",\\\"lpToken\\\":\\\"0xb627b318a537dff3883fcb7f0bd247ab6201b8d3\\\",\\\"type\\\":\\\"DVM\\\",\\\"tokens\\\":[\\\"0x5330467941b3691a2c838769a58ddc5fca22ddec\\\",\\\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",

		// https://arbiscan.io/address/0x68276dc302d390245f3382eb4d2ea3a9317d46ef#code
		"{\"address\":\"0x68276dc302d390245f3382eb4d2ea3a9317d46ef\",\"swapFee\":3000000000000000,\"exchange\":\"dodo-dvm\",\"type\":\"dodo-dvm\",\"timestamp\":1716863956,\"reserves\":[\"15580539464573\",\"54845488636364795\"],\"tokens\":[{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"name\":\"Wrapped Ether\",\"symbol\":\"WETH\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\",\"name\":\"Dai Stablecoin\",\"symbol\":\"DAI\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"100000\\\",\\\"K\\\":\\\"1000000000000000000\\\",\\\"B\\\":\\\"15580539464573\\\",\\\"Q\\\":\\\"54845488636364795\\\",\\\"B0\\\":\\\"2923221347601320894515\\\",\\\"Q0\\\":\\\"0\\\",\\\"R\\\":\\\"1\\\",\\\"mtFeeRate\\\":\\\"600000000000000\\\",\\\"lpFeeRate\\\":\\\"2400000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0x68276dc302d390245f3382eb4d2ea3a9317d46ef\\\",\\\"lpToken\\\":\\\"0x68276dc302d390245f3382eb4d2ea3a9317d46ef\\\",\\\"type\\\":\\\"DVM\\\",\\\"tokens\\\":[\\\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\\\",\\\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",
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
