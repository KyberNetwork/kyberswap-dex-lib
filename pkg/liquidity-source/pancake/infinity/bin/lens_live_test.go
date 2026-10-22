package bin

import (
	"bytes"
	"io"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/bin/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// retryTransport sends a request the node rate-limited again, after a pause, so a shared
// endpoint's limit does not fail the test.
type retryTransport struct{}

func (retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	for attempt := 0; ; attempt++ {
		req.Body = io.NopCloser(bytes.NewReader(body))
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusTooManyRequests || attempt == 20 {
			return resp, err
		}
		_ = resp.Body.Close()
		time.Sleep(time.Duration(attempt+1) * 250 * time.Millisecond)
	}
}

// TestLiveLens: on BSC and Base, for pools of the Pancake, SquadSwap and Omni managers, a bootstrap
// through the lens holds every bin getBin reports for those ids at the same block, the active bin
// included, a gas cap that forces paging reads the same set, a stored set that lost the active bin
// and another gets them back from the lens on its next update, and a pool without state, as a pool
// factory builds it, gets its bins whole from its first update. Run with either or both of:
//
//	PANCAKE_BIN_BSC_RPC_URL=https://... PANCAKE_BIN_BASE_RPC_URL=https://... \
//	  go test ./pkg/liquidity-source/pancake/infinity/bin/ -run LiveLens -v
func TestLiveLens(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("live")
	}
	pools := []struct {
		env, name, manager, id string
	}{
		{"PANCAKE_BIN_BSC_RPC_URL", "bsc pancake", "0xC697d2898e0D09264376196696c51D7aBbbAA4a9",
			"0xa3aabe0dce20b4afc88a7770742be68559c45b0edabbfb85cf5068fb8fe1040f"},
		{"PANCAKE_BIN_BSC_RPC_URL", "bsc pancake", "0xC697d2898e0D09264376196696c51D7aBbbAA4a9",
			"0x0639caccfbfe97e4b70b7ce7bf784afef09b0ca41d30aeefc3fad34265a4bcb8"},
		{"PANCAKE_BIN_BSC_RPC_URL", "bsc squadswap", "0x7f8e5a02f9007b19408ef731982adbba7ae08eb1",
			"0x739928579ca65888b8466b8a5074514c5ddcd453b9216dd396b55e15ef38923a"},
		{"PANCAKE_BIN_BSC_RPC_URL", "bsc squadswap", "0x7f8e5a02f9007b19408ef731982adbba7ae08eb1",
			"0xe2a47d46a8ea0d693f292a7fa354fad23bf002986341f5e1f3e6b391e13840d5"},
		{"PANCAKE_BIN_BASE_RPC_URL", "base omni", "0xBF7927161C04a80e533fE8d03b8C511ac310fD28",
			"0xc875275feedaf39095fa9218427f2751ab11fd4789c5fb021def09dafb07ce10"},
		{"PANCAKE_BIN_BASE_RPC_URL", "base omni, active bin out of the tree", "0xBF7927161C04a80e533fE8d03b8C511ac310fD28",
			"0x7fa7dd71c4380726d03bb4c56118cbd16cb99ce67bbf451dc571856eb15d969c"},
		{"PANCAKE_BIN_BASE_RPC_URL", "base omni", "0xBF7927161C04a80e533fE8d03b8C511ac310fD28",
			"0x3c41b4a6e846a31cf8ae3076a5a7515193eb5a2df164fdf291a420d5861596b5"},
	}
	ran := false
	for _, pc := range pools {
		rpcURL := os.Getenv(pc.env)
		if rpcURL == "" {
			continue
		}
		ran = true
		chainID := lo.Ternary(pc.env == "PANCAKE_BIN_BSC_RPC_URL", valueobject.ChainIDBSC, valueobject.ChainIDBase)
		t.Run(pc.name+" "+pc.id[:10], func(t *testing.T) { liveLens(t, rpcURL, chainID, pc.manager, pc.id) })
	}
	if !ran {
		t.Skip("set PANCAKE_BIN_BSC_RPC_URL or PANCAKE_BIN_BASE_RPC_URL")
	}
}

