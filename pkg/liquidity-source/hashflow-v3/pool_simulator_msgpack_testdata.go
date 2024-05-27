package hashflowv3

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	poolEntities := []*entity.Pool{
		{
			Address:  "hashflow_v3_mm22_0xd26114cd6ee289accf82350c8d8487fedb8a0c07_0xdac17f958d2ee523a2206206994597c13d831ec7",
			Exchange: "hashflow-v3",
			Type:     "hashflow-v3",
			Reserves: []string{"64160215600609997156352", "152481964"},
			Tokens: []*entity.PoolToken{
				{Address: "0xd26114cd6ee289accf82350c8d8487fedb8a0c07", Decimals: 18, Swappable: true},
				{Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Decimals: 6, Swappable: true},
			},
			Extra:       "{\"zeroToOnePriceLevels\":[{\"q\":\"21.491858434308554\",\"p\":\"0.6924563136573486\"},{\"q\":\"2127.693984996547\",\"p\":\"0.6924563136573486\"},{\"q\":\"6450.785753788268\",\"p\":\"0.695858410957807\"},{\"q\":\"7095.864329167098\",\"p\":\"0.6955119978476955\"},{\"q\":\"7805.450762083805\",\"p\":\"0.6951337575443223\"},{\"q\":\"8588.352341200025\",\"p\":\"0.6945303753566658\"},{\"q\":\"9458.145774233493\",\"p\":\"0.6932765640141211\"},{\"q\":\"10403.960351656831\",\"p\":\"0.6927876203981647\"},{\"q\":\"11466.95813207097\",\"p\":\"0.6908910457830065\"},{\"q\":\"741.5123129786516\",\"p\":\"0.6865341216331126\"}],\"oneToZeroPriceLevels\":[{\"q\":\"1.52481964177280676980070027723634\",\"p\":\"1.414875966391418599745334487109784\"},{\"q\":\"150.957144535507867877909404465650\",\"p\":\"1.414875966391418599745334487109784\"}]}",
			StaticExtra: "{\"marketMaker\":\"mm22\"}",
		},
	}
	var err error
	pools := make([]*PoolSimulator, len(poolEntities))
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(*poolEntity)
		if err != nil {
			panic(err)
		}
	}
	return pools
}
