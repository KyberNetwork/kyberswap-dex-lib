package bancorv3

import (
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMssgpackMarshalUnmarshal(t *testing.T) {
	var pools []*PoolSimulator
	rawPools := []string{
		`{
			"address": "0xeef417e1d5cc832e619ae18d2f140de2999dd4fb",
			"exchange": "bancor-v3",
			"type": "bancor-v3",
			"timestamp": 1708577191,
			"reserves": [
			  "16638855656409172130866",
			  "2491675002016096395750018",
			  "1042349177757924279511049",
			  "1343118445611083726107",
			  "21107545732",
			  "9830380626761692641693",
			  "6002398281476492",
			  "931938198338201388096656",
			  "3721760833489447674285",
			  "39315006361336560667820893",
			  "5337035548363797700952884",
			  "10903648670144275885454",
			  "113989250443046404146"
			],
			"tokens": [
			  {
				"address": "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"
			  },
			  {
				"address": "0x0d8775f648430679a709e98d2b0cb6250d2887ef"
			  },
			  {
				"address": "0x514910771af9ca656af840dff83e8264ecf986ca"
			  },
			  {
				"address": "0x4a220e6096b25eadb88358cb44068a3248254675"
			  },
			  {
				"address": "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"
			  },
			  {
				"address": "0x0d438f3b5175bebc262bf23753c1e53d03432bde"
			  },
			  {
				"address": "0xb9ef770b6a5e12e45983c5d80545258aa38f3b78"
			  },
			  {
				"address": "0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0"
			  },
			  {
				"address": "0xd33526068d116ce69f19a9ee46f0bd304f21a51f"
			  },
			  {
				"address": "0x444d6088b0f625f8c20192623b3c43001135e0fa"
			  },
			  {
				"address": "0xf629cbd94d3791c9250152bd8dfbdf380e2a3b9c"
			  },
			  {
				"address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
			  },
			  {
				"address": "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2"
			  }
			],
			"extra": "{\"nativeIdx\":11,\"collectionByPool\":{\"0x0d438f3b5175bebc262bf23753c1e53d03432bde\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0x0d8775f648430679a709e98d2b0cb6250d2887ef\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0x444d6088b0f625f8c20192623b3c43001135e0fa\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0x4a220e6096b25eadb88358cb44068a3248254675\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0x514910771af9ca656af840dff83e8264ecf986ca\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0xb9ef770b6a5e12e45983c5d80545258aa38f3b78\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0xd33526068d116ce69f19a9ee46f0bd304f21a51f\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\",\"0xf629cbd94d3791c9250152bd8dfbdf380e2a3b9c\":\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\"},\"poolCollections\":{\"0xde1b3ccfc45e3f5bff7f43516f2cd43364d883e4\":{\"networkFeePMM\":\"1000000\",\"poolData\":{\"0x0d438f3b5175bebc262bf23753c1e53d03432bde\":{\"poolToken\":\"0xa72279697db11f6f1ca9c3e666707edfc477c6d1\",\"tradingFeePPM\":\"10000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"186822398025481808453704\",\"baseTokenTradingLiquidity\":\"2299006284235592717615\",\"stakedBalance\":\"9830380626761692641693\"}},\"0x0d8775f648430679a709e98d2b0cb6250d2887ef\":{\"poolToken\":\"0xc70d66889c6cd013cc549daf0bdc96127ab1c9f0\",\"tradingFeePPM\":\"5000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"414374309755372641553263\",\"baseTokenTradingLiquidity\":\"1246662168787266546384465\",\"stakedBalance\":\"2491675002016096395750018\"}},\"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984\":{\"poolToken\":\"0x05bf6ca5f348d9575f360d6e29775f2477047a8d\",\"tradingFeePPM\":\"5000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"68345888955432886217622\",\"baseTokenTradingLiquidity\":\"7181649344089467383195\",\"stakedBalance\":\"16638855656409172130866\"}},\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\":{\"poolToken\":\"0x2ce37087559cbe8022fa5d70a0c502b7ae03f290\",\"tradingFeePPM\":\"11000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"5509439347237226780860059\",\"baseTokenTradingLiquidity\":\"8124966001\",\"stakedBalance\":\"21107545732\"}},\"0x444d6088b0f625f8c20192623b3c43001135e0fa\":{\"poolToken\":\"0x356d286a49f484b73e58d757d85fc5abc9ebf4f2\",\"tradingFeePPM\":\"5000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"50437196454796548287941\",\"baseTokenTradingLiquidity\":\"2990625733469916821076380\",\"stakedBalance\":\"39315006361336560667820893\"}},\"0x4a220e6096b25eadb88358cb44068a3248254675\":{\"poolToken\":\"0x8b2368faf88a4dd5b61c52b5862952331293b349\",\"tradingFeePPM\":\"5000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"76172782760868906789421\",\"baseTokenTradingLiquidity\":\"552001631294634594566\",\"stakedBalance\":\"1343118445611083726107\"}},\"0x514910771af9ca656af840dff83e8264ecf986ca\":{\"poolToken\":\"0x516c164a879892a156920a215855c3416616c46e\",\"tradingFeePPM\":\"12000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"14335608050565470317149842\",\"baseTokenTradingLiquidity\":\"589229401217545409667702\",\"stakedBalance\":\"1042349177757924279511049\"}},\"0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0\":{\"poolToken\":\"0xadf829f541a57ef2af4d8a07a7920f7229684dda\",\"tradingFeePPM\":\"5000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"290972290125233502589876\",\"baseTokenTradingLiquidity\":\"232320539326613740508175\",\"stakedBalance\":\"931938198338201388096656\"}},\"0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2\":{\"poolToken\":\"0x40dfb80a253414c07e8189b863424fb19521749b\",\"tradingFeePPM\":\"10000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"80325849522636437455911\",\"baseTokenTradingLiquidity\":\"29823899287168717896\",\"stakedBalance\":\"113989250443046404146\"}},\"0xb9ef770b6a5e12e45983c5d80545258aa38f3b78\":{\"poolToken\":\"0xb6279f7ca49876f9529fdc7983d65a03a819e2d0\",\"tradingFeePPM\":\"5000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"89825440856377923016553\",\"baseTokenTradingLiquidity\":\"3225590631277572\",\"stakedBalance\":\"6002398281476492\"}},\"0xd33526068d116ce69f19a9ee46f0bd304f21a51f\":{\"poolToken\":\"0x7bb2464326e623a353e00a37fa557628e865f014\",\"tradingFeePPM\":\"5000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"85170009817023063051249\",\"baseTokenTradingLiquidity\":\"2297714252318978272737\",\"stakedBalance\":\"3721760833489447674285\"}},\"0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee\":{\"poolToken\":\"0x256ed1d83e3e4efdda977389a5389c3433137dda\",\"tradingFeePPM\":\"8000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"15282570475460670519299723\",\"baseTokenTradingLiquidity\":\"3923946515599871999165\",\"stakedBalance\":\"10903648670144275885454\"}},\"0xf629cbd94d3791c9250152bd8dfbdf380e2a3b9c\":{\"poolToken\":\"0x9250fd963a7c7d23a1e5ca9ade6c43cf5e846b20\",\"tradingFeePPM\":\"5000\",\"tradingEnabled\":true,\"liquidity\":{\"bntTradingLiquidity\":\"1108653492911749135528936\",\"baseTokenTradingLiquidity\":\"2508557169821734221837438\",\"stakedBalance\":\"5337035548363797700952884\"}}},\"bnt\":\"0x1f573d6fb3f13d689ff844b4ce37794d79a7ff1c\"}}}",
			"staticExtra": "{\"bnt\":\"0x1f573d6fb3f13d689ff844b4ce37794d79a7ff1c\",\"chainId\":1}",
			"blockNumber": 19281309
		  }`,
	}
	for _, rawPool := range rawPools {
		poolEntity := new(entity.Pool)
		require.NoError(t, json.Unmarshal([]byte(rawPool), poolEntity))
		pool, err := NewPoolSimulator(*poolEntity)
		require.NoError(t, err)
		pools = append(pools, pool)
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(PoolSimulator)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(PoolSimulator{})...))
	}
}
