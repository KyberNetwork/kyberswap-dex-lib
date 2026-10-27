package umbraedlmm

import (
	"context"
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Robinhood Chain (4663) deployment, 2026-10-15: SwapRevisionV2Oct2026.
const (
	rhFactory = "0xea3a13d8734b53A322181885E8c6C3D82FFb3A02"
	rhViewer  = "0x31F07C8b11Fa846c8a5c9aDB7dAf64F8D1F488D8"
	rhRouter  = "0xffD908D8C4c02c914b6DEA0a75425ce913a165D1"
	// Multicall3 is at the canonical address on Robinhood Chain.
	rhMulticall3 = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

// The pairs verified: 18/6 and 18/18 decimals, bin steps 10 to 50, tokenX = WETH on all of them
// (WETH sorts first on Robinhood Chain).
var rhPairs = []string{
	"0x5A914ea3a33ca267af913D821E8ABe45aA1901a0", // WETH/USDG   binStep 20
	"0xa10E99195f56C64b11Ef9bA6b67697FFC880b045", // WETH/USDG   binStep 10
	"0xEdd3aC7a612cc738Cd7f63D0819f4Db84F6bf12D", // WETH/VIRTUAL binStep 25
	"0xEd5C9f2143b5ab25287fC3B12304a3aA8fb13F33", // WETH/DELTA  binStep 50
}

// Pair reverts the router surfaces, by selector, and the simulator error each one must map to.
var rhRevertMap = map[string]error{
	selector("LBPair__InsufficientLiquidityForSwap()"): ErrInsufficientLiquidity,
	selector("LBPair__SwapBinDistanceExceeded()"):      ErrSwapMovementExceeded,
	selector("LBPair__SwapTraversalLimitExceeded()"):   ErrSwapTraversalLimit,
	selector("LBPair__InsufficientAmountOut()"):        ErrInsufficientOutput,
}

func selector(sig string) string { return hexutil.Encode(crypto.Keccak256([]byte(sig))[:4]) }

// TestVerifyRobinhood checks the SwapRevisionV2Oct2026 simulator against EXECUTION: for each pair
// and a spread of amounts in both directions (plus the amounts at which the first bin just empties,
// where the dust rule decides), the real tracker -> simulator output must equal what
// router.swapExactTokensForTokens returns when eth_call'd by a funded trader at the tracked block.
// When execution reverts, the simulator must reject with the matching error.
//
// Needs a node that serves eth_call from an address holding the input tokens with the router
// approved, so in practice an anvil fork of Robinhood Chain with a funded trader:
//
//	UMBRAE_ROBINHOOD_RPC_URL=http://127.0.0.1:8545 UMBRAE_ROBINHOOD_TRADER=0x... go test -run TestVerifyRobinhood -v
func TestVerifyRobinhood(t *testing.T) {
	rpcURL, trader := os.Getenv("UMBRAE_ROBINHOOD_RPC_URL"), os.Getenv("UMBRAE_ROBINHOOD_TRADER")
	if rpcURL == "" || trader == "" {
		t.Skip("UMBRAE_ROBINHOOD_RPC_URL / UMBRAE_ROBINHOOD_TRADER not set; skipping Robinhood execution verification")
	}
	ctx := context.Background()
	client := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(rhMulticall3))
	ec, err := ethclient.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	defer ec.Close()

	var exact, rejected, dustCases, baseWouldDiffer int
	rejectedBy := map[string]int{}
	for _, pair := range rhPairs {
		target := liveTarget{pair: pair, factory: rhFactory, viewer: rhViewer, router: rhRouter, revision: SwapRevisionV2Oct2026}
		sim, tokenX, tokenY, block := buildSimulatorFor(t, ctx, client, rpcURL, target)
		baseSim := cloneAsRevision(t, sim, SwapRevisionV2)
		t.Logf("%s: tracked at block %s, active bin %d, %d bins", pair, block, sim.activeID, len(sim.bins))

		for _, swapForY := range []bool{true, false} {
			tokenIn, tokenOut := tokenX, tokenY
			if !swapForY {
				tokenIn, tokenOut = tokenY, tokenX
			}
			amounts := spreadFor(t, ctx, ec, tokenIn)
			threshold := firstBinDrainAmount(sim, tokenIn, tokenOut)
			if threshold != nil {
				for d := int64(-2); d <= 3; d++ {
					amounts = append(amounts, new(big.Int).Add(threshold, big.NewInt(d)))
				}
			}
			for _, amt := range amounts {
				if amt.Sign() <= 0 {
					continue
				}
				params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amt}, TokenOut: tokenOut}
				res, simErr := sim.CloneState().CalcAmountOut(params)
				baseRes, baseErr := baseSim.CloneState().CalcAmountOut(params)
				chainOut, revert := executeSwap(t, ctx, ec, block, trader, pair, tokenIn, tokenOut, amt)

				label := pair[:10] + " " + map[bool]string{true: "X->Y", false: "Y->X"}[swapForY] + " in=" + amt.String()
				if revert != "" {
					want, known := rhRevertMap[revert]
					require.True(t, known, "%s: execution reverted with unmapped selector %s", label, revert)
					require.ErrorIs(t, simErr, want, "%s: execution reverted %v", label, want)
					rejected++
					rejectedBy[want.Error()]++
					if baseErr == nil || !errors.Is(baseErr, want) {
						baseWouldDiffer++
					}
					continue
				}
				require.NoError(t, simErr, "%s: execution filled %s", label, chainOut)
				require.Equal(t, chainOut.String(), res.TokenAmountOut.Amount.String(), "%s: sim must equal execution", label)
				exact++
				if baseErr != nil || baseRes.TokenAmountOut.Amount.Cmp(chainOut) != 0 {
					baseWouldDiffer++
					if threshold != nil && new(big.Int).Sub(amt, threshold).CmpAbs(big.NewInt(3)) <= 0 {
						dustCases++
					}
				}
			}
		}
	}
	t.Logf("execution-exact fills: %d, matching rejections: %d %v; the Base revision would differ on %d of them (%d at a bin-drain threshold)",
		exact, rejected, rejectedBy, baseWouldDiffer, dustCases)
}

