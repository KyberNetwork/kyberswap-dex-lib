package fulcrom

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	rawPools := []string{
		`{
			"address": "0x8c7ef34aa54210c76d6d5e475f43e0c11f876098",
			"type": "fulcrom",
			"timestamp": 1705352300,
			"reserves": [
				"3164844253",
				"407981862705453089405",
				"1488648645459",
				"628292027378",
				"11981305446",
				"261209766075",
				"280620655518",
				"37075925310",
				"9333383502",
				"977067545087"
			],
			"tokens": [
				{
					"address": "0x062e66477faf219f25d27dced647bf57c3107d52",
					"swappable": true
				},
				{
					"address": "0xe44fd7fcb2b1581822d0c862b68222998a0c299a",
					"swappable": true
				},
				{
					"address": "0xc21223249ca28397b4b6541dffaecc539bff0c59",
					"swappable": true
				},
				{
					"address": "0x66e428c3f67a68878562e79a0234c1f83c208770",
					"swappable": true
				},
				{
					"address": "0xb888d8dd1733d72681b30c00ee76bde93ae7aa93",
					"swappable": true
				},
				{
					"address": "0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0",
					"swappable": true
				},
				{
					"address": "0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15",
					"swappable": true
				},
				{
					"address": "0x9d97be214b68c7051215bb61059b4e299cd792c3",
					"swappable": true
				},
				{
					"address": "0x7589b70abb83427bb7049e08ee9fc6479ccb7a23",
					"swappable": true
				},
				{
					"address": "0xc9de0f3e08162312528ff72559db82590b481800",
					"swappable": true
				}
			],
			"extra": "{\"vault\":{\"hasDynamicFees\":true,\"includeAmmPrice\":false,\"isSwapEnabled\":true,\"stableSwapFeeBasisPoints\":1,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"taxBasisPoints\":50,\"totalTokenWeights\":100000,\"whitelistedTokens\":[\"0x062e66477faf219f25d27dced647bf57c3107d52\",\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\",\"0xc21223249ca28397b4b6541dffaecc539bff0c59\",\"0x66e428c3f67a68878562e79a0234c1f83c208770\",\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\",\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\",\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\",\"0x9d97be214b68c7051215bb61059b4e299cd792c3\",\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\",\"0xc9de0f3e08162312528ff72559db82590b481800\"],\"poolAmounts\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":3164844253,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":261209766075,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":628292027378,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":9333383502,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":37075925310,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":11981305446,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":280620655518,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":1488648645459,\"0xc9de0f3e08162312528ff72559db82590b481800\":977067545087,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":407981862705453089405},\"bufferAmounts\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":1600000000,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":160000000000,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":570000000000,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":6200000000,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":20000000000,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":12000000000,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":200000000000,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":910000000000,\"0xc9de0f3e08162312528ff72559db82590b481800\":590000000000,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":310000000000000000000},\"reservedAmounts\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":1483801599,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":152785893326,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":8530356764,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":4819564425,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":7157579282,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":1181604923,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":132962863475,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":58591018662,\"0xc9de0f3e08162312528ff72559db82590b481800\":694007455781,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":240424920555819866828},\"tokenDecimals\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":8,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":6,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":6,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":8,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":8,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":6,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":6,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":6,\"0xc9de0f3e08162312528ff72559db82590b481800\":9,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":18},\"stableTokens\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":false,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":false,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":true,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":false,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":false,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":false,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":false,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":true,\"0xc9de0f3e08162312528ff72559db82590b481800\":false,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":false},\"usdgAmounts\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":1269253204177016042299857,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":148762964598771913035464,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":628555183346144671484622,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":22389985595290798631700,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":25012737783739541468619,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":113265269274853567141994,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":160062949314803878462094,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":1469458366089667194649382,\"0xc9de0f3e08162312528ff72559db82590b481800\":84568490519676064583638,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":938836986036312645429339},\"maxUsdgAmounts\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":1500000000000000000000000,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":230000000000000000000000,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":1000000000000000000000000,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":59000000000000000000000,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":59000000000000000000000,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":250000000000000000000000,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":290000000000000000000000,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":1500000000000000000000000,\"0xc9de0f3e08162312528ff72559db82590b481800\":170000000000000000000000,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":1200000000000000000000000},\"tokenWeights\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":20000,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":3000,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":17000,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":1000,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":1000,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":4000,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":5000,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":25000,\"0xc9de0f3e08162312528ff72559db82590b481800\":3000,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":21000},\"priceFeed\":{\"minPrices\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":42858111666670000000000000000000000,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":531590000000000000000000000000,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":1000000000000000000000000000000,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":251272000000000000000000000000000,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":70023600000000000000000000000000,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":10259160000000000000000000000000,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":578210000000000000000000000000,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":1000000000000000000000000000000,\"0xc9de0f3e08162312528ff72559db82590b481800\":95079800000000000000000000000000,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":2526105000000000000000000000000000},\"maxPrices\":{\"0x062e66477faf219f25d27dced647bf57c3107d52\":42858111666670000000000000000000000,\"0x0e517979c2c1c1522ddb0c73905e0d39b3f990c0\":531590000000000000000000000000,\"0x66e428c3f67a68878562e79a0234c1f83c208770\":1000000000000000000000000000000,\"0x7589b70abb83427bb7049e08ee9fc6479ccb7a23\":251272000000000000000000000000000,\"0x9d97be214b68c7051215bb61059b4e299cd792c3\":70023600000000000000000000000000,\"0xb888d8dd1733d72681b30c00ee76bde93ae7aa93\":10259160000000000000000000000000,\"0xb9ce0dd29c91e02d4620f57a66700fc5e41d6d15\":578210000000000000000000000000,\"0xc21223249ca28397b4b6541dffaecc539bff0c59\":1000000000000000000000000000000,\"0xc9de0f3e08162312528ff72559db82590b481800\":95079800000000000000000000000000,\"0xe44fd7fcb2b1581822d0c862b68222998a0c299a\":2526105000000000000000000000000000}},\"usdg\":{\"address\":\"0xB09BD2bAf03e19550473a5DC1D5023805E04a4f5\",\"totalSupply\":4208732677493008283439248},\"UseSwapPricing\":false}}"
		}`,
	}
	poolEntites := make([]entity.Pool, len(rawPools))
	for i, rawPool := range rawPools {
		err := json.Unmarshal([]byte(rawPool), &poolEntites[i])
		if err != nil {
			panic(err)
		}
	}
	var err error
	pools := make([]*PoolSimulator, len(rawPools))
	for i, poolEntity := range poolEntites {
		pools[i], err = NewPoolSimulator(poolEntity)
		if err != nil {
			panic(err)
		}
	}
	return pools
}
