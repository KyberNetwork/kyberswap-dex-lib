package bunniv2

import (
	"context"
	"encoding/json"
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

	rpcClient := ethrpc.New("https://unichain.drpc.org").
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	p := &entity.Pool{
		Address: "0xeec51c6b1a9e7c4bb4fc4fa9a02fc4fff3fe94efd044f895d98b5bfbd2ff9433",
		Tokens: []*entity.PoolToken{
			{
				Address: "0x078d782b760474a361dda0af3839290b0ef57ad6",
			},
			{
				Address: "0x9151434b16b9763660705744891fa906f660ecc5",
			},
		},
		StaticExtra: "{\"tickSpacing\":1}",
	}

	cfg := &uniswapv4.Config{
		ChainID: 130,
	}

	hookExtra := ""

	h := NewHook(&uniswapv4.HookParam{
		Cfg:         cfg,
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0x005af73a245d8171a0550ffae2631f12cc211888"),
		Pool:        p,
	})

	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		Cfg:       cfg,
		RpcClient: rpcClient,
		Pool:      p,
		HookExtra: hookExtra,
	})
	require.NoError(t, err)
}

func Test_CalcAmountOut(t *testing.T) {
	var p entity.Pool
	poolData := `{"address":"0xeec51c6b1a9e7c4bb4fc4fa9a02fc4fff3fe94efd044f895d98b5bfbd2ff9433","exchange":"uniswap-v4-bunni-v2","type":"uniswap-v4","timestamp":1754722330,"reserves":["0","0"],"tokens":[{"address":"0x078d782b760474a361dda0af3839290b0ef57ad6","symbol":"USDC","decimals":6,"swappable":true},{"address":"0x9151434b16b9763660705744891fa906f660ecc5","symbol":"USD₮0","decimals":6,"swappable":true}],"extra":"{\"liquidity\":0,\"sqrtPriceX96\":79208352997136529422885942753,\"tickSpacing\":1,\"tick\":-6,\"ticks\":[],\"hX\":\"{\\\"HookletExtra\\\":\\\"\\\",\\\"HookletAddress\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"LDFAddress\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"HookFee\\\":\\\"0\\\",\\\"PoolManagerReserves\\\":[\\\"112662567266135\\\",\\\"68841382417687\\\"],\\\"LdfState\\\":[1,255,255,250,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],\\\"Vaults\\\":[{\\\"Address\\\":\\\"0x6eae95ee783e4d862867c4e0e4c3f4b95aa682ba\\\",\\\"Decimals\\\":6,\\\"RedeemRate\\\":\\\"1012692219348465292\\\",\\\"MaxDeposit\\\":\\\"79069491808604\\\"},{\\\"Address\\\":\\\"0xd49181c522ecdb265f0d9c175cf26fface64ead3\\\",\\\"Decimals\\\":6,\\\"RedeemRate\\\":\\\"1008975512755596057\\\",\\\"MaxDeposit\\\":\\\"79069491808604\\\"}],\\\"AmAmm\\\":{\\\"AmAmmManager\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"SwapFee0For1\\\":\\\"0\\\",\\\"SwapFee1For0\\\":\\\"0\\\"},\\\"ObservationState\\\":{\\\"Index\\\":36,\\\"Cardinality\\\":97,\\\"CardinalityNext\\\":97,\\\"IntermediateObservation\\\":{\\\"BlockTimestamp\\\":1754722054,\\\"PrevTick\\\":-2,\\\"TickCumulative\\\":-20895705,\\\"Initialized\\\":true}},\\\"CuratorFees\\\":{\\\"FeeRate\\\":\\\"0\\\"},\\\"Observations\\\":[{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":3953048515,\\\"PrevTick\\\":-974589,\\\"TickCumulative\\\":-9923440841788703,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":86400,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":4080116892,\\\"PrevTick\\\":-6542852,\\\"TickCumulative\\\":16647030664248144,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":1749155135,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false},{\\\"BlockTimestamp\\\":0,\\\"PrevTick\\\":0,\\\"TickCumulative\\\":0,\\\"Initialized\\\":false}],\\\"HookParams\\\":{\\\"FeeMin\\\":\\\"3\\\",\\\"FeeMax\\\":\\\"500\\\",\\\"FeeQuadraticMultiplier\\\":\\\"30000\\\",\\\"FeeTwapSecondsAgo\\\":43200,\\\"SurgeFeeHalfLife\\\":\\\"30\\\",\\\"SurgeFeeAutostartThreshold\\\":60,\\\"VaultSurgeThreshold0\\\":\\\"100\\\",\\\"VaultSurgeThreshold1\\\":\\\"100\\\",\\\"AmAmmEnabled\\\":false,\\\"OracleMinInterval\\\":1800},\\\"Slot0\\\":{\\\"SqrtPriceX96\\\":\\\"79221610121413693564263582962\\\",\\\"Tick\\\":-2,\\\"LastSwapTimestamp\\\":1754722054,\\\"LastSurgeTimestamp\\\":1754711205},\\\"BunniState\\\":{\\\"Hooklet\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"TwapSecondsAgo\\\":172800,\\\"LdfParams\\\":[0,255,255,252,0,4,1,252,147,80,29,205,101,0,0,4,17,225,163,0,29,205,101,0,59,154,202,0,0,0,0,0],\\\"HookParams\\\":\\\"AAADAAH0AHUwAKjAAAAAAB4APABkAGQACgAyBwgBLAAAAAcIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\\\",\\\"LdfType\\\":2,\\\"MinRawTokenRatio0\\\":\\\"50000\\\",\\\"TargetRawTokenRatio0\\\":\\\"100000\\\",\\\"MaxRawTokenRatio0\\\":\\\"150000\\\",\\\"MinRawTokenRatio1\\\":\\\"50000\\\",\\\"TargetRawTokenRatio1\\\":\\\"100000\\\",\\\"MaxRawTokenRatio1\\\":\\\"150000\\\",\\\"Currency0Decimals\\\":6,\\\"Currency1Decimals\\\":6,\\\"RawBalance0\\\":\\\"2252906330930\\\",\\\"RawBalance1\\\":\\\"342447385263\\\",\\\"Reserve0\\\":\\\"19043297721270\\\",\\\"Reserve1\\\":\\\"4720696956433\\\",\\\"IdleBalance\\\":[128,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,16,179,181,110,230,208]},\\\"VaultSharePrices\\\":{\\\"Initialized\\\":true,\\\"SharedPrice0\\\":\\\"1012691656053880231\\\",\\\"SharedPrice1\\\":\\\"1008975290254813319\\\"}}\"}","staticExtra":"{\"0x0\":[false,false],\"fee\":0,\"tS\":1,\"hooks\":\"0x005af73a245d8171a0550ffae2631f12cc211888\",\"uR\":\"0xef740bf23acae26f6492b10de645d6b98dc8eaf3\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":23973695}`

	assert.NoError(t, json.Unmarshal([]byte(poolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDUnichain)
	assert.NoError(t, err)

	got, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x078d782b760474a361dda0af3839290b0ef57ad6",
			Amount: big.NewInt(1000000000),
		},
		TokenOut: "0x9151434b16b9763660705744891fa906f660ecc5",
	})
	assert.NoError(t, err)
	assert.Equal(t, "18870367192095562543977004", got.TokenAmountOut.Amount.String())
}
