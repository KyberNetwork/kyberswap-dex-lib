package liquidityparty

import (
	"context"
	"fmt"
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

// Mainnet deployment (chain 1).
var mainnetConfig = &Config{
	DexID:               DexType,
	PartyPlannerAddress: "0x5E9DB9fa66aeA7f254d4A6783b1a6180C4B8AAe3",
	PartyInfoAddress:    "0xefF3Ed388D3887e7C9F375B7f1ad8A0B77C05643",
}

const testPoolAddress = "0x1270Da05Cf1d047763CEEfDe25a4a5438b26fdA6" // live 3-token pool

func newTestClient(t *testing.T) *ethrpc.Client {
	rpcURL := os.Getenv("ETHEREUM_RPC_URL")
	if rpcURL == "" {
		t.Skip("ETHEREUM_RPC_URL not set")
	}
	return ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
}

func TestIntegration_PoolListUpdater(t *testing.T) {
	client := newTestClient(t)
	u := NewPoolsListUpdater(mainnetConfig, client)

	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(pools), 0, "should discover at least one pool")

	found := false
	for i := range pools {
		p := &pools[i]
		require.Equal(t, DexType, p.Type)
		require.Equal(t, strings.ToLower(p.Address), p.Address, "pool address must be lowercased")
		require.Equal(t, len(p.Tokens), len(p.Reserves))
		require.Greater(t, len(p.Tokens), 0)
		for _, tok := range p.Tokens {
			require.True(t, tok.Swappable)
			require.Equal(t, strings.ToLower(tok.Address), tok.Address, "token address must be lowercased")
		}
		if p.Address == strings.ToLower(testPoolAddress) {
			found = true
		}
	}
	t.Logf("discovered %d pools", len(pools))
	require.True(t, found, "expected to discover the known test pool %s", testPoolAddress)
}

func TestIntegration_PoolTracker(t *testing.T) {
	client := newTestClient(t)
	tracker, err := NewPoolTracker(mainnetConfig, client)
	require.NoError(t, err)

	// A minimal pool entity as the lister would produce it (3 tokens, zero reserves).
	p := entity.Pool{
		Address: common.HexToAddress(testPoolAddress).Hex(),
		Type:    DexType,
		Tokens: []*entity.PoolToken{
			{Address: "0x0000000000000000000000000000000000000000", Swappable: true},
			{Address: "0x0000000000000000000000000000000000000001", Swappable: true},
			{Address: "0x0000000000000000000000000000000000000002", Swappable: true},
		},
		Reserves: entity.PoolReserves{"0", "0", "0"},
	}

	updated, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	require.Greater(t, updated.BlockNumber, uint64(0))
	require.Len(t, updated.Reserves, 3)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
	require.NotNil(t, extra.Kappa)
	require.NotNil(t, extra.EffectiveSigmaQ)
	require.Len(t, extra.QInternal, 3)
	require.Len(t, extra.Bases, 3)
	require.Len(t, extra.FeesPpm, 3)

	t.Logf("block=%d killed=%v kappa=%s effSigmaQ=%s", updated.BlockNumber, extra.Killed,
		extra.Kappa.String(), extra.EffectiveSigmaQ.String())
	t.Logf("reserves=%v", updated.Reserves)
	t.Logf("qInternal=%v bases=%v feesPpm=%v", extra.QInternal, extra.Bases, extra.FeesPpm)
}

