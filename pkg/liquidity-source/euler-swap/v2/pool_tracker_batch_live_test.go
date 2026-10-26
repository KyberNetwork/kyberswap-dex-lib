package v2

import (
	"context"
	"math/big"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// countingTransport counts the HTTP requests a client makes, which is what an RPC provider meters.
type countingTransport struct {
	n atomic.Int64
}

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.n.Add(1)
	return http.DefaultTransport.RoundTrip(req)
}

// TestLiveBatchMatchesSinglePool: every EulerSwap v2 pool on Ethereum, read at one block one pool
// at a time over RPC and all together through the state lens, comes out identical. Run with:
//
//	EULER_SWAP_V2_RPC_URL=https://... go test ./pkg/liquidity-source/euler-swap/v2/ -run Live -v
func TestLiveBatchMatchesSinglePool(t *testing.T) {
	rpcURL := os.Getenv("EULER_SWAP_V2_RPC_URL")
	if rpcURL == "" || os.Getenv("CI") != "" {
		t.Skip("set EULER_SWAP_V2_RPC_URL to an Ethereum endpoint to run this")
	}
	ctx := t.Context()

	counter := &countingTransport{}
	rc, err := rpc.DialOptions(ctx, rpcURL, rpc.WithHTTPClient(&http.Client{Transport: counter}))
	require.NoError(t, err)
	client := ethrpc.NewWithClient(ethclient.NewClient(rc)).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	// Every pool the factory lists. GetNewPools only looks at the tail of that list, relying on
	// earlier runs for the rest, so the pools are listed and initialized directly here.
	cfg := shared.Config{FactoryAddress: "0x5fccb84363f020c0cade052c9c654aabf932814a", UseLens: true}
	updater := NewPoolsListUpdater(&cfg, client)
	length, err := updater.getPoolsLength(ctx)
	require.NoError(t, err)
	addresses, err := updater.listPoolAddresses(ctx, 0, length)
	require.NoError(t, err)
	pools, err := updater.initPools(ctx, addresses)
	require.NoError(t, err)
	require.NotEmpty(t, pools)
	fillDecimals(t, ctx, client, pools)

	tracker, err := NewPoolTracker(&cfg, client)
	require.NoError(t, err)

	latest, err := client.GetBlockNumber(ctx)
	require.NoError(t, err)
	block := new(big.Int).SetUint64(latest - 2)

	before := counter.n.Load()
	single := make([]entity.Pool, len(pools))
	for i, p := range pools {
		single[i], err = tracker.readOne(ctx, p, block)
		require.NoError(t, err, "single-pool read of %s", p.Address)
	}
	singleRequests := counter.n.Load() - before

	before = counter.n.Load()
	batch, errs := tracker.getNewPoolStates(ctx, pools, block)
	batchRequests := counter.n.Load() - before

	require.Len(t, batch, len(pools))
	for i, p := range batch {
		require.NoError(t, errs[i], "lens read of %s", pools[i].Address)
		assert.Equal(t, single[i].Reserves, p.Reserves, "reserves of %s", pools[i].Address)
		assert.JSONEq(t, single[i].Extra, p.Extra, "extra of %s", pools[i].Address)
		assert.Equal(t, single[i].BlockNumber, p.BlockNumber, "block of %s", pools[i].Address)
	}

	t.Logf("%d pools at block %s: %d RPC requests one pool at a time, %d through the state lens",
		len(pools), block, singleRequests, batchRequests)
}

// fillDecimals sets each token's decimals, which the lister leaves for later and the tracker's
// quote amounts are scaled by.
func fillDecimals(t *testing.T, ctx context.Context, client *ethrpc.Client, pools []entity.Pool) {
	t.Helper()
	decimals := map[string]*uint8{}
	req := client.NewRequest().SetContext(ctx)
	for _, p := range pools {
		for _, tok := range p.Tokens {
			if _, ok := decimals[tok.Address]; ok {
				continue
			}
			d := new(uint8)
			decimals[tok.Address] = d
			req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: abi.Erc20DecimalsMethod}, []any{d})
		}
	}
	_, err := req.Aggregate()
	require.NoError(t, err)
	for _, p := range pools {
		for _, tok := range p.Tokens {
			tok.Decimals = *decimals[tok.Address]
		}
	}
}

// TestLiveLensSplitsUnderALowGasCap: under a gas cap below what all of Ethereum's pools need
// together (15 pools fit 8M and take 3 calls at 5M, in October 2026), the batch is split, and the
// split read returns exactly what one uncapped call does.
func TestLiveLensSplitsUnderALowGasCap(t *testing.T) {
	rpcURL := os.Getenv("EULER_SWAP_V2_RPC_URL")
	if rpcURL == "" || os.Getenv("CI") != "" {
		t.Skip("set EULER_SWAP_V2_RPC_URL to an Ethereum endpoint to run this")
	}
	ctx := t.Context()
	counter := &countingTransport{}
	rc, err := rpc.DialOptions(ctx, rpcURL, rpc.WithHTTPClient(&http.Client{Transport: counter}))
	require.NoError(t, err)
	client := ethrpc.NewWithClient(ethclient.NewClient(rc)).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	cfg := shared.Config{FactoryAddress: "0x5fccb84363f020c0cade052c9c654aabf932814a"}
	updater := NewPoolsListUpdater(&cfg, client)
	length, err := updater.getPoolsLength(ctx)
	require.NoError(t, err)
	addresses, err := updater.listPoolAddresses(ctx, 0, length)
	require.NoError(t, err)
	pools, err := updater.initPools(ctx, addresses)
	require.NoError(t, err)
	fillDecimals(t, ctx, client, pools)

	inputs := make([]poolInput, 0, len(pools))
	for _, p := range pools {
		staticExtra, vaults, err := poolVaults(p)
		require.NoError(t, err)
		inputs = append(inputs, lensInput(p.Address, staticExtra, vaults))
	}

	latest, err := client.GetBlockNumber(ctx)
	require.NoError(t, err)
	block := new(big.Int).SetUint64(latest - 2)

	tracker, err := NewPoolTracker(&cfg, client)
	require.NoError(t, err)
	// The first uncapped call probes the node's gas cap once per client: take that out of the count.
	_, _, err = tracker.readLens(ctx, inputs, block, 0)
	require.NoError(t, err)
	before := counter.n.Load()
	_, whole, err := tracker.readLens(ctx, inputs, block, 0)
	require.NoError(t, err)
	wholeRequests := counter.n.Load() - before
	before = counter.n.Load()
	_, split, err := tracker.readLens(ctx, inputs, block, 5_000_000)
	require.NoError(t, err)
	splitRequests := counter.n.Load() - before

	assert.Greater(t, splitRequests, wholeRequests, "the capped read was not split")
	assert.Equal(t, whole, split)
}
