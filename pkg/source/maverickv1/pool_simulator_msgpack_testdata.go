package maverickv1

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

var maverickPool, err = NewPoolSimulator(entity.Pool{
	Tokens: []*entity.PoolToken{
		{
			Address:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Decimals: 6,
		},
		{
			Address:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Decimals: 18,
		},
	},
	Reserves: []string{
		"7448514891591076678798",
		"4960078724015931105",
	},
	Extra:       "{\"fee\":400000000000000,\"protocolFeeRatio\":0,\"activeTick\":379,\"binCounter\":122,\"bins\":{\"1\":{\"reserveA\":17453201008635512394640,\"reserveB\":0,\"lowerTick\":375,\"kind\":0,\"mergeId\":0},\"10\":{\"reserveA\":0,\"reserveB\":632634315831505118,\"lowerTick\":384,\"kind\":0,\"mergeId\":0},\"11\":{\"reserveA\":0,\"reserveB\":568174206788937614,\"lowerTick\":385,\"kind\":0,\"mergeId\":0},\"12\":{\"reserveA\":0,\"reserveB\":0,\"lowerTick\":379,\"kind\":1,\"mergeId\":0},\"13\":{\"reserveA\":0,\"reserveB\":24179624473369718938,\"lowerTick\":384,\"kind\":1,\"mergeId\":0},\"15\":{\"reserveA\":7448514891591076678798,\"reserveB\":4960078724015931105,\"lowerTick\":379,\"kind\":3,\"mergeId\":0},\"16\":{\"reserveA\":1631153083876778654919,\"reserveB\":0,\"lowerTick\":373,\"kind\":0,\"mergeId\":0},\"17\":{\"reserveA\":14604684077518837517486,\"reserveB\":0,\"lowerTick\":374,\"kind\":0,\"mergeId\":0},\"2\":{\"reserveA\":21271280872855300434039,\"reserveB\":0,\"lowerTick\":376,\"kind\":0,\"mergeId\":0},\"23\":{\"reserveA\":0,\"reserveB\":0,\"lowerTick\":379,\"kind\":2,\"mergeId\":0},\"25\":{\"reserveA\":426150857836353291022,\"reserveB\":0,\"lowerTick\":373,\"kind\":2,\"mergeId\":0},\"3\":{\"reserveA\":25965452451154862154091,\"reserveB\":0,\"lowerTick\":377,\"kind\":0,\"mergeId\":0},\"32\":{\"reserveA\":0,\"reserveB\":30757567785565755,\"lowerTick\":386,\"kind\":0,\"mergeId\":0},\"34\":{\"reserveA\":0,\"reserveB\":0,\"lowerTick\":379,\"kind\":3,\"mergeId\":0},\"37\":{\"reserveA\":973003208635914825127,\"reserveB\":0,\"lowerTick\":372,\"kind\":0,\"mergeId\":0},\"4\":{\"reserveA\":22309339486762762891065,\"reserveB\":0,\"lowerTick\":378,\"kind\":0,\"mergeId\":0},\"41\":{\"reserveA\":28773102441950282148,\"reserveB\":0,\"lowerTick\":371,\"kind\":0,\"mergeId\":0},\"47\":{\"reserveA\":596733989717113121,\"reserveB\":0,\"lowerTick\":369,\"kind\":0,\"mergeId\":0},\"48\":{\"reserveA\":993638242463261223,\"reserveB\":0,\"lowerTick\":370,\"kind\":0,\"mergeId\":0},\"5\":{\"reserveA\":9361000987001231865441,\"reserveB\":6233632141023853827,\"lowerTick\":379,\"kind\":0,\"mergeId\":0},\"50\":{\"reserveA\":968206263636201246648,\"reserveB\":0,\"lowerTick\":376,\"kind\":2,\"mergeId\":0},\"53\":{\"reserveA\":2153035881950200782250,\"reserveB\":1433739158145507548,\"lowerTick\":379,\"kind\":2,\"mergeId\":0},\"6\":{\"reserveA\":0,\"reserveB\":10375023547668913537,\"lowerTick\":380,\"kind\":0,\"mergeId\":0},\"7\":{\"reserveA\":0,\"reserveB\":9381324932473456976,\"lowerTick\":381,\"kind\":0,\"mergeId\":0},\"8\":{\"reserveA\":0,\"reserveB\":8271837842446867401,\"lowerTick\":382,\"kind\":0,\"mergeId\":0},\"84\":{\"reserveA\":0,\"reserveB\":821816663509517,\"lowerTick\":387,\"kind\":0,\"mergeId\":0},\"87\":{\"reserveA\":0,\"reserveB\":140916273379942,\"lowerTick\":388,\"kind\":0,\"mergeId\":0},\"9\":{\"reserveA\":0,\"reserveB\":732155171838690157,\"lowerTick\":383,\"kind\":0,\"mergeId\":0}},\"binPositions\":{\"369\":{\"0\":47},\"370\":{\"0\":48},\"371\":{\"0\":41},\"372\":{\"0\":37},\"373\":{\"0\":16,\"2\":25},\"374\":{\"0\":17},\"375\":{\"0\":1},\"376\":{\"0\":2,\"2\":50},\"377\":{\"0\":3},\"378\":{\"0\":4},\"379\":{\"0\":5,\"1\":12,\"2\":53,\"3\":15},\"380\":{\"0\":6},\"381\":{\"0\":7},\"382\":{\"0\":8},\"383\":{\"0\":9},\"384\":{\"0\":10,\"1\":13},\"385\":{\"0\":11},\"386\":{\"0\":32},\"387\":{\"0\":84},\"388\":{\"0\":87}},\"binMap\":{\"5\":7721018714868875516017241010155757617493946277325927722722110067420054945792,\"6\":69907},\"liquidity\":60474424673766490639024,\"sqrtPriceX96\":42792872587486068317}",
	StaticExtra: "{\"tickSpacing\":198}",
})

// MsgpackTestPools ...
func MsgpackTestPools() []*Pool {
	return []*Pool{
		maverickPool,
	}
}