func liveLens(t *testing.T, rpcURL string, chainID valueobject.ChainID, manager, id string) {
	ctx := t.Context()
	raw, err := rpc.DialOptions(ctx, rpcURL, rpc.WithHTTPClient(&http.Client{Transport: retryTransport{}}))
	require.NoError(t, err)
	client := ethrpc.NewWithClient(ethclient.NewClient(raw)).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	cfg := &Config{ChainID: int(chainID), BinPoolManagerAddress: manager, UseLens: true}
	tracker := NewPoolTracker(cfg, client, nil)

	// The pool as its factory would build it from its Initialize log.
	key, err := shared.BinPoolManagerABI.Methods["poolIdToPoolKey"].Outputs.Unpack(lo.Must(client.GetETHClient().
		CallContract(ctx, callMsg(manager, lo.Must(shared.BinPoolManagerABI.Pack("poolIdToPoolKey",
			common.HexToHash(id)))), nil)))
	require.NoError(t, err)
	head, err := client.GetETHClient().BlockNumber(ctx)
	require.NoError(t, err)
	block := head - 3
	created, err := NewPoolFactory(cfg).newPool(&abi.PancakeInfinityPoolManagerInitialize{
		Id:         common.HexToHash(id),
		Currency0:  key[0].(common.Address),
		Currency1:  key[1].(common.Address),
		Hooks:      key[2].(common.Address),
		Fee:        key[4].(*big.Int),
		Parameters: key[5].([32]byte),
		ActiveId:   big.NewInt(1 << 23),
	}, 1)
	require.NoError(t, err)
	require.False(t, hasState(*created), "a pool factory leaves the bins out")

	// Bootstrap at a pinned block: every bin the lens returns is getBin's at that block.
	snap, err := tracker.snapshot(ctx, id, new(big.Int).SetUint64(block))
	require.NoError(t, err)
	assert.EqualValues(t, block, snap.block, "the read's block is the one pinned")
	ids := append(lo.Map(snap.bins, func(b Bin, _ int) uint32 { return b.ID }), snap.activeID)
	direct, err := tracker.queryRPCBins(ctx, id, lo.Uniq(ids), block)
	require.NoError(t, err)
	assert.Equal(t, binsByID(snap.bins), binsByID(filterEmptyAndSortBins(direct)), "lens bins equal getBin's")
	activeHolds := lo.ContainsBy(snap.bins, func(b Bin) bool { return b.ID == snap.activeID })
	t.Logf("block %d: %d bins, active %d holds reserves: %v", block, len(snap.bins), snap.activeID, activeHolds)

	// Paged under a low gas cap, the same set.
	paged := NewPoolTracker(&Config{ChainID: int(chainID), BinPoolManagerAddress: manager, UseLens: true, LensGas: 250_000},
		client, nil)
	pagedSnap, err := paged.snapshot(ctx, id, new(big.Int).SetUint64(block))
	require.NoError(t, err)
	assert.Equal(t, binsByID(snap.bins), binsByID(pagedSnap.bins), "paged reads the same set")

	booted, _, err := tracker.bootstrapFromLens(ctx, *created, new(big.Int).SetUint64(block))
	require.NoError(t, err)
	assert.True(t, hasState(booted))
	assert.EqualValues(t, block, booted.BlockNumber)
	_, err = NewPoolSimulator(booted, chainID)
	require.NoError(t, err, "the bootstrapped pool builds a simulator")
	var extra Extra
	require.NoError(t, unmarshalExtra(booted, &extra))
	_, err = GetBinById(extra.Bins, extra.ActiveBinID)
	assert.NoError(t, err, "the active bin is in the stored set, empty or not")

	// A pool without state gets its bins whole from its first update, the empty log the caller
	// sends to trigger one.
	trigger := []ethtypes.Log{{BlockNumber: head}}
	updated, err := tracker.GetNewPoolState(ctx, *created, poolpkg.GetNewPoolStateParams{Logs: trigger})
	require.NoError(t, err)
	require.True(t, hasState(updated), "a pool without state gets its bins")
	assert.GreaterOrEqual(t, updated.BlockNumber, head)

	// A stored set that lost the active bin and another gets them back from the lens; reading the
	// stored ids again cannot.
	if !activeHolds || len(snap.bins) < 2 {
		t.Logf("active bin empty or alone; repair case skipped")
		return
	}
	other := lo.Filter(snap.bins, func(b Bin, _ int) bool { return b.ID != snap.activeID })[0].ID
	lossy := withExtraBins(t, booted, lo.Filter(snap.bins, func(b Bin, _ int) bool {
		return b.ID != snap.activeID && b.ID != other
	}))
	// A tracker of its own: the guard keeps a pool from the lens for a cooldown after a read.
	repaired, err := NewPoolTracker(cfg, client, nil).GetNewPoolState(ctx, lossy,
		poolpkg.GetNewPoolStateParams{Logs: trigger})
	require.NoError(t, err)
	var repairedExtra Extra
	require.NoError(t, unmarshalExtra(repaired, &repairedExtra))
	_, err = GetBinById(repairedExtra.Bins, other)
	assert.NoError(t, err, "the lost bin %d is back", other)
	active, err := GetBinById(repairedExtra.Bins, repairedExtra.ActiveBinID)
	require.NoError(t, err)
	assert.False(t, active.ReserveX.IsZero() && active.ReserveY.IsZero(), "the active bin holds its reserves")

	withoutLens := NewPoolTracker(&Config{ChainID: int(chainID), BinPoolManagerAddress: manager}, client, nil)
	kept, err := withoutLens.GetNewPoolState(ctx, lossy, poolpkg.GetNewPoolStateParams{Logs: trigger})
	require.NoError(t, err)
	var keptExtra Extra
	require.NoError(t, unmarshalExtra(kept, &keptExtra))
	_, err = GetBinById(keptExtra.Bins, other)
	assert.ErrorIs(t, err, ErrBinIDNotFound, "without the lens the lost bin stays lost")
}

func callMsg(to string, data []byte) ethereum.CallMsg {
	addr := common.HexToAddress(to)
	return ethereum.CallMsg{To: &addr, Data: data}
}

func binsByID(bins []Bin) map[uint32][2]string {
	return lo.SliceToMap(bins, func(b Bin) (uint32, [2]string) {
		return b.ID, [2]string{b.ReserveX.Dec(), b.ReserveY.Dec()}
	})
}

func unmarshalExtra(p entity.Pool, extra *Extra) error {
	return json.Unmarshal([]byte(p.Extra), extra)
}

func withExtraBins(t *testing.T, p entity.Pool, bins []Bin) entity.Pool {
	var extra Extra
	require.NoError(t, unmarshalExtra(p, &extra))
	extra.Bins = bins
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(b)
	return p
}
