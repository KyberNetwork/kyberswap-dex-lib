package gyroeclp

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/goccy/go-json"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	rawPools := []string{
		`{
			"address": "0xe0e8ac08de6708603cfd3d23b613d2f80e3b7afb",
			"exchange": "gyroscope-eclp",
			"type": "gyroscope-eclp",
			"timestamp": 1705566147,
			"reserves": [
			  "1432237821990898965",
			  "2685567802993977683"
			],
			"tokens": [
			  {
				"address": "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
				"weight": 1,
				"swappable": true
			  },
			  {
				"address": "0xf951e335afb289353dc249e82926178eac7ded78",
				"weight": 1,
				"swappable": true
			  }
			],
			"extra": "{\"paused\":false,\"swapFeePercentage\":\"0x5af3107a4000\",\"paramsAlpha\":\"999500249875062469\",\"paramsBeta\":\"1010101010101010101\",\"paramsC\":\"705688316491160463\",\"paramsS\":\"708522406115622955\",\"paramsLambda\":\"500000000000000000000\",\"tauAlphaX\":\"-74798712145497721414789338637153095764\",\"tauAlphaY\":\"66371324089360848501248857841320837382\",\"tauBetaX\":\"83383678297259876539161659077817401265\",\"tauBetaY\":\"55201106815163337488949515922664830840\",\"u\":\"79090559955836985090533912561030798620\",\"v\":\"60763830337203480831932978680724761109\",\"w\":\"-5585063777148738251815296884188865778\",\"z\":\"3975485570515915653508200992108476537\",\"dSq\":\"100000000000000000082596734413730639400\",\"tokenRates\":[\"0x1003dadd43ba4f85\",\"0xe8c3a22e66c5342\"]}",
			"staticExtra": "{\"poolId\":\"0xe0e8ac08de6708603cfd3d23b613d2f80e3b7afb00020000000000000000058a\",\"poolType\":\"GyroE\",\"poolTypeVersion\":2,\"tokenDecimals\":[18,18],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}",
			"blockNumber": 19032529
		  }`,
	}
	var err error
	poolEntities := make([]*entity.Pool, len(rawPools))
	for i, rawPool := range rawPools {
		poolEntities[i] = new(entity.Pool)
		err = json.Unmarshal([]byte(rawPool), poolEntities[i])
		if err != nil {
			panic(err)
		}
	}
	pools := make([]*PoolSimulator, len(poolEntities))
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(*poolEntity)
		if err != nil {
			panic(err)
		}
	}
	return pools
}
