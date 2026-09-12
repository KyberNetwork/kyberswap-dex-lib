package lunarbase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func publicJSON(t *testing.T, p entity.Pool) []byte {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func publicExtra(t *testing.T, p entity.Pool) Extra {
	t.Helper()
	var e Extra
	if err := json.Unmarshal([]byte(p.Extra), &e); err != nil {
		t.Fatal(err)
	}
	return e
}
func publicTracker(c *ethrpc.Client) *PoolTracker {
	return NewPoolTracker(&Config{ChainID: valueobject.ChainIDBSC}, c)
}

func TestPublicSnapshotIgnoresPartialEventPayloads(t *testing.T) {
	address := common.HexToAddress(verifiedPoolAddress)
	cases := []struct {
		name      string
		params    pool.GetNewPoolStateParams
		wantError bool
	}{
		{"no_logs", pool.GetNewPoolStateParams{}, false},
		{"matching_log_newer_than_rpc", pool.GetNewPoolStateParams{Logs: []types.Log{{Address: address, BlockNumber: 901}}}, true},
		{"removed_future_fork_log", pool.GetNewPoolStateParams{Logs: []types.Log{{Address: address, BlockNumber: 999, Removed: true}}}, false},
		{"other_pool_future_log", pool.GetNewPoolStateParams{Logs: []types.Log{{Address: common.HexToAddress("0x1234"), BlockNumber: 999}}}, false},
		{"late_matching_log", pool.GetNewPoolStateParams{Logs: []types.Log{{Address: address, BlockNumber: 700, Topics: []common.Hash{common.HexToHash("0x01")}, Data: []byte{255}}}}, false},
		{"duplicate_reordered_malformed_logs", pool.GetNewPoolStateParams{Logs: []types.Log{{Address: address, BlockNumber: 900, Data: []byte{1}}, {Address: address, BlockNumber: 899, Data: []byte{2}}, {Address: address, BlockNumber: 900, Data: []byte{1}}}}, false},
		{"headers_minimum", pool.GetNewPoolStateParams{BlockHeaders: map[uint64]entity.BlockHeader{901: {Number: big.NewInt(901)}}}, true},
		{"headers_equal", pool.GetNewPoolStateParams{BlockHeaders: map[uint64]entity.BlockHeader{900: {Number: big.NewInt(900)}}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newVerifiedFixture()
			p := verifiedEntity()
			stale, _ := json.Marshal(Extra{Paused: true, ConcentrationK: 4096, MaxPunishmentX24: 99999, BlockDelay: 3, FeeAskX24: 999, FeeBidX24: 888})
			p.Extra = string(stale)
			before := publicJSON(t, p)
			got, err := publicTracker(f.client(t)).GetNewPoolState(context.Background(), p, tc.params)
			if !bytes.Equal(publicJSON(t, p), before) {
				t.Fatal("public refresh mutated input entity")
			}
			if tc.wantError {
				if err == nil || !bytes.Equal(publicJSON(t, got), before) {
					t.Fatalf("behind snapshot was published: err=%v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			e := publicExtra(t, got)
			if e.Paused || e.ConcentrationK != 0 || e.MaxPunishmentX24 != 125000 || e.BlockDelay != 25 || e.FeeAskX24 != 15 || e.FeeBidX24 != 25 {
				t.Fatalf("partial cache fields survived RPC refresh: %+v", e)
			}
			if got.BlockNumber != 900 || got.Reserves[0] != "1230000000" || got.Reserves[1] != "4560000000" {
				t.Fatal("snapshot fields not coherent")
			}
			if len(f.requests) != 3 {
				t.Fatalf("want3RPC calls, got%d", len(f.requests))
			}
		})
	}
}

func TestPublicSnapshotReplacesSameHeightAndRejectsMidflightReorg(t *testing.T) {
	for _, midflight := range []bool{false, true} {
		t.Run(fmt.Sprintf("midflight-%t", midflight), func(t *testing.T) {
			f := newVerifiedFixture()
			p := verifiedEntity()
			p.BlockNumber = 900
			p.Extra = `{"bh":"old-fork","mp":99999}`
			before := publicJSON(t, p)
			if midflight {
				h := *f.header
				h.Extra = []byte{222}
				f.recheck = &h
			}
			got, err := publicTracker(f.client(t)).GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
			if midflight {
				if err == nil || !bytes.Equal(publicJSON(t, got), before) {
					t.Fatal("midflight replacement accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if publicExtra(t, got).BlockHash != f.header.Hash().Hex() || publicExtra(t, got).MaxPunishmentX24 != 125000 {
				t.Fatal("same-height newer canonical snapshot was skipped")
			}
		})
	}
}

func TestPublicSnapshotPauseAndUnpauseWithoutEvents(t *testing.T) {
	for _, paused := range []bool{false, true} {
		t.Run(fmt.Sprintf("paused-%t", paused), func(t *testing.T) {
			f := newVerifiedFixture()
			p := verifiedEntity()
			old, _ := json.Marshal(Extra{Paused: !paused})
			p.Extra = string(old)
			if paused {
				f.items[7].ReturnData = verifiedWord(big.NewInt(1))
			}
			got, err := publicTracker(f.client(t)).GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
			if err != nil {
				t.Fatal(err)
			}
			if publicExtra(t, got).Paused != paused {
				t.Fatal("pause state not refreshed")
			}
			sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: got, ChainID: valueobject.ChainIDBSC})
			if err != nil {
				t.Fatal(err)
			}
			_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(1000)}, TokenOut: p.Tokens[1].Address})
			if paused && !errors.Is(err, ErrPoolPaused) {
				t.Fatalf("paused quote wasn't rejected: %v", err)
			}
			if !paused && err != nil {
				t.Fatalf("unpaused valid quote rejected: %v", err)
			}
		})
	}
}

func TestPublicSnapshotFailuresPreserveCaller(t *testing.T) {
	for _, name := range []string{"header_failure", "call_failure", "malformed_required", "old_rpc_snapshot", "token_order", "static_native_mismatch", "missing_token"} {
		t.Run(name, func(t *testing.T) {
			f := newVerifiedFixture()
			p := verifiedEntity()
			switch name {
			case "header_failure":
				f.failMethod = "eth_getBlockByNumber"
			case "call_failure":
				f.failMethod = "eth_call"
			case "malformed_required":
				f.items[7].ReturnData = []byte{1}
			case "old_rpc_snapshot":
				p.BlockNumber = 901
			case "token_order":
				p.Tokens[0], p.Tokens[1] = p.Tokens[1], p.Tokens[0]
			case "static_native_mismatch":
				p.StaticExtra = `{"n":true}`
			case "missing_token":
				p.Tokens = p.Tokens[:1]
			}
			before := publicJSON(t, p)
			got, err := publicTracker(f.client(t)).GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
			if err == nil {
				t.Fatal("invalid refresh returned success")
			}
			if !bytes.Equal(before, publicJSON(t, p)) || !bytes.Equal(before, publicJSON(t, got)) {
				t.Fatal("failed refresh changed caller or returned a partial pool")
			}
		})
	}
}

func TestPublicSnapshotConcurrentClientsAndPools(t *testing.T) {
	fixtures := []*verifiedFixture{newVerifiedFixture(), newVerifiedFixture()}
	entities := []entity.Pool{verifiedEntity(), verifiedEntity()}
	entities[1].Address = "0x0000000000000000000000000000000000004567"
	fixtures[1].expectedTarget = common.HexToAddress(entities[1].Address)
	fixtures[1].items[5].ReturnData = verifiedWord(big.NewInt(999000000))
	trackers := []*PoolTracker{publicTracker(fixtures[0].client(t)), publicTracker(fixtures[1].client(t))}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := range trackers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p := entities[i]
			for n := 0; n < 10; n++ {
				got, err := trackers[i].GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
				if err != nil {
					errs <- err
					return
				}
				want := "1230000000"
				if i == 1 {
					want = "999000000"
				}
				if got.Address != p.Address || got.Reserves[0] != want {
					errs <- fmt.Errorf("cross-pool/client state for%d: %+v", i, got)
					return
				}
				p = got
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	for i, f := range fixtures {
		if len(f.requests) != 30 {
			t.Fatalf("client%d want30calls, got%d", i, len(f.requests))
		}
	}
}

func TestPublicLegacyFlashCompatibilityFailsClosed(t *testing.T) {
	InitFlashBlockSubscriber("ws://invalid.invalid", "ws://invalid.invalid", []common.Address{common.HexToAddress(verifiedPoolAddress)})
	if GetFlashBlockSubscriber() != nil {
		t.Fatal("retired subscriber still publishes an instance")
	}
	var s *FlashBlockSubscriber
	if s.GetLatestState() != nil || (&FlashBlockSubscriber{}).GetLatestState() != nil {
		t.Fatal("legacy subscriber returned partial state")
	}
	if !(new(poolState)).IsStale() {
		t.Fatal("legacy state must not be quoteable")
	}
}
