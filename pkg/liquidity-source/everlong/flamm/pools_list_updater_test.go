package everlongflamm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The lister over the recorded listing (testdata/tracker_rpc_51302915.json.gz): the registered pool is listed once
// with its pinned wiring, and any contract whose runtime code is not the registry's keeps it unlisted.

func TestListerReplay(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	u := NewPoolsListUpdater(tapeConfig(), tp.client())
	pools, md, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	p := pools[0]
	require.Equal(t, "0xc0fdcb1799ccc2cebaa1fe247157b0df33d57572", p.Address)
	require.Equal(t, DexType, p.Type)
	require.Equal(t, []string{"0", "0"}, []string(p.Reserves))
	require.Equal(t, "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf", p.Tokens[0].Address)
	require.Equal(t, "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", p.Tokens[1].Address)
	require.True(t, p.Tokens[0].Swappable && p.Tokens[1].Swappable)

	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &se))
	prof := &c104
	require.Equal(t, prof.hookSet(), se.Hooks)
	w, err := validStatic(&se, p.Address, p.Type, []string{p.Tokens[0].Address, p.Tokens[1].Address})
	require.NoError(t, err)
	// The venue set is mutable wiring, so it is published in Extra, not pinned in StaticExtra (which a listing
	// cursor and pool-service both treat as the pool's identity).
	require.NotContains(t, p.StaticExtra, "venues")
	venues := entityVenues(t, p)
	require.Len(t, venues, 1)
	require.Equal(t, prof.Account, venues[0].Account)
	require.Equal(t, common.HexToHash("0x9103c3b4e834476c9a62ea009ba2c884ee42e94e6e314a26f04d312434191836"),
		venues[0].MarketID)
	require.Equal(t, adaptiveCurveIrm, venues[0].Irm)
	require.Equal(t, "860000000000000000", venues[0].Lltv.Dec())
	require.NoError(t, validVenues(w, venues))
	require.Equal(t, common.HexToAddress("0xBCF85224fc0756B9Fa45aA7892530B47e10b6433"), se.SequencerFeed)
	require.Equal(t, [2]common.Address{common.HexToAddress("0x07DA0E54543a844a80ABE69c8A12F22B3aA59f9D"),
		common.HexToAddress("0x7e860098F58bBFC8648a4311b374B1D669a2bc6B")}, se.Aggregators)
	require.Equal(t, [2]uint64{3600, 90000}, se.Heartbeats)

	// The cursor suppresses an unchanged relisting and forgets a pool the factory no longer lists. A poll that
	// finds the wiring the pool is listed with stops at the view round: the venue rounds and the runtime-code
	// batch, which verify only what an unchanged StaticExtra cannot have changed, are not sent at all.
	var asked map[string]int
	tp.mutate = tapeRecorder(&asked)
	again, md2, err := u.GetNewPools(context.Background(), md)
	tp.mutate = nil
	require.NoError(t, err)
	require.Empty(t, again)
	require.JSONEq(t, string(md), string(md2))
	require.Equal(t, map[string]int{"eth_call": 2}, asked, "a steady-state poll is two round trips")

	stale, _ := json.Marshal(Metadata{Listed: map[string]string{"0x00000000000000000000000000000000000000aa": "x",
		p.Address: "0xstale"}})
	asked = nil
	tp.mutate = tapeRecorder(&asked)
	relisted, md3, err := u.GetNewPools(context.Background(), stale)
	tp.mutate = nil
	require.NoError(t, err)
	require.Len(t, relisted, 1, "a changed digest relists")
	require.NotContains(t, string(md3), "0x00000000000000000000000000000000000000aa")
	// Six round trips: the five aggregates and the one JSON-RPC batch the nine eth_getCode requests are sent in.
	require.Equal(t, map[string]int{"eth_call": 5, "eth_getCode": 9}, asked, "a changed listing is verified in full")

	// A registered pool the factory does not own (isPool false) is not listed and leaves the cursor: the lister reads
	// that flag instead of enumerating the permissionless pools() array.
	isPool, err := factoryABI.Pack("isPool", prof.Pool)
	require.NoError(t, err)
	tp.mutate = aggregateCallMutation(t, func(target common.Address, data []byte) bool {
		return target == prof.Factory && bytes.Equal(data, isPool)
	}, func(r *tapeResult) { r.ReturnData = make([]byte, 32) })
	disowned, md4, err := u.GetNewPools(context.Background(), md)
	require.NoError(t, err)
	require.Empty(t, disowned)
	require.NotContains(t, string(md4), p.Address)
	tp.mutate = nil

	// Wrong runtime code on any registered contract of the pool: not listed, and not an error.
	for _, who := range []common.Address{prof.Hook, prof.Implementation, prof.Account, prof.PriceFeed} {
		tp.mutate = func(method string, params json.RawMessage, e *tapeEntry) {
			if method == "eth_getCode" && strings.Contains(strings.ToLower(string(params)), strings.ToLower(who.Hex()[2:])) {
				e.Result = json.RawMessage(`"0x00"`)
			}
		}
		pools, _, err := u.GetNewPools(context.Background(), nil)
		require.NoError(t, err)
		require.Empty(t, pools, who.Hex())
	}
	tp.mutate = nil

	// A pool whose views the chain answers with a revert, or with data that does not decode, is skipped like a
	// registry mismatch; a transport failure fails the run.
	for name, edit := range map[string]func(r *tapeResult){
		"reverted":    func(r *tapeResult) { r.Success = false },
		"undecodable": func(r *tapeResult) { r.ReturnData = nil },
	} {
		tp.mutate = aggregateMutation(t, 2, func(results []tapeResult) { edit(&results[0]) })
		pools, _, err := u.GetNewPools(context.Background(), nil)
		require.NoError(t, err, name)
		require.Empty(t, pools, name)
	}
	tp.mutate = func(method string, _ json.RawMessage, e *tapeEntry) {
		if method == "eth_getCode" {
			e.Result, e.Error = nil, &tapeError{Code: -32000, Message: "unavailable"}
		}
	}
	_, _, err = u.GetNewPools(context.Background(), nil)
	require.Error(t, err, "transport")
	tp.mutate = nil

	// Config errors.
	_, _, err = NewPoolsListUpdater(&Config{DexID: DexType, ChainID: valueobject.ChainIDBase}, tp.client()).
		GetNewPools(context.Background(), nil)
	require.ErrorIs(t, err, ErrInvalidProfile)
	// A factory the registry does not know on this chain lists nothing.
	pools, _, err = NewPoolsListUpdater(&Config{DexID: DexType, ChainID: valueobject.ChainIDEthereum,
		Factory: prof.Factory.Hex()}, tp.client()).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, pools)
}

