// Package titan fetches pricing state overrides from Titan Builder's custom
// titan_getPammStateOverrides RPC method (docs.titanbuilder.xyz/propamms/takers),
// shared by every pAMM-style venue quoted through Titan (kipseli's pAMM
// variant, fermi, titan-prop, ...).
package titan

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
)

const DefaultTimeout = 10 * time.Second

// DefaultMergeWindow is how long FetchState keeps collecting other regions'
// responses after the first usable one arrives.
const DefaultMergeWindow = 300 * time.Millisecond

// beaconGenesisTS/secsPerSlot compute the canonical block.timestamp for a given
// beacon slot: genesis + slot*12. lambdaclass's SDK documents this as the value
// venues actually validate their pushed oracle timestamp against.
const (
	beaconGenesisTS uint64 = 1_606_824_023
	secsPerSlot     uint64 = 12
)

// Config points at Titan Builder's RPC.
type Config struct {
	URLs    []string      `json:"urls"`
	Timeout time.Duration `json:"timeout,omitempty"`
}

// StateOverrides is the persisted, JSON-friendly {address: {slot: value}}
// shape downstream Tenderly/gas-estimation tooling (~/bin/swap.zsh's
// poolExtra.so reader) replays a quote's state against.
type StateOverrides map[string]map[string]string

type State struct {
	Overrides      map[common.Address]gethclient.OverrideAccount
	BlockNumber    *big.Int
	BlockTimestamp uint64
}

type statePayload struct {
	StateOverride map[string]stateDiff `json:"stateOverride"`
}

type stateDiff struct {
	StateDiff map[string]string `json:"stateDiff"`
	Balance   string            `json:"balance"`
	Nonce     string            `json:"nonce"`
}

func NewClients(cfg Config) []*rpc.Client {
	clients := make([]*rpc.Client, 0, len(cfg.URLs))
	for _, url := range cfg.URLs {
		c, err := rpc.DialContext(context.Background(), url)
		if err != nil {
			logger.WithFields(logger.Fields{"url": url, "error": err.Error()}).Warn("titan RPC dial failed")
			continue
		}
		clients = append(clients, c)
	}
	return clients
}

// FetchState queries every client in parallel and merges their responses.
// Regions often disagree on which pAMM entries they carry, even for the same
// block, so after the first response with a pAMM entry arrives it waits up to
// DefaultMergeWindow for the others, then merges (see mergeResponses).
//
// Every venue's stateOverride entry is merged into one combined map rather
// than filtered to a single quoter address: the top-level stream key isn't
// reliably the venue's own call-target address, and every known pAMM's
// override ultimately lands on state the venue's own quote() call already
// reads, so applying the merged map works regardless of which entries belong
// to which venue.
//
// The override's freshness window is a few seconds at most — callers must
// use the result immediately, not cache it.
func FetchState(ctx context.Context, clients []*rpc.Client, timeout time.Duration) State {
	if len(clients) == 0 {
		return State{}
	}
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	results := make(chan map[string]json.RawMessage, len(clients))
	for _, client := range clients {
		go func() {
			var result map[string]json.RawMessage
			if err := client.CallContext(ctx, &result, "titan_getPammStateOverrides"); err != nil {
				// Canceled means the merge window already closed; not a failure.
				if !errors.Is(err, context.Canceled) {
					logger.WithFields(logger.Fields{"error": err.Error()}).Warn("titan RPC failed")
				}
				result = nil
			}
			results <- result
		}()
	}

	var responses []map[string]json.RawMessage
	var window <-chan time.Time
collect:
	for range clients {
		select {
		case result := <-results:
			if result == nil {
				continue
			}
			responses = append(responses, result)
			if window == nil && hasPammEntry(result) {
				window = time.After(DefaultMergeWindow)
			}
		case <-window:
			break collect
		}
	}

	state := buildState(mergeResponses(responses))
	if len(state.Overrides) == 0 {
		return State{}
	}
	return state
}

// mergeResponses keeps only the responses for the highest block number, then
// takes each pAMM entry whole from the first response that carries it. An
// entry's slots are never mixed across regions.
func mergeResponses(responses []map[string]json.RawMessage) map[string]json.RawMessage {
	var best *big.Int
	for _, r := range responses {
		if bn := parseBlockNumber(r); bn != nil && (best == nil || bn.Cmp(best) > 0) {
			best = bn
		}
	}
	var merged map[string]json.RawMessage
	for _, r := range responses {
		if bn := parseBlockNumber(r); best != nil && (bn == nil || bn.Cmp(best) != 0) {
			continue
		}
		if merged == nil {
			merged = make(map[string]json.RawMessage, len(r))
		}
		for k, v := range r {
			if _, ok := merged[k]; !ok {
				merged[k] = v
			}
		}
	}
	return merged
}

