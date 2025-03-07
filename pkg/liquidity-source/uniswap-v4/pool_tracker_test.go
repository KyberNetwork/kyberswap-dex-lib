package uniswapv4

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

func TestPoolTracker_GetNewPoolState(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	pt := &PoolTracker{
		config:        &Config{DexID: DexType, StateViewAddress: "0x7fFE42C4a5DEeA5b0feC41C94C136Cf115597227"},
		ethrpcClient:  ethrpc.New("https://ethereum.kyberengineering.io").SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
		graphqlClient: graphqlpkg.NewClient(os.ExpandEnv("https://gateway.thegraph.com/api/$THEGRAPH_API_KEY/subgraphs/id/DiYPVdygkfjDWhbxGSqAQxwBKmfKnkWQojqeM2rkLb3G")),
	}
	got, err := pt.GetNewPoolState(context.Background(),
		entity.Pool{Address: "0x6b77c5119ea25b4b46ec79166075eed433bf8ad4bfe907490bb06305e3c0012a",
			StaticExtra: `{"tS":200}`},
		pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	t.Log(got)
}

func TestPoolTracker_GetTickFromStateView(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	pt := &PoolTracker{
		config: &Config{
			DexID:                  DexType,
			StateViewAddress:       "0x76fd297e2d437cd7f76d50f01afe6160f86e9990",
			FetchTickFromStateView: true,
		},
		ethrpcClient: ethrpc.New("https://arbitrum.kyberengineering.io").
			SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
	}

	// need to fetch fresh data each time
	orgPoolData := `{"address":"0x70bf44c3a9b6b047bf60e5a05968225dbf3d6a5b9e8a95a73727e48921e889c1","swapFee":3000,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1741343362,"reserves":["42879327258","38068162626172"],"tokens":[{"address":"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f","name":"","symbol":"","decimals":8,"weight":0,"swappable":true},{"address":"0xaf88d065e77c8cc2239327c5edb3a432268e5831","name":"","symbol":"","decimals":6,"weight":0,"swappable":true}],"extra":"{\"liquidity\":1277629525089,\"sqrtPriceX96\":2360676953638630026402609772996,\"tickSpacing\":60,\"tick\":67890,\"ticks\":[{\"index\":-887220,\"liquidityGross\":46566348,\"liquidityNet\":46566348},{\"index\":47880,\"liquidityGross\":2692586298,\"liquidityNet\":2692586298},{\"index\":62400,\"liquidityGross\":545744,\"liquidityNet\":545744},{\"index\":63960,\"liquidityGross\":521798631,\"liquidityNet\":521798631},{\"index\":65520,\"liquidityGross\":15294300594,\"liquidityNet\":15294300594},{\"index\":65820,\"liquidityGross\":4668609323,\"liquidityNet\":4668609323},{\"index\":65880,\"liquidityGross\":186731457,\"liquidityNet\":186731457},{\"index\":65940,\"liquidityGross\":144141615,\"liquidityNet\":144141615},{\"index\":66060,\"liquidityGross\":83651384,\"liquidityNet\":83651384},{\"index\":66180,\"liquidityGross\":972297097,\"liquidityNet\":972297097},{\"index\":66240,\"liquidityGross\":70340765,\"liquidityNet\":70340765},{\"index\":66360,\"liquidityGross\":12213884,\"liquidityNet\":12213884},{\"index\":66480,\"liquidityGross\":1738994995,\"liquidityNet\":1738994995},{\"index\":66540,\"liquidityGross\":32539305,\"liquidityNet\":32539305},{\"index\":66600,\"liquidityGross\":3774863806,\"liquidityNet\":3774863806},{\"index\":66660,\"liquidityGross\":1739674294,\"liquidityNet\":1739674294},{\"index\":66720,\"liquidityGross\":700496243,\"liquidityNet\":700496243},{\"index\":66840,\"liquidityGross\":1304724953,\"liquidityNet\":1304724953},{\"index\":66960,\"liquidityGross\":830713771,\"liquidityNet\":830713771},{\"index\":67020,\"liquidityGross\":478190353,\"liquidityNet\":478190353},{\"index\":67080,\"liquidityGross\":8500079103,\"liquidityNet\":8500079103},{\"index\":67140,\"liquidityGross\":3274180541,\"liquidityNet\":3274180541},{\"index\":67200,\"liquidityGross\":590711827,\"liquidityNet\":590711827},{\"index\":67260,\"liquidityGross\":475598169612,\"liquidityNet\":475598169612},{\"index\":67320,\"liquidityGross\":751959522781,\"liquidityNet\":751951813751},{\"index\":67380,\"liquidityGross\":42696134,\"liquidityNet\":42696134},{\"index\":67440,\"liquidityGross\":17172026119,\"liquidityNet\":-13383473713},{\"index\":67500,\"liquidityGross\":9065986634,\"liquidityNet\":9054543842},{\"index\":67560,\"liquidityGross\":17794990,\"liquidityNet\":17794990},{\"index\":67620,\"liquidityGross\":11340325,\"liquidityNet\":-34019},{\"index\":67680,\"liquidityGross\":7119024283,\"liquidityNet\":7119024283},{\"index\":67740,\"liquidityGross\":489462843,\"liquidityNet\":-478224169},{\"index\":67800,\"liquidityGross\":4581938768,\"liquidityNet\":2585490930},{\"index\":67860,\"liquidityGross\":2809635437,\"liquidityNet\":-2537228883},{\"index\":67920,\"liquidityGross\":168169499,\"liquidityNet\":-56789643},{\"index\":67980,\"liquidityGross\":9274412105,\"liquidityNet\":-9270822187},{\"index\":68040,\"liquidityGross\":7374191630,\"liquidityNet\":-2692633718},{\"index\":68160,\"liquidityGross\":122306865,\"liquidityNet\":11593251},{\"index\":68220,\"liquidityGross\":1530147513,\"liquidityNet\":-1481075421},{\"index\":68280,\"liquidityGross\":9828897517,\"liquidityNet\":-7604566017},{\"index\":68340,\"liquidityGross\":2167744825,\"liquidityNet\":-2167744825},{\"index\":68400,\"liquidityGross\":112074630,\"liquidityNet\":336206},{\"index\":68460,\"liquidityGross\":5146982916,\"liquidityNet\":-5146982916},{\"index\":68520,\"liquidityGross\":835294102,\"liquidityNet\":-835294102},{\"index\":68580,\"liquidityGross\":11867846278,\"liquidityNet\":-11754419230},{\"index\":68640,\"liquidityGross\":61671821,\"liquidityNet\":-61671821},{\"index\":68700,\"liquidityGross\":810030380,\"liquidityNet\":-810030380},{\"index\":68760,\"liquidityGross\":1218531650,\"liquidityNet\":-1218531650},{\"index\":68820,\"liquidityGross\":448797252,\"liquidityNet\":-448797252},{\"index\":69000,\"liquidityGross\":1238352968,\"liquidityNet\":-1238352968},{\"index\":69060,\"liquidityGross\":1223689880179,\"liquidityNet\":-1223689880179},{\"index\":69120,\"liquidityGross\":128345421,\"liquidityNet\":-128345421},{\"index\":69480,\"liquidityGross\":119597263,\"liquidityNet\":-119597263},{\"index\":69540,\"liquidityGross\":972297097,\"liquidityNet\":-972297097},{\"index\":69600,\"liquidityGross\":70340765,\"liquidityNet\":-70340765},{\"index\":69720,\"liquidityGross\":24536046,\"liquidityNet\":-24536046},{\"index\":69840,\"liquidityGross\":498897587,\"liquidityNet\":-498897587},{\"index\":69960,\"liquidityGross\":48204744,\"liquidityNet\":-48204744},{\"index\":70020,\"liquidityGross\":521798631,\"liquidityNet\":-521798631},{\"index\":70080,\"liquidityGross\":156586486,\"liquidityNet\":-156586486},{\"index\":70500,\"liquidityGross\":444839200,\"liquidityNet\":-444839200},{\"index\":71100,\"liquidityGross\":482488,\"liquidityNet\":-482488},{\"index\":71160,\"liquidityGross\":830713771,\"liquidityNet\":-830713771},{\"index\":72420,\"liquidityGross\":1267173328,\"liquidityNet\":-1267173328},{\"index\":72480,\"liquidityGross\":13885119,\"liquidityNet\":-13885119},{\"index\":72600,\"liquidityGross\":545744,\"liquidityNet\":-545744},{\"index\":72840,\"liquidityGross\":186731457,\"liquidityNet\":-186731457},{\"index\":73140,\"liquidityGross\":1139734444,\"liquidityNet\":-1139734444},{\"index\":887220,\"liquidityGross\":2739152646,\"liquidityNet\":-2739152646}]}","staticExtra":"{\"0x0\":[false,false],\"fee\":3000,\"tS\":60,\"hooks\":\"0x0000000000000000000000000000000000000000\",\"uR\":\"0xa51afafe0263b40edaef0df8781ea9aa03e381a3\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}"}`
	var orgPool entity.Pool
	require.NoError(t, json.Unmarshal([]byte(orgPoolData), &orgPool))

	// normal case, no changes, should get back as is
	{
		got, err := pt.GetNewPoolState(
			context.Background(),
			orgPool,
			pool.GetNewPoolStateParams{
				Logs: []types.Log{
					{
						Topics: []common.Hash{
							common.HexToHash("0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c"),
							common.BigToHash(big.NewInt(-887220)),
							common.BigToHash(big.NewInt(62400)),
							{},
						},
					},
				},
			},
		)
		require.NoError(t, err)
		t.Log(got)
		got.BlockNumber = 0
		assert.Equal(t, orgPool, got)
	}
}
