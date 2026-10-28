package pancakeprop

import (
	"context"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestLiveLister runs the lister and tracker against a node (an anvil fork at a block where the
// MM has a live price) and checks the simulator against quoteExactInput for every pair.
//
//	PANCAKE_PROP_RPC=<url> PANCAKE_PROP_IMPL=<pricing implementation> [PANCAKE_PROP_MULTICALL=<addr>] go test -run Live
func TestLiveLister(t *testing.T) {
	url, impl := os.Getenv("PANCAKE_PROP_RPC"), os.Getenv("PANCAKE_PROP_IMPL")
	if url == "" || impl == "" {
		t.Skip("PANCAKE_PROP_RPC / PANCAKE_PROP_IMPL not set")
	}
	client := ethrpc.New(url)
	if mc := os.Getenv("PANCAKE_PROP_MULTICALL"); mc != "" {
		client.SetMulticallContract(common.HexToAddress(mc))
	}
	cfg := &Config{
		DexID: DexType, Entry: "0x12B76204f5A1b390002A73e3d869196F0e6e8b1C", BlockTimeMs: 450,
		PricingImplementations: []string{impl},
	}
	ctx := context.Background()

	pools, meta, err := NewPoolsListUpdater(cfg, client).GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	again, _, err := NewPoolsListUpdater(cfg, client).GetNewPools(ctx, meta)
	require.NoError(t, err)
	assert.Empty(t, again, "listed pairs are not listed again")

	tracker := NewPoolTracker(cfg, client)
	for _, p := range pools {
		p, err = tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
		require.NoError(t, err)
		t.Logf("pool %s %s/%s block %d reserves %v", p.Address, p.Tokens[0].Address, p.Tokens[1].Address, p.BlockNumber, p.Reserves)

		blockNumber := new(big.Int).SetUint64(p.BlockNumber)
		for in := range 2 {
			room := new(uint256.Int).Sub(&sim.Extra.MaxInput[in], &sim.Extra.Cum[in])
			for _, div := range []uint64{100000, 1000, 100, 20, 10, 3, 2, 1} {
				amount := new(uint256.Int).Div(room, uint256.NewInt(div))
				if amount.IsZero() {
					continue
				}
				data := append(common.FromHex("0x21860838"), common.LeftPadBytes(common.HexToAddress(p.Tokens[in].Address).Bytes(), 32)...)
				data = append(data, common.LeftPadBytes(common.HexToAddress(p.Tokens[1-in].Address).Bytes(), 32)...)
				data = append(data, amount.PaddedBytes(32)...)
				entry := common.HexToAddress(cfg.Entry)
				raw, chainErr := client.GetETHClient().CallContract(ctx, ethereum.CallMsg{To: &entry, Data: data}, blockNumber)

				res, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: p.Tokens[in].Address, Amount: amount.ToBig()}, TokenOut: p.Tokens[1-in].Address,
				})
				name := p.Tokens[in].Address + " " + strconv.FormatUint(div, 10)
				if chainErr != nil {
					assert.Error(t, simErr, name+" chain reverts: "+chainErr.Error())
					continue
				}
				if simErr != nil {
					// The only simulator-only rejection is the vault balance, which the chain also checks.
					t.Logf("%s sim error %v although chain quoted %x", name, simErr, raw)
					assert.ErrorIs(t, simErr, pool.ErrNotEnoughInventory, name)
					continue
				}
				assert.Equal(t, new(big.Int).SetBytes(raw).String(), res.TokenAmountOut.Amount.String(), name)
			}
		}
	}
}
