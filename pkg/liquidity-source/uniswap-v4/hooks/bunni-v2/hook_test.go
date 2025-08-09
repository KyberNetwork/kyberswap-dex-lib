package bunniv2

import (
	"context"
	"encoding/json"
	"log"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHook_Track(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://ethereum.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	p := &entity.Pool{
		Address: "0xd9f673912e1da331c9e56c5f0dbc7273c0eb684617939a375ec5e227c62d6707",
		Tokens: []*entity.PoolToken{
			{
				Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			{
				Address: "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
		},
		StaticExtra: "{\"tickSpacing\":1}",
	}

	cfg := &uniswapv4.Config{
		ChainID: 1,
	}

	hookExtra := `{"HookletExtra":"","HookletAddress":"0x00ece5a72612258f20eb24573c544f9dd8c5000c","LDFAddress":"0x0000000000000000000000000000000000000000","HookFee":"0","PoolManagerReserves":["85550931376972","79924599940186"],"LdfState":[1,255,255,251,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Vaults":[{"Address":"0xe0a80d35bb6618cba260120b279d357978c42bce","Decimals":6,"RedeemRate":"1045269391137680003","MaxDeposit":"25023883317638"},{"Address":"0x7c280dbdef569e96c7919251bd2b0edf0734c5a8","Decimals":6,"RedeemRate":"1039317907566756282","MaxDeposit":"25023883317638"}],"AmAmm":{"AmAmmManager":"0x0000000000000000000000000000000000000000","SwapFee0For1":"0","SwapFee1For0":"0"},"ObservationState":{"Index":1,"Cardinality":25,"CardinalityNext":25,"IntermediateObservation":{"BlockTimestamp":1754714435,"PrevTick":-3,"TickCumulative":-3050736,"Initialized":true}},"CuratorFees":{"FeeRate":"0"},"Observations":[{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":3708425450,"PrevTick":-3005853,"TickCumulative":-26698954285081392,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":3600,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":4080116892,"PrevTick":-6542852,"TickCumulative":16647030664248144,"Initialized":false},{"BlockTimestamp":1749684419,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false},{"BlockTimestamp":0,"PrevTick":0,"TickCumulative":0,"Initialized":false}],"HookParams":{"FeeMin":"5","FeeMax":"5","FeeQuadraticMultiplier":"0","FeeTwapSecondsAgo":0,"SurgeFeeHalfLife":"60","SurgeFeeAutostartThreshold":120,"VaultSurgeThreshold0":"100","VaultSurgeThreshold1":"100","AmAmmEnabled":false,"OracleMinInterval":1800},"Slot0":{"SqrtPriceX96":"79218849896775008690452952280","Tick":-3,"LastSwapTimestamp":1754714435,"LastSurgeTimestamp":1754690351},"BunniState":{"Hooklet":"0x0000000000000000000000000000000000000000","TwapSecondsAgo":43200,"LdfParams":[0,255,255,253,0,4,2,250,240,128,29,205,101,0,0,4,11,235,194,0,29,205,101,0,59,154,202,0,0,0,0,0],"HookParams":"AAAFAAAFAAAAAAAAAAAAADwAeABkAGQAZABkBwgBLAAAAAcIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","LdfType":2,"MinRawTokenRatio0":"100000","TargetRawTokenRatio0":"200000","MaxRawTokenRatio0":"300000","MinRawTokenRatio1":"100000","TargetRawTokenRatio1":"200000","MaxRawTokenRatio1":"300000","Currency0Decimals":6,"Currency1Decimals":6,"RawBalance0":"430722373692","RawBalance1":"4978507390520","Reserve0":"1898867969173","Reserve1":"16743738242891","IdleBalance":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,19,217,204,146,122,22]},"VaultSharePrices":{"Initialized":true,"SharedPrice0":"1045264221165641923","SharedPrice1":"1039313291905914631"}}`

	h := NewHook(&uniswapv4.HookParam{
		Cfg:         cfg,
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0x000052423c1db6b7ff8641b85a7eefc7b2791888"),
		Pool:        p,
	})

	s, err := h.Track(context.Background(), &uniswapv4.HookParam{
		Cfg:       cfg,
		RpcClient: rpcClient,
		Pool:      p,
		HookExtra: hookExtra,
	})
	require.NoError(t, err)

	log.Fatalf("hook: %s", s)
}

func Test_CalcAmountOut(t *testing.T) {
	var p entity.Pool
	poolData := `{"address":"0xabb949ef8d1e86c37e2620de318da773273460a843a623c3c05d2b85dc893e77","swapFee":8388608,"exchange":"uniswap-v4","type":"uniswap-v4","timestamp":1753702228,"reserves":["83499061346031704444327158001","4265567673400257806"],"tokens":[{"address":"0x109ddc73b46b2f8141880a5573d2b0d2acf10b07","symbol":"neged2.0","decimals":18,"swappable":true},{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true}],"extra":"{\"liquidity\":596800550298755887846132,\"sqrtPriceX96\":566274760763156353397804,\"tickSpacing\":200,\"tick\":-236988,\"ticks\":[{\"index\":-237400,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":596800550298755887846132},{\"index\":-120000,\"liquidityGross\":596800550298755887846132,\"liquidityNet\":-596800550298755887846132}],\"hX\":\"{\\\"ProtocolFee\\\":6000,\\\"ClankerFee\\\":30000,\\\"PairedFee\\\":30000,\\\"ClankerIsToken0\\\":true,\\\"ClankerTracked\\\":false}\"}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":200,\"hooks\":\"0xdd5eeaff7bd481ad55db083062b13a3cdf0a68cc\",\"uR\":\"0x6ff5693b99212da76ad316178a184ab56d299b43\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":33456440}`
	assert.NoError(t, json.Unmarshal([]byte(poolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDBase)
	assert.NoError(t, err)

	got, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x4200000000000000000000000000000000000006",
			Amount: big.NewInt(1000000000000000),
		},
		TokenOut: "0x109ddc73b46b2f8141880a5573d2b0d2acf10b07",
	})
	assert.NoError(t, err)
	assert.Equal(t, "18870367192095562543977004", got.TokenAmountOut.Amount.String())
}