// cloneAsRevision rebuilds a simulator's state under another revision (sensitivity check only).
func cloneAsRevision(t *testing.T, s *PoolSimulator, rev uint8) *PoolSimulator {
	t.Helper()
	c := s.CloneState().(*PoolSimulator)
	c.revision = rev
	return c
}

// spreadFor is ten amounts from 1e-6 to 1e3 whole tokens, not all round numbers.
func spreadFor(t *testing.T, ctx context.Context, ec *ethclient.Client, token string) []*big.Int {
	t.Helper()
	dec := erc20Decimals(t, ctx, ec, token)
	unit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(dec)), nil)
	var out []*big.Int
	for _, f := range []string{"0.000001", "0.0000137", "0.0001", "0.00073", "0.001", "0.0093", "0.05", "0.37", "2.9", "1000"} {
		r, _ := new(big.Rat).SetString(f)
		r.Mul(r, new(big.Rat).SetInt(unit))
		out = append(out, new(big.Int).Quo(r.Num(), r.Denom()))
	}
	return out
}

func erc20Decimals(t *testing.T, ctx context.Context, ec *ethclient.Client, token string) uint8 {
	t.Helper()
	to := common.HexToAddress(token)
	b, err := ec.CallContract(ctx, ethereum.CallMsg{To: &to, Data: crypto.Keccak256([]byte("decimals()"))[:4]}, nil)
	require.NoError(t, err)
	return uint8(new(big.Int).SetBytes(b).Uint64())
}

