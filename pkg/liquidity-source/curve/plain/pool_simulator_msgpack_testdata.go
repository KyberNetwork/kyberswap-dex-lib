package plain

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/samber/lo"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	rawPools := []string{
		// plain3basic: http://etherscan.io/address/0xe7a3b38c39f97e977723bd1239c3470702568e7b
		"{\"address\":\"0xe7a3b38c39f97e977723bd1239c3470702568e7b\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708682750,\"reserves\":[\"103902458912250371998101\",\"96026429950922739854657\",\"90684626303\",\"289489289998600589912023\"],\"tokens\":[{\"address\":\"0xee586e7eaad39207f0549bc65f19e336942c992f\",\"symbol\":\"cEUR\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x1a7e4e63778b4f12a199c062f3efdd288afcbce8\",\"symbol\":\"agEUR\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x1abaea1f7c830bd89acc67ec4af516284b1bc33c\",\"symbol\":\"EURC\",\"decimals\":6,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"1000\\\",\\\"FutureA\\\":\\\"1000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xe7A3b38c39F97E977723bd1239C3470702568e7B\\\"}\"}",

		// plain2ethema: https://etherscan.io/address/0x94b17476a93b3262d87b9a326965d1e91f9c13e7#readContract
		"{\"address\":\"0x94b17476a93b3262d87b9a326965d1e91f9c13e7\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708930755,\"reserves\":[\"8189776041162322264444\",\"9661706603857954240258\",\"17827858048153259470189\"],\"tokens\":[{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"symbol\":\"ETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x856c4efb76c1d1ae02e20ceb03a2a6a08b0b8dc3\",\"symbol\":\"OETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"40000\\\",\\\"FutureA\\\":\\\"40000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0x94B17476A93b3262d87B9a326965D1E91f9c13E7\\\"}\"}",

		// plain3balances: https://etherscan.io/address/0xb9446c4Ef5EBE66268dA6700D26f96273DE3d571#code
		"{\"address\":\"0xb9446c4ef5ebe66268da6700d26f96273de3d571\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708930755,\"reserves\":[\"549022857960890312641141\",\"1075362632212\",\"46720010\",\"2069614823685039402821670\"],\"tokens\":[{\"address\":\"0x1a7e4e63778b4f12a199c062f3efdd288afcbce8\",\"symbol\":\"agEUR\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xc581b735a1688071a1746c968e0798d642ede491\",\"symbol\":\"EURT\",\"decimals\":6,\"swappable\":true},{\"address\":\"0xdb25f211ab05b1c97d595516f45794528a807ad8\",\"symbol\":\"EURS\",\"decimals\":2,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"20000\\\",\\\"FutureA\\\":\\\"20000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xb9446c4Ef5EBE66268dA6700D26f96273DE3d571\\\"}\"}",

		// plain4optimized: https://etherscan.io/address/0xda5b670ccd418a187a3066674a8002adc9356ad1#readContract
		"{\"address\":\"0xda5b670ccd418a187a3066674a8002adc9356ad1\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708930755,\"reserves\":[\"310644979221390280\",\"2806169166643327027\",\"360381510649494878\",\"218999711791367011\",\"3256514088341791400\"],\"tokens\":[{\"address\":\"0xd533a949740bb3306d119cc777fa900ba034cd52\",\"symbol\":\"CRV\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x9d409a0a012cfba9b15f6d4b36ac57a46966ab9a\",\"symbol\":\"yvBOOST\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x62b9c7356a2dc64a1969e19c23e4f579f9810aa7\",\"symbol\":\"cvxCRV\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xd38aeb759891882e78e957c80656572503d8c1b1\",\"symbol\":\"sCRV\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"1000\\\",\\\"FutureA\\\":\\\"1000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\"}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xDa5B670CcD418a187a3066674A8002Adc9356Ad1\\\"}\"}",

		// plain2price: https://etherscan.io/address/0x1539c2461d7432cc114b0903f1824079bfca2c92#readContract
		// the stored_rates change fast, use a script to fetch all test cases together at once
		"{\"address\":\"0x1539c2461d7432cc114b0903f1824079bfca2c92\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1708942235,\"reserves\":[\"207488005116042557636229\",\"47921035344869338429831\",\"256666057306386486195311\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"symbol\":\"crvUSD\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x83f20f44975d03b1b09e64809b757c47f942beea\",\"symbol\":\"sDAI\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"150000\\\",\\\"FutureA\\\":\\\"150000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"1000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\", \\\"RateMultipliers\\\":[\\\"1000000000000000000\\\", \\\"1057419823498475822\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0x1539c2461d7432cc114b0903f1824079BfCA2C92\\\"}\"}",

		// plain oracle: https://arbiscan.io/address/0x6eb2dc694eb516b16dc9fbc678c60052bbdd7d80#readContract
		"{\"address\":\"0x6eb2dc694eb516b16dc9fbc678c60052bbdd7d80\",\"exchange\":\"curve-stable-plain\",\"type\":\"curve-stable-plain\",\"timestamp\":1709021551,\"reserves\":[\"171562283322052190070\",\"159666449951883581558\",\"344265475511890460140\"],\"tokens\":[{\"address\":\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"symbol\":\"ETH\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x5979d7b546e38e414f7e9822514be443a4800529\",\"symbol\":\"wstETH\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"InitialA\\\":\\\"5000\\\",\\\"FutureA\\\":\\\"5000\\\",\\\"InitialATime\\\":0,\\\"FutureATime\\\":0,\\\"SwapFee\\\":\\\"4000000\\\",\\\"AdminFee\\\":\\\"5000000000\\\",\\\"RateMultipliers\\\":[\\\"1000000000000000000\\\",\\\"1158379174506084879\\\"]}\",\"staticExtra\":\"{\\\"APrecision\\\":\\\"100\\\",\\\"LpToken\\\":\\\"0xDbcD16e622c95AcB2650b38eC799f76BFC557a0b\\\",\\\"Oracle\\\":\\\"0xb1552c5e96b312d0bf8b554186f846c40614a540\\\"}\"}",
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
