package balancercomposablestable

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	poolEntities := []entity.Pool{
		{
			Address:      "0x9001cbbd96f54a658ff4e6e65ab564ded76a5431",
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			SwapFee:      0.000001,
			Exchange:     "balancer-composable-stable",
			Type:         "balancer-composable-stable",
			Timestamp:    1689789778,
			Reserves:     entity.PoolReserves{"2518960237189623226641", "2596148429266323438822175768385755", "3457262534881651304610"},
			Tokens: entity.PoolTokens{
				&entity.PoolToken{
					Address:   "0x60d604890feaa0b5460b28a424407c24fe89374a",
					Name:      "A",
					Symbol:    "",
					Weight:    333333333333333333,
					Swappable: true,
				},
				&entity.PoolToken{
					Address:   "0x9001cbbd96f54a658ff4e6e65ab564ded76a5431",
					Name:      "B",
					Symbol:    "",
					Weight:    333333333333333333,
					Swappable: true,
				},
				&entity.PoolToken{
					Address:   "0xbe9895146f7af43049ca1c1ae358b0541ea49704",
					Name:      "C",
					Symbol:    "",
					Weight:    333333333333333333,
					Swappable: true,
				},
			},
			Extra:       "{\"amplificationParameter\":{\"value\":700000,\"isUpdating\":false,\"precision\":1000},\"scalingFactors\":[1003649423771917631,1000000000000000000,1043680240732074966],\"bptIndex\":1,\"actualSupply\":6105781862789255176406,\"lastJoinExit\":{\"LastJoinExitAmplification\":700000,\"LastPostJoinExitInvariant\":6135006746648647084879},\"rateProviders\":[\"0x60d604890feaa0b5460b28a424407c24fe89374a\",\"0x0000000000000000000000000000000000000000\",\"0x7311e4bb8a72e7b300c5b8bde4de6cdaa822a5b1\"],\"tokensExemptFromYieldProtocolFee\":[false,false,false],\"tokenRateCaches\":[{\"Rate\":1003649423771917631,\"OldRate\":1003554274984131981,\"Duration\":21600,\"Expires\":1689845039},{\"Rate\":null,\"OldRate\":null,\"Duration\":null,\"Expires\":null},{\"Rate\":1043680240732074966,\"OldRate\":1043375386816533719,\"Duration\":21600,\"Expires\":1689845039}],\"protocolFeePercentageCacheSwapType\":0,\"protocolFeePercentageCacheYieldType\":0}",
			StaticExtra: "{\"vaultAddress\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\",\"poolId\":\"0x9001cbbd96f54a658ff4e6e65ab564ded76a543100000000000000000000050a\",\"tokenDecimals\":[18,18,18]}",
			TotalSupply: "2596148429272429220684965023562161",
		},
	}
	var err error
	pools := make([]*PoolSimulator, len(poolEntities))
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(poolEntity)
		if err != nil {
			panic(err)
		}
	}
	return pools
}
