package efficientprop

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// liveChain is one chain's deployed front as its deployment record names it, and what the source must read there.
type liveChain struct {
	name     string
	chain    valueobject.ChainID
	rpc      string // EFFICIENT_PROP_LIVE_RPC_<chain id> overrides it
	cell     common.Hash
	door     common.Hash
	implHash common.Hash
	mode     uint64
	// desk1 is the chain's first maker's desk contract, the CREATE2 child of the front for desk 1 under cell
	desk1 common.Address
	// builder: read through /state, with the front's layout and desk code hash
	builder bool
}

// liveFront and liveFactory are the front and its door factory on every chain this build is deployed to, and
// liveImpl the front's implementation there: one address, each chain's own code.
var (
	liveFront   = "0x0000000000403aeB66691063CC2fD42E71d6AcD6"
	liveFactory = "0x39eD9993449C96503f6a629cCD2BB5F132D7cc93"
	liveImpl    = common.HexToAddress("0x48C6efab497EeC8501483Ceb23B6623f626d8371")
)

var liveChains = []liveChain{
	{name: "Base", chain: valueobject.ChainIDBase, rpc: "https://base-rpc.publicnode.com",
		cell:     common.HexToHash("0x0972dba2b21d750e0679e53404b218ee08e291b24b79194a666701306c1371f1"),
		door:     common.HexToHash("0x873f0dfb8706beb017672a11fd93543e3ebcd737171aa1432d991a20b33a2e56"),
		implHash: common.HexToHash("0x614d29e839560fe31787583f5e133e2de8157aeb58d63c4f92806d04dca00b66"),
		desk1:    common.HexToAddress("0x2b347C8e89520aF26d30e6932f6273B699Ae746A"),
		mode:     clockBlocks},
	{name: "BNB Chain", chain: valueobject.ChainIDBSC, rpc: "https://bsc-rpc.publicnode.com",
		cell:     common.HexToHash("0xb4f7512e65eee793ac5939d62b8383d8544eb1ca850c276fd706e97a0b7938de"),
		door:     common.HexToHash("0x0716310d0f4dac4c4f088ccf08c4fe5bfdfea57e9073fcd90a26feefc7c0d6d9"),
		implHash: common.HexToHash("0x2aa41223971899b7002026d8a4b6e693b253e647ea3cec7df158424803a711b9"),
		desk1:    common.HexToAddress("0x4f8dd270acE370dE2f6Ac5EBf3230cBE099a0753"),
		mode:     clockPrecompile, builder: true},
}