// buildSimulatorFromChain fetches the live pool snapshot pinned to a block, marshals it the way the
// tracker does, and constructs a PoolSimulator with synthetic (index-aligned) token addresses.
func buildSimulatorFromChain(t *testing.T, client *ethrpc.Client) (*PoolSimulator, *PoolStateSnapshotRPC, uint64) {
	t.Helper()
	tracker, err := NewPoolTracker(mainnetConfig, client)
	require.NoError(t, err)

	snapshot, killed, blockNumber, err := tracker.fetchPoolState(context.Background(), testPoolAddress, 0, true, nil)
	require.NoError(t, err)
	require.NoError(t, validateSnapshot(snapshot, len(snapshot.QInternal)))
	require.Greater(t, blockNumber, uint64(0))

	n := len(snapshot.QInternal)
	extra := Extra{
		Kappa:           snapshot.Kappa,
		EffectiveSigmaQ: snapshot.EffectiveSigmaQ,
		QInternal:       snapshot.QInternal,
		Bases:           snapshot.Bases,
		FeesPpm:         make([]uint64, n),
		Killed:          killed,
	}
	for i, f := range snapshot.FeesPpm {
		extra.FeesPpm[i] = f.Uint64()
	}
	extraBytes, err := json.Marshal(&extra)
	require.NoError(t, err)

	tokens := make([]*entity.PoolToken, n)
	reserves := make(entity.PoolReserves, n)
	for i := 0; i < n; i++ {
		tokens[i] = &entity.PoolToken{Address: fmt.Sprintf("0x%040x", i+1), Swappable: true}
		reserves[i] = snapshot.CachedBalances[i].String()
	}

	sim, err := NewPoolSimulator(entity.Pool{
		Address:     strings.ToLower(testPoolAddress),
		Exchange:    DexType,
		Type:        DexType,
		Tokens:      tokens,
		Reserves:    reserves,
		Extra:       string(extraBytes),
		BlockNumber: blockNumber,
	})
	require.NoError(t, err)
	return sim, snapshot, blockNumber
}

// ethrpc unpacks only into Output[0], so multi-return view methods decode into a struct whose fields
// mirror the return values (by name/order); scalar pointers would fail with "cannot unmarshal tuple".
type swapAmountsRPC struct {
	AmountIn  *big.Int
	AmountOut *big.Int
	OutFee    *big.Int
}

type swapAmountsExactOutRPC struct {
	AmountIn *big.Int
	OutFee   *big.Int
}

// rpcSwapAmounts calls PartyInfo.swapAmounts (exact-in) at a pinned block.
func rpcSwapAmounts(t *testing.T, client *ethrpc.Client, block uint64, i, j int, amountIn *big.Int) (amountOut, outFee *big.Int, revert bool) {
	t.Helper()
	var out swapAmountsRPC
	_, err := client.NewRequest().SetContext(context.Background()).
		SetBlockNumber(new(big.Int).SetUint64(block)).
		AddCall(&ethrpc.Call{
			ABI:    partyInfoABI,
			Target: mainnetConfig.PartyInfoAddress,
			Method: infoMethodSwapAmounts,
			Params: []any{common.HexToAddress(testPoolAddress), big.NewInt(int64(i)), big.NewInt(int64(j)), amountIn},
		}, []any{&out}).Call()
	if err != nil {
		return nil, nil, true
	}
	return out.AmountOut, out.OutFee, false
}

// rpcSwapAmountsForExactOutput calls PartyInfo.swapAmountsForExactOutput at a pinned block.
func rpcSwapAmountsForExactOutput(t *testing.T, client *ethrpc.Client, block uint64, i, j int, amountOut *big.Int) (amountIn, outFee *big.Int, revert bool) {
	t.Helper()
	var out swapAmountsExactOutRPC
	_, err := client.NewRequest().SetContext(context.Background()).
		SetBlockNumber(new(big.Int).SetUint64(block)).
		AddCall(&ethrpc.Call{
			ABI:    partyInfoABI,
			Target: mainnetConfig.PartyInfoAddress,
			Method: infoMethodSwapAmountsForExactOutput,
			Params: []any{common.HexToAddress(testPoolAddress), big.NewInt(int64(i)), big.NewInt(int64(j)), amountOut},
		}, []any{&out}).Call()
	if err != nil {
		return nil, nil, true
	}
	return out.AmountIn, out.OutFee, false
}

