package crystalob

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const monadRouter = "0x508254c838b2e936b0631440c5c6e3ab3a4a98bd"

// ERC20 balance mapping slots on Monad (allowance is slot+1) for state-overridden swap calls.
var balanceSlots = map[string]int64{
	"0x754704bc059f8c67012fed69bc8a327a5aafb603": 9, // USDC
	"0x3bd359c1119da7da1d913d1c4d2b7c461115433a": 3, // WMON
	"0x3ebbbaa60c309ec7d857a399051a5e20eb886215": 3,
}

// TestOnChainParity tracks every live market, then compares the simulator against an eth_call of
// Crystal.swap (exact input, orderType 0, worstPrice 0, as executeCrystal calls it) at the
// tracked block. Needs an RPC that supports eth_call state overrides: CRYSTAL_RPC=<monad rpc>.
func TestOnChainParity(t *testing.T) {
	rpcURL := os.Getenv("CRYSTAL_RPC")
	if rpcURL == "" {
		t.Skip("set CRYSTAL_RPC to run")
	}
	ctx := context.Background()
	cfg := &Config{DexID: DexType, RouterAddress: monadRouter}
	client := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	lister := NewPoolsListUpdater(cfg, client)
	pools, metadataBytes, err := lister.GetNewPools(ctx, nil)
	require.NoError(t, err)
	var metadata Metadata
	require.NoError(t, json.Unmarshal(metadataBytes, &metadata))
	t.Logf("listed %d markets, %d launchpad markets pending", len(pools), len(metadata.Pending))
	for _, p := range pools { // only markets swap() accepts are listed
		var canonical common.Address
		_, err = client.NewRequest().AddCall(&ethrpc.Call{ABI: crystalABI, Target: monadRouter,
			Method: "getMarketByTokens", Params: []any{common.HexToAddress(p.Tokens[0].Address),
				common.HexToAddress(p.Tokens[1].Address)}}, []any{&canonical}).Call()
		require.NoError(t, err)
		require.Equal(t, common.HexToAddress(p.Address), canonical)
	}
	again, _, err := lister.GetNewPools(ctx, metadataBytes)
	require.NoError(t, err)
	require.Empty(t, again) // pending markets are rechecked, not re-listed
	raw, err := rpc.Dial(rpcURL)
	require.NoError(t, err)
	gc := gethclient.New(raw)
	swapABI, _ := abi.JSON(strings.NewReader(`[{"type":"function","name":"swap","inputs":[{"type":"bool"},
		{"type":"address"},{"type":"address"},{"type":"uint256"},{"type":"uint256"},{"type":"uint256"},
		{"type":"uint256"},{"type":"address"}],"outputs":[{"type":"uint256"},{"type":"uint256"},{"type":"uint256"}]}]`))
	taker := common.HexToAddress("0x00000000000000000000000000000000000c0ffe")

	tracker := NewPoolTracker(cfg, client)
	checked := 0
	for _, p := range pools {
		p, err = tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		if p.Reserves[0] == "0" && p.Reserves[1] == "0" {
			continue
		}
		sim, err := NewPoolSimulator(p)
		require.NoError(t, err)
		t.Logf("pool %s block %d: %d bid levels %v, %d ask levels %v", p.Address, p.BlockNumber,
			len(sim.bids), orderCounts(sim.bids), len(sim.asks), orderCounts(sim.asks))
		block := new(big.Int).SetUint64(p.BlockNumber)
		for in := range 2 {
			tokenIn, tokenOut := p.Tokens[in].Address, p.Tokens[1-in].Address
			slot, ok := balanceSlots[tokenIn]
			if !ok {
				continue
			}
			for e := 3; e <= 24; e += 3 {
				amt := bignumber.TenPowInt(e)
				res, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amt}, TokenOut: tokenOut})

				data, _ := swapABI.Pack("swap", true, common.HexToAddress(tokenIn), common.HexToAddress(tokenOut),
					big.NewInt(0), amt, big.NewInt(0), big.NewInt(1<<62), common.Address{})
				router := common.HexToAddress(monadRouter)
				balKey := crypto.Keccak256Hash(common.LeftPadBytes(taker[:], 32), common.BigToHash(big.NewInt(slot)).Bytes())
				allowKey := crypto.Keccak256Hash(common.LeftPadBytes(router[:], 32),
					crypto.Keccak256(common.LeftPadBytes(taker[:], 32), common.BigToHash(big.NewInt(slot+1)).Bytes()))
				funded := common.BigToHash(new(big.Int).Lsh(big.NewInt(1), 200))
				overrides := map[common.Address]gethclient.OverrideAccount{common.HexToAddress(tokenIn): {
					StateDiff: map[common.Hash]common.Hash{balKey: funded, allowKey: funded}}}
				out, callErr := gc.CallContract(ctx, ethereum.CallMsg{From: taker, To: &router, Data: data}, block, &overrides)
				if callErr != nil {
					require.Error(t, simErr, "pool %s in %s amt %s: on-chain reverted %v", p.Address, tokenIn, amt, callErr)
					continue
				}
				vals, err := swapABI.Unpack("swap", out)
				require.NoError(t, err)
				onIn, onOut := vals[0].(*big.Int), vals[1].(*big.Int)
				if onOut.Sign() == 0 {
					require.Error(t, simErr, "pool %s in %s amt %s", p.Address, tokenIn, amt)
					continue
				}
				require.NoError(t, simErr, "pool %s in %s amt %s", p.Address, tokenIn, amt)
				simIn := new(big.Int).Sub(amt, res.RemainingTokenAmountIn.Amount)
				t.Logf("pool %s in %s amt 1e%d: out %s/%s in %s/%s", p.Address, tokenIn, e,
					res.TokenAmountOut.Amount, onOut, simIn, onIn)
				require.Equal(t, onOut.String(), res.TokenAmountOut.Amount.String())
				require.Equal(t, onIn.String(), simIn.String())
			}
		}
		checked++
	}
	require.Positive(t, checked)
}

func orderCounts(levels []Level) []int {
	counts := make([]int, len(levels))
	for i, lvl := range levels {
		counts[i] = len(lvl) - 1
	}
	return counts
}
