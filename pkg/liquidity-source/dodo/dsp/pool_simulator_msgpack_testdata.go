package dsp

import (
	"encoding/json"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	pools := []string{
		// Pool data at block https://arbiscan.io/block/215792414

		// https://arbiscan.io/address/0xa6ec95be503f803bce9e7dd498602f1b28c9a02a#code
		"{\"address\":\"0xa6ec95be503f803bce9e7dd498602f1b28c9a02a\",\"swapFee\":100000000000000,\"exchange\":\"dodo-dsp\",\"type\":\"dodo-dsp\",\"timestamp\":1716870877,\"reserves\":[\"33336489800302\",\"1888512\"],\"tokens\":[{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"name\":\"Wrapped Ether\",\"symbol\":\"WETH\",\"decimals\":18,\"weight\":50,\"swappable\":true},{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"name\":\"Tether USD\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"3723935145\\\",\\\"K\\\":\\\"100000000000000\\\",\\\"B\\\":\\\"33336489800302\\\",\\\"Q\\\":\\\"1888512\\\",\\\"B0\\\":\\\"270192202826890\\\",\\\"Q0\\\":\\\"1005850\\\",\\\"R\\\":\\\"1\\\",\\\"mtFeeRate\\\":\\\"20000000000000\\\",\\\"lpFeeRate\\\":\\\"80000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xa6ec95be503f803bce9e7dd498602f1b28c9a02a\\\",\\\"lpToken\\\":\\\"0xa6ec95be503f803bce9e7dd498602f1b28c9a02a\\\",\\\"type\\\":\\\"DSP\\\",\\\"tokens\\\":[\\\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\\\",\\\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",

		// https://arbiscan.io/address/0xd55128cbdba933bcf9b5f508108129ffe7e2e9bb#code
		"{\"address\":\"0xd55128cbdba933bcf9b5f508108129ffe7e2e9bb\",\"swapFee\":3000000000000000,\"exchange\":\"dodo-dsp\",\"type\":\"dodo-dsp\",\"timestamp\":1716870877,\"reserves\":[\"233467\",\"1117670600914973\"],\"tokens\":[{\"address\":\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"name\":\"Tether USD\",\"symbol\":\"USDT\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0x0c1cf6883efa1b496b01f654e247b9b419873054\",\"name\":\"SushiSwap LP Token\",\"symbol\":\"SLP\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"i\\\":\\\"538000000000000000000000000\\\",\\\"K\\\":\\\"100000000000000000\\\",\\\"B\\\":\\\"233467\\\",\\\"Q\\\":\\\"1117670600914973\\\",\\\"B0\\\":\\\"1036546\\\",\\\"Q0\\\":\\\"536995355922673\\\",\\\"R\\\":\\\"1\\\",\\\"mtFeeRate\\\":\\\"600000000000000\\\",\\\"lpFeeRate\\\":\\\"2400000000000000\\\",\\\"swappable\\\":true}\",\"staticExtra\":\"{\\\"poolId\\\":\\\"0xd55128cbdba933bcf9b5f508108129ffe7e2e9bb\\\",\\\"lpToken\\\":\\\"0xd55128cbdba933bcf9b5f508108129ffe7e2e9bb\\\",\\\"type\\\":\\\"DSP\\\",\\\"tokens\\\":[\\\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\\\",\\\"0x0c1cf6883efa1b496b01f654e247b9b419873054\\\"],\\\"dodoV1SellHelper\\\":\\\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\\\"}\"}",
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