// TestIntegration_Simulator_ExactIn_WeiExact verifies CalcAmountOut matches PartyInfo.swapAmounts to
// the wei across every ordered pair and a range of sizes.
func TestIntegration_Simulator_ExactIn_WeiExact(t *testing.T) {
	client := newTestClient(t)
	sim, snapshot, block := buildSimulatorFromChain(t, client)
	n := len(snapshot.QInternal)

	// Fractions of the input reserve, spanning tiny (rounding/too-small) to large (capacity).
	fracsPpm := []int64{1, 10, 100, 1_000, 10_000, 100_000, 500_000, 1_000_000, 2_000_000}

	checked := 0
	for i := 0; i < n; i++ {
		reserveI := snapshot.CachedBalances[i]
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			for _, fr := range fracsPpm {
				amountIn := new(big.Int).Div(new(big.Int).Mul(reserveI, big.NewInt(fr)), big.NewInt(1_000_000))
				if amountIn.Sign() == 0 {
					amountIn = big.NewInt(1)
				}

				rpcOut, rpcFee, revert := rpcSwapAmounts(t, client, block, i, j, amountIn)

				res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[i], Amount: amountIn},
					TokenOut:      sim.Info.Tokens[j],
				})

				switch {
				case revert && err != nil:
					// Both reject — the swap is infeasible. Consistent.
				case revert && err == nil:
					require.Fail(t, "simulator quoted a swap the view reverts",
						"pair (%d->%d) amt=%s: sim out=%s", i, j, amountIn, res.TokenAmountOut.Amount)
				case !revert && err != nil:
					// swapAmounts is an optimistic view: it can return the drain-capped output where
					// the real swap() reverts in applySwap ("pool drained"). The simulator enforces
					// that drain guard so it never routes a reverting edge — the only allowed
					// divergence, and only at capacity (view output exceeds half the output reserve).
					require.ErrorIs(t, err, ErrTooLarge,
						"pair (%d->%d) amt=%s: unexpected simulator rejection of a feasible swap", i, j, amountIn)
					half := new(big.Int).Div(snapshot.CachedBalances[j], big.NewInt(2))
					require.Positive(t, rpcOut.Cmp(half),
						"pair (%d->%d) amt=%s: simulator rejected a non-drain swap (view out=%s, half reserve=%s)",
						i, j, amountIn, rpcOut, half)
				default:
					require.Equal(t, 0, res.TokenAmountOut.Amount.Cmp(rpcOut),
						"amountOut mismatch pair (%d->%d) amt=%s: sim=%s chain=%s",
						i, j, amountIn, res.TokenAmountOut.Amount, rpcOut)
					require.Equal(t, 0, res.Fee.Amount.Cmp(rpcFee),
						"outFee mismatch pair (%d->%d) amt=%s: sim=%s chain=%s",
						i, j, amountIn, res.Fee.Amount, rpcFee)
					checked++
				}
			}
		}
	}
	t.Logf("exact-in wei-exact: %d quotes matched at block %d", checked, block)
	require.Greater(t, checked, 0)
}

// TestIntegration_Simulator_ExactOut_WeiExact verifies CalcAmountIn matches
// PartyInfo.swapAmountsForExactOutput to the wei (test layer 2, exact-out direction).
func TestIntegration_Simulator_ExactOut_WeiExact(t *testing.T) {
	client := newTestClient(t)
	sim, snapshot, block := buildSimulatorFromChain(t, client)
	n := len(snapshot.QInternal)

	fracsPpm := []int64{10, 100, 1_000, 10_000, 100_000, 300_000}

	checked := 0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			reserveJ := snapshot.CachedBalances[j]
			for _, fr := range fracsPpm {
				amountOut := new(big.Int).Div(new(big.Int).Mul(reserveJ, big.NewInt(fr)), big.NewInt(1_000_000))
				if amountOut.Sign() == 0 {
					continue
				}

				rpcIn, rpcFee, revert := rpcSwapAmountsForExactOutput(t, client, block, i, j, amountOut)

				res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: sim.Info.Tokens[j], Amount: amountOut},
					TokenIn:        sim.Info.Tokens[i],
				})

				if revert {
					require.Error(t, err, "pair (%d->%d) out=%s: view reverted, simulator must reject", i, j, amountOut)
					continue
				}
				require.NoError(t, err, "pair (%d->%d) out=%s: view ok but simulator errored", i, j, amountOut)
				require.Equal(t, 0, res.TokenAmountIn.Amount.Cmp(rpcIn),
					"amountIn mismatch pair (%d->%d) out=%s: sim=%s chain=%s",
					i, j, amountOut, res.TokenAmountIn.Amount, rpcIn)
				require.Equal(t, 0, res.Fee.Amount.Cmp(rpcFee),
					"outFee mismatch pair (%d->%d) out=%s: sim=%s chain=%s",
					i, j, amountOut, res.Fee.Amount, rpcFee)
				checked++
			}
		}
	}
	t.Logf("exact-out wei-exact: %d quotes matched at block %d", checked, block)
	require.Greater(t, checked, 0)
}