// tapeRecorder counts the JSON-RPC requests the tape is asked for, by method. Storage and code reads are sent in
// batches of at most ten, so a method's count is requests, not round trips.
func tapeRecorder(asked *map[string]int) func(string, json.RawMessage, *tapeEntry) {
	var mu sync.Mutex
	return func(method string, _ json.RawMessage, _ *tapeEntry) {
		mu.Lock()
		defer mu.Unlock()
		if *asked == nil {
			*asked = map[string]int{}
		}
		(*asked)[method]++
	}
}

// TestListerProfileChecks: every check listPool makes on what the chain answered, one answer at a time. None of
// them is reported as an error -- a pool that is not what the registry has is simply not listed -- so each needs a
// case of its own or nothing depends on it.
func TestListerProfileChecks(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	u := NewPoolsListUpdater(tapeConfig(), tp.client())
	prof := &c104
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1, "the control listing")
	other := common.HexToAddress("0x0000000000000000000000000000000000000bad")
	u64 := func(v uint64) []byte { return new(big.Int).SetUint64(v).Bytes() }
	for _, c := range []struct {
		name  string
		match func(common.Address, []byte) bool
		edit  func(r *tapeResult)
	}{
		{"pool asset", callTo(t, prof.Pool, &flammABI, "asset"), setResultWord(0, other.Bytes())},
		{"loan asset", callTo(t, prof.Pool, &flammABI, "loanAsset"), setResultWord(0, other.Bytes())},
		{"loan count", callTo(t, prof.Pool, &flammABI, "loanCount"), setResultWord(0, u64(2))},
		{"router loan count", callTo(t, prof.Router, &routerABI, "loanCount"), setResultWord(0, u64(2))},
		{"pool router", callTo(t, prof.Pool, &flammABI, "router"), setResultWord(0, other.Bytes())},
		{"pool price feed", callTo(t, prof.Pool, &flammABI, "priceFeed"), setResultWord(0, other.Bytes())},
		{"pool factory", callTo(t, prof.Pool, &flammABI, "factory"), setResultWord(0, other.Bytes())},
		{"factory implementation", callTo(t, prof.Factory, &factoryABI, "implementation"),
			setResultWord(0, other.Bytes())},
		{"hook set", callTo(t, prof.Pool, &flammABI, "hooks"), setResultWord(3, other.Bytes())},
		{"swap hook pool", callTo(t, prof.Hook, &hookABI, "POOL"), setResultWord(0, other.Bytes())},
		{"swap hook loan scale", callTo(t, prof.Hook, &hookABI, "LOAN_SCALE"), addWord(0, 1)},
		{"genesis strategy", callTo(t, prof.Hook, &hookABI, "genesisStrategyHash"), addWord(0, 1)},
		{"leverage hook binding", callTo(t, prof.LeverageHook, &levHookABI, "HOOK"), setResultWord(0, other.Bytes())},
		{"leverage hook pool", callTo(t, prof.LeverageHook, &levHookABI, "POOL"), setResultWord(0, other.Bytes())},
		{"leverage hook loan scale", callTo(t, prof.LeverageHook, &levHookABI, "LOAN_SCALE"), addWord(0, 1)},
		{"spread hook pool", callTo(t, prof.SpreadHook, &spreadHookABI, "POOL"), setResultWord(0, other.Bytes())},
		{"no venues", callTo(t, prof.Router, &routerABI, "venueCount"), setResultWord(0, nil)},
		{"too many venues", callTo(t, prof.Router, &routerABI, "venueCount"), setResultWord(0, u64(maxVenues+1))},
		// The venue set the listing publishes: the account it names is the one whose runtime code the listing
		// pins, and the market behind its id has to be the pool's own pair (readVenues).
		{"foreign venue account", callTo(t, prof.Router, &routerABI, "venue"), setResultWord(0, other.Bytes())},
		{"venue market loan token", callTo(t, prof.Morpho, &morphoABI, "idToMarketParams"),
			setResultWord(0, other.Bytes())},
		{"venue market collateral token", callTo(t, prof.Morpho, &morphoABI, "idToMarketParams"),
			setResultWord(1, other.Bytes())},
		{"venue market irm", callTo(t, prof.Morpho, &morphoABI, "idToMarketParams"), setResultWord(3, other.Bytes())},
		{"venue market oracle", callTo(t, prof.Morpho, &morphoABI, "idToMarketParams"), setResultWord(2, nil)},
	} {
		// A tamper that renames a contract also sends the codehash round to an address the tape has no answer
		// for; it has no code, which is what the chain would say.
		noCode := func(method string, params json.RawMessage, e *tapeEntry) {
			if method == "eth_getCode" && strings.Contains(strings.ToLower(string(params)),
				strings.ToLower(other.Hex()[2:])) {
				e.Result, e.Error = json.RawMessage(`"0x"`), nil
			}
		}
		tp.mutate = chainMutations(aggregateCallMutation(t, c.match, c.edit), noCode)
		pools, _, err := u.GetNewPools(context.Background(), nil)
		require.NoError(t, err, c.name)
		require.Empty(t, pools, c.name)
	}
	tp.mutate = nil
}

