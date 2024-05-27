package tricryptong

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/samber/lo"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	rawPools := []string{
		// https://etherscan.io/address/0x2889302a794da87fbf1d6db415c1492194663d13#events
		"{\"address\":\"0x2889302a794da87fbf1d6db415c1492194663d13\",\"exchange\":\"curve-tricrypto-ng\",\"type\":\"curve-tricrypto-ng\",\"timestamp\":1710842900,\"reserves\":[\"3848079508071253519125552\",\"60997386412794855327\",\"1028200997183081004168\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x18084fba666a33d37592fa2633fd49a74dd93a88\",\"symbol\":\"tBTC\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0\",\"symbol\":\"wstETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"1707629\\\",\\\"InitialGamma\\\":\\\"11809167828997\\\",\\\"InitialAGammaTime\\\":1705051559,\\\"FutureA\\\":\\\"540000\\\",\\\"FutureGamma\\\":\\\"80500000000000\\\",\\\"FutureAGammaTime\\\":1705537322,\\\"D\\\":\\\"11990883592127090140834712\\\",\\\"PriceScale\\\":[\\\"66313464177401058702341\\\",\\\"3988288337309167729564\\\"],\\\"PriceOracle\\\":[\\\"63612706012126486095056\\\",\\\"3782761569503404058823\\\"],\\\"LastPrices\\\":[\\\"63608488224235038716789\\\",\\\"3782322291001686876800\\\"],\\\"LastPricesTimestamp\\\":1710838775,\\\"FeeGamma\\\":\\\"400000000000000\\\",\\\"MidFee\\\":\\\"1000000\\\",\\\"OutFee\\\":\\\"140000000\\\",\\\"LpSupply\\\":\\\"6209561906175920711602\\\",\\\"XcpProfit\\\":\\\"1005532234158713186\\\",\\\"VirtualPrice\\\":\\\"1002781276086899355\\\",\\\"AllowedExtraProfit\\\":\\\"100000000\\\",\\\"AdjustmentStep\\\":\\\"100000000000\\\",\\\"MaTime\\\":\\\"601\\\"}\",\"staticExtra\":\"{\\\"IsNativeCoins\\\":[false,false,false]}\",\"blockNumber\":19468099}",
	}
	pools := lo.Map(rawPools, func(rawPool string, _ int) *PoolSimulator {
		var poolEntity entity.Pool
		err := json.Unmarshal([]byte(rawPool), &poolEntity)
		if err != nil {
			panic(err)
		}
		p, err := NewPoolSimulator(poolEntity)
		if err != nil {
			panic(err)
		}
		return p
	})
	return pools
}
