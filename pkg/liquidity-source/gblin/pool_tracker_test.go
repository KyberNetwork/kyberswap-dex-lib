package gblin

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var baseConfig = Config{
	DexID:            DexType,
	ChainID:          valueobject.ChainIDBase,
	Vault:            "0xc2181d975c05c8c724b334bcED0764c0b86B1D53",
	Lens:             "0xfCFea8027019E8551A1f09AD91532471F5D26f61",
	MulticallAddress: "0xcA11bde05977b3631167028862bE2a173976CA11",
}

// TestLiveBase lists and tracks the vault on Base, then checks that the simulator's quotes equal
// GBLINLens.quoteBuy at the block the state was read at, to the wei.
func TestLiveBase(t *testing.T) {
	rpcURL := os.Getenv("GBLIN_RPC")
	if rpcURL == "" {
		t.Skip("set GBLIN_RPC to a Base RPC URL to run against the live vault")
	}
	ctx := context.Background()
	client := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(baseConfig.MulticallAddress))

	pools, _, err := NewPoolsListUpdater(&baseConfig, client).GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	p := pools[0]
	assert.Equal(t, valueobject.WrappedNativeMap[valueobject.ChainIDBase], p.Tokens[0].Address)
	assert.Equal(t, "0xc2181d975c05c8c724b334bced0764c0b86b1d53", p.Tokens[1].Address)

	p, err = NewPoolTracker(&baseConfig, client).GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.NotZero(t, p.BlockNumber)
	for i := range p.Tokens {
		p.Tokens[i].Decimals = 18
	}

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	require.True(t, sim.extra.NavReliable, "vault NAV not reliable at this block")
	require.True(t, sim.extra.SequencerUp, "sequencer reported down at this block")

	for _, s := range []string{"1000000000000000", "10000000000000000", "100000000000000000", "1000000000000000000",
		"5000000000000000000"} {
		amountIn, _ := new(big.Int).SetString(s, 10)

		var lensOut struct {
			Out          *big.Int
			ProtocolFee  *big.Int
			StabilityFee *big.Int
		}
		_, err := client.NewRequest().SetContext(ctx).SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber)).
			AddCall(&ethrpc.Call{ABI: lensABI, Target: baseConfig.Lens, Method: "quoteBuy",
				Params: []any{common.HexToAddress(baseConfig.Vault), amountIn}}, []any{&lensOut}).Call()
		require.NoError(t, err)

		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: amountIn},
			TokenOut:      p.Tokens[1].Address,
		})
		require.NoError(t, err)
		assert.Equal(t, lensOut.Out.String(), res.TokenAmountOut.Amount.String(), "amountIn %s", s)
		t.Logf("block %d amountIn %s -> %s GBLIN (lens %s)", p.BlockNumber, s, res.TokenAmountOut.Amount,
			lensOut.Out)
	}
}