// firstBinDrainAmount is the smallest input whose swap leaves the active bin (binary search on the
// simulator): just past it the first bin empties with almost nothing left over, which is where the
// dust rule decides the outcome.
func firstBinDrainAmount(sim *PoolSimulator, tokenIn, tokenOut string) *big.Int {
	moves := func(a *big.Int) bool {
		res, err := sim.CloneState().CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: a}, TokenOut: tokenOut})
		if err != nil {
			return true
		}
		return res.SwapInfo.(SwapInfo).newActiveID != sim.activeID
	}
	lo, hi := big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
	if !moves(hi) {
		return nil
	}
	for new(big.Int).Sub(hi, lo).Cmp(big.NewInt(1)) > 0 {
		mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)
		if moves(mid) {
			hi = mid
		} else {
			lo = mid
		}
	}
	return hi
}

// executeSwap eth_calls router.swapExactTokensForTokens from the trader at block, returning the
// output, or the 4-byte revert selector when it reverts.
func executeSwap(t *testing.T, ctx context.Context, ec *ethclient.Client, block *big.Int, trader, pair, tokenIn, tokenOut string, amountIn *big.Int) (*big.Int, string) {
	t.Helper()
	data, err := RouterABI.Pack("swapExactTokensForTokens", amountIn, big.NewInt(0),
		[]common.Address{common.HexToAddress(tokenIn), common.HexToAddress(tokenOut)},
		[]uint16{binStepOf(t, ctx, ec, pair)}, []common.Address{common.HexToAddress(pair)},
		common.HexToAddress(trader), big.NewInt(time.Now().Add(time.Hour).Unix()))
	require.NoError(t, err)
	router := common.HexToAddress(rhRouter)
	out, err := ec.CallContract(ctx, ethereum.CallMsg{From: common.HexToAddress(trader), To: &router, Data: data}, block)
	if err != nil {
		var de rpc.DataError
		require.True(t, errors.As(err, &de), "eth_call failed without revert data: %v", err)
		hexData, _ := de.ErrorData().(string)
		require.GreaterOrEqual(t, len(hexData), 10, "revert without a selector: %v", err)
		return nil, strings.ToLower(hexData[:10])
	}
	return new(big.Int).SetBytes(out), ""
}

func binStepOf(t *testing.T, ctx context.Context, ec *ethclient.Client, pair string) uint16 {
	t.Helper()
	data, err := pairABI.Pack(pairMethodBinStep)
	require.NoError(t, err)
	to := common.HexToAddress(pair)
	b, err := ec.CallContract(ctx, ethereum.CallMsg{To: &to, Data: data}, nil)
	require.NoError(t, err)
	return uint16(new(big.Int).SetBytes(b).Uint64())
}

// TestVerifyListerRobinhood runs the real lister against the Robinhood factory: every pair is
// listed with the deployment's revision persisted. Set UMBRAE_ROBINHOOD_RPC_URL to run.
func TestVerifyListerRobinhood(t *testing.T) {
	rpcURL := os.Getenv("UMBRAE_ROBINHOOD_RPC_URL")
	if rpcURL == "" {
		t.Skip("UMBRAE_ROBINHOOD_RPC_URL not set; skipping Robinhood lister verification")
	}
	client := ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(rhMulticall3))
	lister := NewPoolsListUpdater(&Config{
		DexID: DexType, FactoryAddress: rhFactory, ViewerAddress: rhViewer, RouterAddress: rhRouter,
		SwapRevision: SwapRevisionV2Oct2026,
	}, client)
	pools, _, err := lister.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	byAddr := map[string]entity.Pool{}
	for _, p := range pools {
		var se StaticExtra
		require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
		require.Equal(t, SwapRevisionV2Oct2026, se.SwapRevision, "revision persisted for %s", p.Address)
		require.Equal(t, rhRouter, se.Router)
		byAddr[strings.ToLower(p.Address)] = p
	}
	for _, pair := range rhPairs {
		_, ok := byAddr[strings.ToLower(pair)]
		require.True(t, ok, "%s not listed", pair)
	}
	t.Logf("lister listed %d Robinhood pair(s), all with swapRevision %d", len(pools), SwapRevisionV2Oct2026)
}
