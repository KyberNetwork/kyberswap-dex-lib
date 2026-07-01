package umbraedamm

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Canonical Base mainnet U1/WETH DAMM pair.
const (
	verifyPair     = "0x296964C34a571fCf85d3F74FB815ee871F5A08d4"
	multicall3Base = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

// TestVerifyAgainstChain compares the simulator's CalcAmountOut against the pair's getAmountOut.
// Reserves, fee, and every getAmountOut probe are pinned to one block (getAmountOut recomputes the
// dynamic fee live, so the snapshot only matches at the same block). Set UMBRAE_BASE_RPC_URL to run.
func TestVerifyAgainstChain(t *testing.T) {
	rpcURL := os.Getenv("UMBRAE_BASE_RPC_URL")
	if rpcURL == "" {
		t.Skip("UMBRAE_BASE_RPC_URL not set; skipping live verification")
	}
	client := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(multicall3Base))
	ctx := context.Background()

	lister := NewPoolsListUpdater(&Config{DexID: DexType, Pools: []string{verifyPair}}, client)
	pools, _, err := lister.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	tokenX := pools[0].Tokens[0].Address
	tokenY := pools[0].Tokens[1].Address

	amounts := []*big.Int{exp10(14), exp10(15), exp10(16), exp10(17), exp10(18)}
	dirs := []bool{true, false}

	var (
		reserves struct{ ReserveX, ReserveY *big.Int }
		feeBps   uint16
		feeToken common.Address
	)
	chainOuts := make([]*big.Int, len(amounts)*len(dirs))
	req := client.R().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: verifyPair, Method: pairMethodGetReserves}, []any{&reserves}).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: verifyPair, Method: pairMethodCurrentFeeBps}, []any{&feeBps}).
		AddCall(&ethrpc.Call{ABI: pairABI, Target: verifyPair, Method: pairMethodFeeToken}, []any{&feeToken})
	i := 0
	for _, xToY := range dirs {
		for _, amt := range amounts {
			req.AddCall(&ethrpc.Call{ABI: pairABI, Target: verifyPair, Method: "getAmountOut", Params: []any{amt, xToY}}, []any{&chainOuts[i]})
			i++
		}
	}
	_, err = req.TryBlockAndAggregate()
	require.NoError(t, err)
	t.Logf("DAMM same-block: reservesX=%s reservesY=%s feeBps=%d feeToken=%s", reserves.ReserveX, reserves.ReserveY, feeBps, feeToken.Hex())

	extraBytes, _ := json.Marshal(Extra{FeeBps: uint64(feeBps), FeeToken: strings.ToLower(feeToken.Hex())})
	ep := pools[0]
	ep.Reserves = entity.PoolReserves{reserves.ReserveX.String(), reserves.ReserveY.String()}
	ep.Extra = string(extraBytes)
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	var total, matched, idx int
	for _, xToY := range dirs {
		tokenIn, tokenOut := tokenX, tokenY
		if !xToY {
			tokenIn, tokenOut = tokenY, tokenX
		}
		for _, amt := range amounts {
			chainOut := chainOuts[idx]
			idx++
			var simOut *big.Int = big.NewInt(0)
			res, serr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amt}, TokenOut: tokenOut,
			})
			if serr == nil {
				simOut = res.TokenAmountOut.Amount
			}
			total++
			ok := chainOut.Cmp(simOut) == 0
			if ok {
				matched++
			}
			dir := "X->Y"
			if !xToY {
				dir = "Y->X"
			}
			t.Logf("%s in=%-22s | chain=%-24s sim=%-24s match=%v", dir, amt, chainOut, simOut, ok)
		}
	}
	t.Logf("MATCHED %d/%d", matched, total)
	require.Equal(t, total, matched, "simulator diverged from on-chain getAmountOut")
}

func exp10(n uint) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil) }