// TestLiveFrontsOnBaseAndBNBChain reads the deployed front on Base and on BNB Chain from a public node and checks what
// this source reads there against the chain's deployment record: the front's desk code hash (CELL_CODEHASH, the
// packed desk's on Base), its door factory's DOOR_CODEHASH and FRONT, its implementation's code hash, its clock (the
// block count from Base's millisecond anchor; BNB Chain's millisecond precompile mode, whose precompile answers
// nothing yet), and that the lister runs on the front as configured and finds every listing of desk 1 from the chain
// alone, none named in the config, each of which a refresh reads at the head (on BNB Chain with a /state answer that
// holds no update, so nothing is quoted). A chain where the front, its implementation, its door factory or desk 1's
// contract holds no code yet is skipped with a note, not failed: a chain landing in part is not a fault here.
// EFFICIENT_PROP_LIVE_RPC_<chain id> points a chain at another node, such as a local fork of a rehearsal.
//
//	EFFICIENT_PROP_LIVE=1 go test ./pkg/liquidity-source/efficient-prop -run TestLive -v
func TestLiveFrontsOnBaseAndBNBChain(t *testing.T) {
	if os.Getenv("EFFICIENT_PROP_LIVE") == "" {
		t.Skip("EFFICIENT_PROP_LIVE is not set")
	}
	for _, c := range liveChains {
		t.Run(c.name, func(t *testing.T) {
			if u := os.Getenv("EFFICIENT_PROP_LIVE_RPC_" + big.NewInt(int64(c.chain)).String()); u != "" {
				c.rpc = u
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			client := ethrpc.New(c.rpc).SetMulticallContract(multicall3)
			eth := client.GetETHClient()
			id, err := eth.ChainID(ctx)
			require.NoError(t, err)
			require.Equal(t, uint64(c.chain), id.Uint64())
			front := common.HexToAddress(liveFront)
			for _, a := range []struct {
				name string
				addr common.Address
			}{{"front", front}, {"implementation", liveImpl}, {"door factory", common.HexToAddress(liveFactory)},
				{"desk 1's contract", c.desk1}} {
				code, err := eth.CodeAt(ctx, a.addr, nil)
				require.NoError(t, err)
				if len(code) == 0 {
					t.Skipf("%s: the %s %s holds no code at the head: not deployed, or deployed in part", c.name,
						a.name, a.addr)
				}
			}

			f := Front{Address: liveFront, DoorFactory: liveFactory, FillMode: FillModeDoor,
				Implementations: []string{liveImpl.Hex()}}
			// no /state url answers for chain 56 yet: a local one answers no update, so a refresh reads the head and
			// quotes nothing
			st := newStateServer(t)
			if c.builder {
				f.Layout, f.CellCodehash = LayoutBuilder, c.cell.Hex()
				f.Titan = &titan.Config{URLs: []string{st.URL}}
				f.BundleOnly = true
			} else {
				f.BaseTime = "0x4200000000000000000000000000000000000030"
			}
			cfg := &Config{ChainID: c.chain, ApprovedDesks: []string{c.desk1.Hex()}, Fronts: []Front{f}}
			require.NoError(t, cfg.validate())

			hashes := newCodeHashes(client)
			cell, err := hashes.cell(ctx, cfg.Fronts[0])
			require.NoError(t, err)
			assert.Equal(t, c.cell, cell, "CELL_CODEHASH")
			door, err := hashes.door(ctx, cfg.Fronts[0])
			require.NoError(t, err)
			assert.Equal(t, c.door, door, "DOOR_CODEHASH")
			implHash, err := eth.CodeAt(ctx, liveImpl, nil)
			require.NoError(t, err)
			assert.Equal(t, c.implHash, crypto.Keccak256Hash(implHash), "the implementation's code")
			slot, err := eth.StorageAt(ctx, front, common.HexToHash(
				"0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc"), nil)
			require.NoError(t, err)
			assert.Equal(t, liveImpl, common.BytesToAddress(slot), "the proxy's implementation")
			// desk 1's contract is the front's CREATE2 child under its desk code hash (deskContractOf)
			assert.Equal(t, c.desk1, deskContractOf(front, cell, 1), "desk 1's contract")

			var clock frontClock
			_, err = client.NewRequest().SetContext(ctx).
				AddCall(&ethrpc.Call{ABI: frontABI, Target: liveFront, Method: "clock"}, []any{&clock}).Call()
			require.NoError(t, err)
			require.True(t, clock.valid())
			assert.Equal(t, c.mode, clock.ClockMode.Uint64(), "the front's clock mode")
			if c.builder {
				out, err := eth.CallContract(ctx, ethereum.CallMsg{To: &msPrecompile,
					Data: baseTimeABI.Methods["timestampMs"].ID}, nil)
				require.NoError(t, err)
				assert.Empty(t, out, "the millisecond precompile answers nothing yet: the front reads the seal time")
			}

			u, err := NewPoolsListUpdater(cfg, client)
			require.NoError(t, err)
			pools, _, err := u.GetNewPools(ctx, nil)
			require.NoError(t, err)
			t.Logf("%s: %d listing door(s) listed from the front", c.name, len(pools))
			tr, err := NewPoolTracker(cfg, client)
			require.NoError(t, err)
			for _, p := range pools {
				var se StaticExtra
				require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
				assert.Equal(t, cfg.Fronts[0].Address, se.Front)
				for _, tok := range p.Tokens {
					tok.Decimals = 18
				}
				got, err := tr.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
				require.NoError(t, err, "listing %d", se.Listing)
				e := extraOf(t, got)
				t.Logf("%s: listing %d door %s pool %s block %d credit mode %d tip cap %d idle %v", c.name,
					se.Listing, p.Address, se.Custody, got.BlockNumber, e.CreditMode, e.TipCapWei, e.Idle)
				if c.builder {
					assert.Empty(t, e.Ladders[0], "no /state update: nothing quoted")
					assert.Nil(t, e.Credit)
					assert.True(t, e.Builder)
				}
			}
		})
	}
}
