package aegisprop

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

type liveCase struct {
	hook, pool, usdc string
	decimals1        uint8
}

var liveCases = map[string]liveCase{
	"base": {"0x09b846cadee474ee290db4bb169f156200d92088", "0x4666f34ad74415e66d72cb16029b42389bb3d44ef9f151917b38c4fb40df4a95",
		"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", 6},
	"arbitrum": {"0x0da3f5af21715a2e74adf11ac0b04d9aca842088", "0xbd642124e22d19b0b25f679509e6dcbea4ec1b8ea9be6149bfc35859a8314344",
		"0xaf88d065e77c8cc2239327c5edb3a432268e5831", 6},
}

// TestLiveQuoteMatchesChain tracks a V1 pool at a pinned block and compares the port against the
// hook's own quote() at that block, both directions, exact-in and exact-out, across level sizes.
// Run: AEGISPROP_LIVE_CHAIN=base AEGISPROP_LIVE_RPC=<url> go test -run TestLiveQuoteMatchesChain -v
func TestLiveQuoteMatchesChain(t *testing.T) {
	rpcURL, chain := os.Getenv("AEGISPROP_LIVE_RPC"), os.Getenv("AEGISPROP_LIVE_CHAIN")
	c, ok := liveCases[chain]
	if rpcURL == "" || !ok {
		t.Skip("set AEGISPROP_LIVE_RPC and AEGISPROP_LIVE_CHAIN=base|arbitrum")
	}
	ctx := context.Background()
	eth, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	latest, err := eth.BlockNumber(ctx)
	require.NoError(t, err)
	blockNum := new(big.Int).SetUint64(latest - 2)
	header, err := eth.HeaderByNumber(ctx, blockNum)
	require.NoError(t, err)

	hookAddr := common.HexToAddress(c.hook)
	staticExtra, _ := json.Marshal(uniswapv4.StaticExtra{IsNative: [2]bool{true, false}, HooksAddress: hookAddr})
	pool := &entity.Pool{Address: c.pool, StaticExtra: string(staticExtra), Tokens: []*entity.PoolToken{
		{Address: "0x4200000000000000000000000000000000000006"}, {Address: c.usdc}}}
	client := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	param := &uniswapv4.HookParam{RpcClient: client, Pool: pool, HookAddress: hookAddr, BlockNumber: blockNum, QuoteAt: int64(header.Time)}
	base, ok := uniswapv4.GetHook(hookAddr, param)
	require.True(t, ok)
	raw, err := base.Track(ctx, param)
	require.NoError(t, err)
	param.HookExtra = uniswapv4.HookExtra(raw)
	h, _ := uniswapv4.GetHook(hookAddr, param)
	hv := h.(*HookV1)
	t.Logf("block %d ts %d expiration %d bids %d asks %d inv0 %s inv1 %s fee %d", blockNum, header.Time,
		hv.expiration, len(hv.bids), len(hv.asks), hv.Inv0.Dec(), hv.Inv1.Dec(), hv.TakerFeePips)
	require.False(t, hv.Unsupported)

	key := struct {
		Currency0, Currency1 common.Address
		Fee                  *big.Int
		TickSpacing          *big.Int
		Hooks                common.Address
	}{common.Address{}, common.HexToAddress(c.usdc), big.NewInt(1), big.NewInt(1000), hookAddr}
	sizes := [2][]uint64{{1e12, 1e14, 1e15, 1e16, 5e16, 1e17, 1e18}, {1e3, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10}}

	compared, fills := 0, 0
	for _, zfo := range []bool{true, false} {
		for _, exactIn := range []bool{true, false} {
			cur := 0 // index of the currency `amount` is in
			if zfo != exactIn {
				cur = 1
			}
			for _, sz := range sizes[cur] {
				amt := new(big.Int).SetUint64(sz)
				spec := new(big.Int).Set(amt)
				if exactIn {
					spec.Neg(spec)
				}
				data, err := aegisPropHookV1ABI.Pack("quote", zfo, spec, key)
				require.NoError(t, err)
				out, err := eth.CallContract(ctx, ethereum.CallMsg{To: &hookAddr, Data: data}, blockNum)
				require.NoError(t, err)
				vals, err := aegisPropHookV1ABI.Unpack("quote", out)
				require.NoError(t, err)
				wantAmt, wantReason := vals[0].(*big.Int), vals[1].(uint8)

				f, ferr := hv.fill(zfo, exactIn, uint256.MustFromBig(amt))
				name := fmt.Sprintf("zfo=%v exactIn=%v amt=%d", zfo, exactIn, sz)
				if wantReason != 0 {
					require.ErrorIs(t, ferr, ErrYield, name)
					require.Contains(t, ferr.Error(), fmt.Sprintf("reason %d)", wantReason), name)
					t.Logf("%s yield %d", name, wantReason)
				} else {
					require.NoError(t, ferr, name)
					got := f.takerOut
					if !exactIn {
						got = f.takerIn
					}
					require.Equal(t, wantAmt.String(), got.Dec(), name)
					fills++
					t.Logf("%s amount %s (takerIn %s)", name, got.Dec(), f.takerIn.Dec())
				}
				compared++
			}
		}
	}
	require.Positive(t, fills, "no size filled; the comparison proved nothing")
	t.Logf("compared %d requests, %d fills", compared, fills)
	_ = errors.Is
}