// TestValidStatic: every pinned field of a listing, and every entry of a refreshed venue set, is checked against
// the registries.
func TestValidStatic(t *testing.T) {
	t.Parallel()
	stored := loadTracked(t)
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(stored.StaticExtra), &se))
	tokens := []string{stored.Tokens[0].Address, stored.Tokens[1].Address}
	w, err := validStatic(&se, stored.Address, DexType, tokens)
	require.NoError(t, err)
	other := common.HexToAddress("0x0000000000000000000000000000000000000bad")
	for name, tamper := range map[string]func(s *StaticExtra){
		"version":        func(s *StaticExtra) { s.ProfileVersion++ },
		"chain":          func(s *StaticExtra) { s.ChainID = valueobject.ChainIDEthereum },
		"factory":        func(s *StaticExtra) { s.Factory = other },
		"implementation": func(s *StaticExtra) { s.Implementation = other },
		"router":         func(s *StaticExtra) { s.Router = other },
		"feed":           func(s *StaticExtra) { s.PriceFeed = other },
		"morpho":         func(s *StaticExtra) { s.Morpho = other },
		"pool asset":     func(s *StaticExtra) { s.PoolAsset = other },
		"hooks":          func(s *StaticExtra) { s.Hooks[4] = other },
	} {
		c := se
		tamper(&c)
		_, err := validStatic(&c, stored.Address, DexType, tokens)
		require.ErrorIs(t, err, ErrInvalidProfile, name)
	}

	venues := entityVenues(t, stored)
	require.NoError(t, validVenues(w, venues))
	for name, tamper := range map[string]func(v *[]StaticVenue){
		"no venues": func(v *[]StaticVenue) { *v = nil },
		"account":   func(v *[]StaticVenue) { (*v)[0].Account = other },
		"irm":       func(v *[]StaticVenue) { (*v)[0].Irm = other },
		"oracle":    func(v *[]StaticVenue) { (*v)[0].Oracle = common.Address{} },
		"too many": func(v *[]StaticVenue) {
			for len(*v) <= maxVenues {
				*v = append(*v, (*v)[0])
			}
		},
	} {
		c := append([]StaticVenue(nil), venues...)
		tamper(&c)
		require.ErrorIs(t, validVenues(w, c), ErrInvalidProfile, name)
	}
	require.ErrorIs(t, validVenues(nil, venues), ErrInvalidProfile)
	_, err = validStatic(&se, "0xC0fdcb1799ccc2cebaa1fe247157b0df33d57572", DexType, tokens)
	require.ErrorIs(t, err, ErrInvalidProfile, "address must be lowercase")
	_, err = validStatic(&se, stored.Address, "other", tokens)
	require.ErrorIs(t, err, ErrInvalidProfile)
	_, err = validStatic(&se, stored.Address, DexType, []string{tokens[1], tokens[0]})
	require.ErrorIs(t, err, ErrInvalidProfile, "token order is (pool asset, loan asset)")
	// The arity guard is what keeps a malformed entity a refusal rather than a panic: every read below it indexes
	// tokens[0] and tokens[1]. Each identity clause is bound on its own by a list whose other entry is right, since
	// a swapped pair is refused by either of them.
	for name, tokens := range map[string][]string{
		"no tokens":                        {},
		"one token":                        {tokens[0]},
		"three tokens":                     {tokens[0], tokens[1], lowerHex(other)},
		"another token for the pool asset": {lowerHex(other), tokens[1]},
		"another token for the loan asset": {tokens[0], lowerHex(other)},
	} {
		_, err = validStatic(&se, stored.Address, DexType, tokens)
		require.ErrorIs(t, err, ErrInvalidProfile, name)
	}
	require.Equal(t, c104.HookSetHash, hookSetHash(se.Hooks))
}

// TestReadPlanErrorChain: a decoder's own error stays in the chain under the read's name, so a listing decoder that
// reports ErrInvalidProfile skips the pool instead of failing the run.
func TestReadPlanErrorChain(t *testing.T) {
	t.Parallel()
	p := &readPlan{}
	p.add("factory.implementation", common.Address{}, &factoryABI, "implementation", nil, func([]any) error {
		return fmt.Errorf("%w: venue 1 market pair", ErrInvalidProfile)
	})
	p.try("feed.pegOk", common.Address{}, &priceFeedABI, "pegOk", nil, func(vals []any, _ []byte) error {
		return fmt.Errorf("%w: peg", ErrInvalidProfile)
	})
	word := common.LeftPadBytes([]byte{1}, 32)
	for i, res := range []mcResult{{Ok: true, Data: word}, {Ok: true, Data: word}} {
		err := p.decs[i](res)
		require.ErrorIs(t, err, ErrInvalidProfile)
		require.ErrorIs(t, err, errReadDecode)
		require.True(t, chainAnswered(err))
		require.Contains(t, err.Error(), p.calls[i].Name)
	}
}