func hasPammEntry(result map[string]json.RawMessage) bool {
	for k := range result {
		if k != "slot" && k != "blockNumber" {
			return true
		}
	}
	return false
}

func buildState(result map[string]json.RawMessage) State {
	blockNumber := parseBlockNumber(result)

	overrides := mergeOverrides(result)
	if len(overrides) == 0 {
		return State{BlockNumber: blockNumber}
	}

	var canonicalTS uint64
	if slotRaw, ok := result["slot"]; ok {
		var slot uint64
		if err := json.Unmarshal(slotRaw, &slot); err == nil && slot > 0 {
			canonicalTS = beaconGenesisTS + slot*secsPerSlot
		}
	}
	if canonicalTS == 0 {
		// No (or an unparseable) "slot" field — fall back to the heuristic
		// rather than leaving an unpatched (likely stale) block.timestamp.
		canonicalTS = ExtractMaxOracleTimestamp(overrides)
	}

	return State{Overrides: overrides, BlockNumber: blockNumber, BlockTimestamp: canonicalTS}
}

func parseBlockNumber(result map[string]json.RawMessage) *big.Int {
	bnRaw, ok := result["blockNumber"]
	if !ok {
		return nil
	}
	var bnHex string
	if err := json.Unmarshal(bnRaw, &bnHex); err != nil {
		return nil
	}
	blockNumber, ok := new(big.Int).SetString(strings.TrimPrefix(bnHex, "0x"), 16)
	if !ok {
		return nil
	}
	return blockNumber
}

// mergeOverrides flattens every entry in a titan_getPammStateOverrides
// response (skipping the "slot"/"blockNumber" metadata keys) into one
// combined account -> stateDiff map, merging at the slot level when multiple
// entries touch the same account.
func mergeOverrides(result map[string]json.RawMessage) map[common.Address]gethclient.OverrideAccount {
	overrides := make(map[common.Address]gethclient.OverrideAccount)
	for key, raw := range result {
		if key == "slot" || key == "blockNumber" {
			continue
		}
		var payload statePayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			continue
		}
		for addrHex, sd := range payload.StateOverride {
			if len(sd.StateDiff) == 0 {
				continue
			}
			addr := common.HexToAddress(addrHex)
			acct, ok := overrides[addr]
			if !ok {
				acct = gethclient.OverrideAccount{
					StateDiff: make(map[common.Hash]common.Hash, len(sd.StateDiff)),
					Balance:   common.HexToHash(sd.Balance).Big(),
					Nonce:     common.HexToHash(sd.Nonce).Big().Uint64(),
				}
			}
			for slot, val := range sd.StateDiff {
				acct.StateDiff[common.HexToHash(slot)] = common.HexToHash(val)
			}
			overrides[addr] = acct
		}
	}
	return overrides
}

// ExtractMaxOracleTimestamp scans every stateDiff slot for a plausible-looking
// Unix timestamp in its first 4 bytes and takes the max, for use as the
// simulated block.timestamp override when a response carries no "slot" (or
// for overrides supplied directly by a caller, e.g. GetNewPoolStateWithOverrides,
// with no RPC response to read a slot from at all).
func ExtractMaxOracleTimestamp(overrides map[common.Address]gethclient.OverrideAccount) uint64 {
	const minTS uint32 = 1_700_000_000
	const maxTS uint32 = 2_000_000_000
	var maxFound uint32
	for _, acct := range overrides {
		for _, val := range acct.StateDiff {
			b := val.Bytes()
			if len(b) < 4 {
				continue
			}
			ts := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
			if ts < minTS || ts > maxTS {
				continue
			}
			if ts > maxFound {
				maxFound = ts
			}
		}
	}
	return uint64(maxFound)
}

// ToStateOverrides flattens the override set into the persisted, JSON-friendly
// {address: {slot: value}} shape (storage diffs only — balance/nonce aren't
// persisted, matching what every current consumer/reader of "so" expects).
func (s State) ToStateOverrides() StateOverrides {
	if len(s.Overrides) == 0 {
		return nil
	}
	out := make(StateOverrides, len(s.Overrides))
	for addr, acct := range s.Overrides {
		if len(acct.StateDiff) == 0 {
			continue
		}
		diff := make(map[string]string, len(acct.StateDiff))
		for slot, val := range acct.StateDiff {
			diff[slot.Hex()] = val.Hex()
		}
		out[hexutil.Encode(addr[:])] = diff
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
