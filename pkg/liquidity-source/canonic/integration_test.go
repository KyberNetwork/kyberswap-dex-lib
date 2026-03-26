package canonic

import (
	"context"
	"encoding/hex"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const (
	testMAOB      = "0x23469683e25b780DFDC11410a8e83c923caDF125"
	testPreviewer = "0xEaeD40cC4bA1e7A2A7CA3f1A22C815B628B074Ea"

	defaultRPCURL    = "https://megaeth.drpc.org"
	multicallAddress = "0xcA11bde05977b3631167028862bE2a173976CA11"

	megaETHChainID = 4326
)

func getTestClient(t *testing.T) *ethrpc.Client {
	t.Helper()
	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		rpcURL = defaultRPCURL
	}
	return ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress(multicallAddress))
}

func fetchPoolAtBlock(t *testing.T, client *ethrpc.Client, blockNum *big.Int) (entity.Pool, *PoolSimulator) {
	t.Helper()
	ctx := context.Background()

	updater := NewPoolsListUpdater(&Config{
		DexId: "canonic",
		Pools: []string{testMAOB},
	}, client)

	pools, _, err := updater.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)

	tracker := &PoolTracker{
		ethrpcClient: client,
	}

	p, err := tracker.getPoolState(ctx, pools[0], blockNum)
	require.NoError(t, err)

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)

	return p, sim
}

func pinBlock(t *testing.T, client *ethrpc.Client) *big.Int {
	t.Helper()
	var rungCount *big.Int
	req := client.NewRequest().SetContext(context.Background())
	req.AddCall(&ethrpc.Call{
		ABI:    maobABI,
		Target: testMAOB,
		Method: maobMethodRungCount,
		Params: nil,
	}, []any{&rungCount})
	resp, err := req.TryBlockAndAggregate()
	require.NoError(t, err)
	require.NotNil(t, resp.BlockNumber)
	return new(big.Int).Sub(resp.BlockNumber, big.NewInt(2))
}

func decodeTakerOutput(output string) (amountOut, feePaid *big.Int) {
	output = strings.TrimPrefix(output, "0x")
	if len(output) < 128 {
		return nil, nil
	}
	amountOut, _ = new(big.Int).SetString(output[:64], 16)
	feePaid, _ = new(big.Int).SetString(output[64:128], 16)
	return amountOut, feePaid
}

func balanceSlot(slot *big.Int, account common.Address, vyper bool) string {
	var data [64]byte
	if vyper {
		slot.FillBytes(data[0:32])
		copy(data[44:64], account.Bytes())
	} else {
		copy(data[12:32], account.Bytes())
		slot.FillBytes(data[32:64])
	}
	return crypto.Keccak256Hash(data[:]).Hex()
}

func allowanceSlot(slot *big.Int, owner, spender common.Address, vyper bool) string {
	var outerData [64]byte
	if vyper {
		slot.FillBytes(outerData[0:32])
		copy(outerData[44:64], owner.Bytes())
	} else {
		copy(outerData[12:32], owner.Bytes())
		slot.FillBytes(outerData[32:64])
	}
	outerHash := crypto.Keccak256(outerData[:])

	var innerData [64]byte
	if vyper {
		copy(innerData[0:32], outerHash)
		copy(innerData[44:64], spender.Bytes())
	} else {
		copy(innerData[12:32], spender.Bytes())
		copy(innerData[32:64], outerHash)
	}
	return crypto.Keccak256Hash(innerData[:]).Hex()
}

