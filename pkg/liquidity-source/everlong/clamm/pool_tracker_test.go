package everlongclamm

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func katanaTestConfig() *Config {
	return &Config{
		DexID:       DexType,
		ChainID:     valueobject.ChainIDKatana,
		PoolManager: "0xF8058204EDdfB48F5FbC7490f4F2871815C1AdB4",
		ALM:         "0xA601E3669D76cD7838C04b90dBcEdACe66454fA9",
		Router:      "0x3067A938aC493Cb15eF239e87DCC1888A596f657",
		PoolID:      "0x7d33a563f58217c160759dbb0809e73933c3189c4f1790b29f3ebc1e69555814",
		Hook:        "0x4391B5bDA5f882e38a5dD630d5054744aA94991b",
		Fee:         0,
		Parameters:  "0x00000000000000000000000000000000000000000000000000000000000a09d4",
		Currency0:   "0x0F26bBb8962d73bC891327F14dB5162D5279899F",
		Currency1:   "0x0913DA6Da4b42f538B445599b46Bb4622342Cf52",
	}
}

func katanaRPCClient() *ethrpc.Client {
	client := ethrpc.New("https://rpc.katana.network")
	client.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	return client
}

// TestLiveListTrackQuote runs the full lister -> tracker -> simulator pipeline against
// live Katana: list the pool, refresh its state through the tracker's multicall, build
// the simulator off the refreshed entity and quote both directions.
func TestLiveListTrackQuote(t *testing.T) {
	test.SkipCI(t)

	cfg := katanaTestConfig()
	client := katanaRPCClient()

	lister := NewPoolsListUpdater(cfg, client)
	pools, _, err := lister.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, "0x7d33a563f58217c160759dbb0809e73933c3189c4f1790b29f3ebc1e69555814", pools[0].Address)

	tracker := NewPoolTracker(cfg, client)
	p, err := tracker.GetNewPoolState(context.Background(), pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.NotEqual(t, "{}", p.Extra)
	require.NotZero(t, p.BlockNumber)

	p.Tokens[0].Decimals = 18
	p.Tokens[1].Decimals = 8
	sim, err := NewPoolSimulator(p, valueobject.ChainIDKatana)
	require.NoError(t, err)

	kai, wbtc := p.Tokens[0].Address, p.Tokens[1].Address
	amountKAI, _ := new(big.Int).SetString("100000000000000000000", 10) // 100 KAI
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: kai, Amount: amountKAI},
		TokenOut:      wbtc,
	})
	require.NoError(t, err)
	require.True(t, res.TokenAmountOut.Amount.Sign() > 0)
	t.Logf("block=%d 100 KAI -> %s WBTC sat (fee %s)", p.BlockNumber, res.TokenAmountOut.Amount, res.Fee.Amount)

	res, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wbtc, Amount: big.NewInt(1_000_000)}, // 0.01 WBTC
		TokenOut:      kai,
	})
	require.NoError(t, err)
	require.True(t, res.TokenAmountOut.Amount.Sign() > 0)
	t.Logf("block=%d 0.01 WBTC -> %s KAI wei (fee %s)", p.BlockNumber, res.TokenAmountOut.Amount, res.Fee.Amount)
}
