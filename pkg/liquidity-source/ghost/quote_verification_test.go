package ghost

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	pool_pkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	rpcURL     = "https://ethereum-rpc.publicnode.com"
	multicall3 = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

type onChainQuote struct {
	Token  common.Address
	Amount *big.Int
}

func quoteOnChain(t *testing.T, client *ethrpc.Client, sourceRouter string, principal *big.Int, targetRouterBytes32 common.Hash) *big.Int {
	t.Helper()

	var quotes []onChainQuote
	_, err := client.NewRequest().AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: sourceRouter,
		Method: "quoteTransferRemoteTo",
		Params: []any{
			uint32(1),
			wildcardRecipient,
			principal,
			targetRouterBytes32,
		},
	}, []any{&quotes}).Call()
	require.NoError(t, err, "quoteTransferRemoteTo failed for principal=%s", principal)
	require.GreaterOrEqual(t, len(quotes), 2)

	return quotes[1].Amount
}

func TestInverseFee_MatchesOnChainQuote(t *testing.T) {
	client := ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress(multicall3))

	// Use the pool tracker to fetch fee params — same as production.
	tracker := NewPoolTracker(&Config{DexID: DexType}, client)

	poolData := ethereumPoolData
	var items []PoolItem
	require.NoError(t, json.Unmarshal(poolData, &items))
	require.NotEmpty(t, items)

	item := items[0]
	pool := entity.Pool{
		Address:  item.ID,
		Exchange: DexType,
		Type:     item.Type,
		Tokens: []*entity.PoolToken{
			{Address: item.Tokens[0].Address},
			{Address: item.Tokens[1].Address},
		},
		Reserves:    []string{"0", "0"},
		StaticExtra: mustMarshal(item.StaticExtra),
	}

	updated, err := tracker.GetNewPoolState(context.Background(), pool, pool_pkg.GetNewPoolStateParams{})
	require.NoError(t, err,
		"tracker.GetNewPoolState failed — leaf fee contract resolution from router.feeRecipient() "+
			"likely returned an unsupported fee type or zero address")

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
	require.NotNil(t, extra.MaxFee, "maxFee is nil — tracker failed to fetch fee params")
	require.NotNil(t, extra.HalfAmount, "halfAmount is nil — tracker failed to fetch fee params")

	maxFee, _ := uint256.FromBig(extra.MaxFee)
	halfAmount, _ := uint256.FromBig(extra.HalfAmount)
	t.Logf("tracker fetched: maxFee=%s halfAmount=%s reserve=%s", extra.MaxFee, extra.HalfAmount, extra.Reserve)

	// Now verify: for various amountIn values, inverseFee with tracker's params
	// produces a principal that the on-chain router quotes back to amountIn.
	targetRouterAddr := common.HexToAddress(item.StaticExtra.TargetRouter)
	targetRouterBytes32 := common.BytesToHash(targetRouterAddr.Bytes())

	amountsIn := []uint64{
		1_000_000,      // 1 USDC
		5_000_000,      // 5 USDC
		10_000_000,     // 10 USDC
		100_000_000,    // 100 USDC
		1_000_000_000,  // 1000 USDC
		10_000_000_000, // 10000 USDC
	}

	for _, amtIn := range amountsIn {
		amountIn := uint256.NewInt(amtIn)
		principal, fee := inverseFee(amountIn, maxFee, halfAmount)

		// Quote the principal on-chain to get the real total cost
		totalCost := quoteOnChain(t, client, item.StaticExtra.SourceRouter, principal.ToBig(), targetRouterBytes32)
		totalCostU, _ := uint256.FromBig(totalCost)

		var diff uint256.Int
		if totalCostU.Gt(amountIn) {
			diff.Sub(totalCostU, amountIn)
		} else {
			diff.Sub(amountIn, totalCostU)
		}

		assert.False(t, diff.Gt(uint256.NewInt(1)),
			"amountIn=%d principal=%s fee=%s totalCost=%s diff=%s (expected ≤ 1)",
			amtIn, principal, fee, totalCostU, &diff)
	}
}