func TestIntegration_TenderlyMultiHop(t *testing.T) {
	tc := testutil.RequireTenderly(t)
	client := getTestClient(t)

	pinnedBlock := pinBlock(t, client)
	t.Logf("Pinned block: %d", pinnedBlock.Uint64())

	p, sim := fetchPoolAtBlock(t, client, pinnedBlock)

	var staticExtra StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &staticExtra))

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	t.Logf("minQuoteTaker=%s, takerFee=%d, marketState=%d midPrice=%s midPrecision=%s rungDenom=%s priceSigfigs=%s",
		extra.MinQuoteTaker, extra.TakerFee, extra.MarketState,
		extra.MidPrice, extra.MidPrecision, extra.RungDenom, extra.PriceSigfigs)
	t.Logf("askRungs=%v askVolumes=%v", extra.AskRungs, extra.AskVolumes)
	t.Logf("bidRungs=%v bidVolumes=%v", extra.BidRungs, extra.BidVolumes)
	t.Logf("baseScale=%s quoteScale=%s", staticExtra.BaseScale, staticExtra.QuoteScale)

	baseTokenAddr := common.HexToAddress(staticExtra.BaseToken)
	quoteTokenAddr := common.HexToAddress(staticExtra.QuoteToken)
	maob := common.HexToAddress(testMAOB)
	sender := testutil.DefaultSender

	baseBal, err := tc.FindBalanceOfSlot(megaETHChainID, baseTokenAddr)
	require.NoError(t, err, "find base token balance slot")
	baseAllow, err := tc.FindAllowanceSlot(megaETHChainID, baseTokenAddr)
	require.NoError(t, err, "find base token allowance slot")
	baseSlots := &testutil.TokenStorageSlots{
		BalanceSlot: baseBal.BalanceSlot,
		AllowSlot:   baseAllow.AllowSlot,
		IsVyper:     baseBal.IsVyper,
	}
	t.Logf("Base token slots: balance=%s allowance=%s vyper=%v",
		baseSlots.BalanceSlot, baseSlots.AllowSlot, baseSlots.IsVyper)

	quoteBal, err := tc.FindBalanceOfSlot(megaETHChainID, quoteTokenAddr)
	require.NoError(t, err, "find quote token balance slot")
	quoteAllow, err := tc.FindAllowanceSlot(megaETHChainID, quoteTokenAddr)
	require.NoError(t, err, "find quote token allowance slot")
	quoteSlots := &testutil.TokenStorageSlots{
		BalanceSlot: quoteBal.BalanceSlot,
		AllowSlot:   quoteAllow.AllowSlot,
		IsVyper:     quoteBal.IsVyper,
	}
	t.Logf("Quote token slots: balance=%s allowance=%s vyper=%v",
		quoteSlots.BalanceSlot, quoteSlots.AllowSlot, quoteSlots.IsVyper)

	minQT := uint256.MustFromDecimal(extra.MinQuoteTaker).ToBig()
	hop1Amount := new(big.Int).Mul(minQT, big.NewInt(2))

	type hopDef struct {
		tokenIn   string
		tokenOut  string
		amountIn  *big.Int
		label     string
		isBuyBase bool
	}

	hops := []hopDef{
		{tokenIn: staticExtra.QuoteToken, tokenOut: staticExtra.BaseToken, amountIn: hop1Amount, label: "hop1: buy base", isBuyBase: true},
	}

	type simOutput struct {
		result *pool.CalcAmountOutResult
	}
	simOutputs := make([]simOutput, 0, 3)

	prevOut := hop1Amount
	for i := range 3 {
		if i > 0 {
			if i%2 == 1 {
				hops = append(hops, hopDef{
					tokenIn: staticExtra.BaseToken, tokenOut: staticExtra.QuoteToken,
					amountIn: prevOut, label: "hop2: sell base", isBuyBase: false,
				})
			} else {
				hops = append(hops, hopDef{
					tokenIn: staticExtra.QuoteToken, tokenOut: staticExtra.BaseToken,
					amountIn: prevOut, label: "hop3: buy base", isBuyBase: true,
				})
			}
		}

		h := hops[i]
		t.Logf("%s: amountIn=%s", h.label, h.amountIn)

		simResult, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: h.tokenIn, Amount: h.amountIn},
			TokenOut:      h.tokenOut,
		})
		require.NoError(t, simErr, "%s: sim failed", h.label)
		require.True(t, simResult.TokenAmountOut.Amount.Sign() > 0, "%s: expected positive output", h.label)

		t.Logf("%s: sim out=%s fee=%s", h.label, simResult.TokenAmountOut.Amount, simResult.Fee.Amount)
		simOutputs = append(simOutputs, simOutput{result: simResult})

		sim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: h.tokenIn, Amount: h.amountIn},
			TokenAmountOut: *simResult.TokenAmountOut,
			Fee:            *simResult.Fee,
			SwapInfo:       simResult.SwapInfo,
		})

		prevOut = simResult.TokenAmountOut.Amount
	}

	quoteBalSlot := balanceSlot(quoteSlots.BalanceSlot, sender, quoteSlots.IsVyper)
	quoteAllowSlot := allowanceSlot(quoteSlots.AllowSlot, sender, maob, quoteSlots.IsVyper)

	approveCalldata, err := testutil.EncodeSwapCalldata("approve(address,uint256)", maob, testutil.MaxUint256())
	require.NoError(t, err, "encode approve calldata")

	baseTx := func() testutil.SimulateRequest {
		return testutil.SimulateRequest{
			NetworkID: "4326", From: sender.Hex(), To: maob.Hex(),
			Gas: 1_000_000, GasPrice: "2000000000",
			Save: true, SaveIfFails: true,
			BlockNumber: pinnedBlock.Uint64() + 1,
		}
	}

	bundleReqs := make([]testutil.SimulateRequest, 0, 7)

	for i, h := range hops {
		if i > 0 {
			approveReq := baseTx()
			if h.isBuyBase {
				approveReq.To = quoteTokenAddr.Hex()
			} else {
				approveReq.To = baseTokenAddr.Hex()
			}
			approveReq.Input = "0x" + hex.EncodeToString(approveCalldata)
			bundleReqs = append(bundleReqs, approveReq)
		}

		var calldata []byte
		if h.isBuyBase {
			calldata, err = testutil.EncodeSwapCalldata(
				"buyBaseTargetIn(uint256,uint256,uint64,uint256)",
				h.amountIn, big.NewInt(0), uint64(99999999999), big.NewInt(0),
			)
		} else {
			calldata, err = testutil.EncodeSwapCalldata(
				"sellBaseTargetIn(uint256,uint256,uint64,uint256)",
				h.amountIn, big.NewInt(0), uint64(99999999999), big.NewInt(0),
			)
		}
		require.NoError(t, err, "%s: encode calldata", h.label)

		req := baseTx()
		req.Input = "0x" + hex.EncodeToString(calldata)

		if i == 0 {
			req.StateObjects = testutil.StateOverride{
				strings.ToLower(sender.Hex()): {Balance: "1000000000000000000000"},
				strings.ToLower(quoteTokenAddr.Hex()): {
					Storage: map[string]string{
						quoteBalSlot:   common.BigToHash(hop1Amount).Hex(),
						quoteAllowSlot: common.BigToHash(testutil.MaxUint256()).Hex(),
					},
				},
			}
		}

		bundleReqs = append(bundleReqs, req)
	}

	t.Logf("Sending %d-tx bundle to Tenderly (%d swaps + %d approves)...",
		len(bundleReqs), len(hops), len(bundleReqs)-len(hops))
	bundleResults, bundleErr := tc.SimulateBundle(bundleReqs)
	require.NoError(t, bundleErr, "tenderly simulate-bundle")
	require.Len(t, bundleResults, len(bundleReqs), "expected %d bundle results", len(bundleReqs))

	swapIndices := []int{0}
	for i := 1; i < len(hops); i++ {
		swapIndices = append(swapIndices, i*2-1+1)
	}

	for i, h := range hops {
		bundleIdx := swapIndices[i]
		res := bundleResults[bundleIdx]
		simOut := simOutputs[i]

		simURL := tc.SimulationURL(res.Simulation.ID)
		t.Logf("%s: tenderly URL: %s", h.label, simURL)

		require.True(t, res.Transaction.Status,
			"%s: on-chain tx reverted — check %s", h.label, simURL)

		actualOut, actualFee := decodeTakerOutput(res.Transaction.TransactionInfo.CallTrace.Output)
		require.NotNil(t, actualOut, "%s: expected non-nil output", h.label)

		t.Logf("%s: on-chain out=%s fee=%s", h.label, actualOut, actualFee)

		outDiff := new(big.Int).Sub(simOut.result.TokenAmountOut.Amount, actualOut)
		feeDiff := new(big.Int).Sub(simOut.result.Fee.Amount, actualFee)
		t.Logf("%s: outDiff=%s feeDiff=%s", h.label, outDiff, feeDiff)

		require.Equal(t, simOut.result.TokenAmountOut.Amount.String(), actualOut.String(),
			"%s: output MISMATCH", h.label)
		require.Equal(t, simOut.result.Fee.Amount.String(), actualFee.String(),
			"%s: fee MISMATCH", h.label)

		t.Logf("%s: EXACT MATCH ✓", h.label)
	}
}
