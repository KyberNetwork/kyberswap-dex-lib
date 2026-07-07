package pamm

import (
	"context"
	"encoding/binary"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kipseli"
)

const titanDefaultTimeout = 10 * time.Second

type TitanConfig struct {
	URLs    []string      `json:"urls"`
	Timeout time.Duration `json:"timeout,omitempty"`
}

type titanQuoterPayload struct {
	StateOverride map[string]titanStateDiff `json:"stateOverride"`
}

type titanStateDiff struct {
	StateDiff map[string]string `json:"stateDiff"`
	Balance   string            `json:"balance"`
	Nonce     string            `json:"nonce"`
}

type titanPammState struct {
	Overrides      map[common.Address]gethclient.OverrideAccount
	BlockTimestamp uint64
}

func newTitanClients(cfg TitanConfig) []*rpc.Client {
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

// fetchStateOverrides returns Titan pricing overrides keyed by RouterAddress,
// plus the PUR update timestamp required for block timestamp override.
func (t *PoolTracker) fetchStateOverrides(ctx context.Context) titanPammState {
	if len(t.titanClients) == 0 {
		return titanPammState{}
	}
	timeout := t.cfg.Titan.Timeout
	if timeout == 0 {
		timeout = titanDefaultTimeout
	}

	var fallback titanPammState
	for _, client := range t.titanClients {
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		state, err := t.doTitanRPC(callCtx, client)
		cancel()
		if err != nil {
			logger.WithFields(logger.Fields{"error": err.Error()}).Warn("titan RPC failed, trying next")
			continue
		}
		if len(state.Overrides) == 0 {
			continue
		}
		if state.BlockTimestamp != 0 {
			return state
		}
		if len(fallback.Overrides) == 0 {
			fallback = state
		}
		logger.WithFields(logger.Fields{
			"target":   t.cfg.LensAddress,
			"registry": t.cfg.PriorityUpdateRegistry,
			"lane":     priorityUpdateLaneIndex,
		}).Warn("titan PAMM overrides missing priority update timestamp, trying next")
	}
	return fallback
}

func (t *PoolTracker) doTitanRPC(ctx context.Context, client *rpc.Client) (titanPammState, error) {
	var result map[string]json.RawMessage
	if err := client.CallContext(ctx, &result, "titan_getPammStateOverrides"); err != nil {
		return titanPammState{}, err
	}

	quoterKey := strings.ToLower(common.HexToAddress(t.cfg.RouterAddress).Hex())
	var quoterRaw json.RawMessage
	for k, v := range result {
		if strings.EqualFold(common.HexToAddress(k).Hex(), quoterKey) {
			quoterRaw = v
			break
		}
	}
	if quoterRaw == nil {
		return titanPammState{}, nil
	}

	var payload titanQuoterPayload
	if err := json.Unmarshal(quoterRaw, &payload); err != nil || len(payload.StateOverride) == 0 {
		return titanPammState{}, nil
	}

	overrides := make(map[common.Address]gethclient.OverrideAccount, len(payload.StateOverride))
	for addrHex, sd := range payload.StateOverride {
		diff := make(map[common.Hash]common.Hash, len(sd.StateDiff))
		for slot, val := range sd.StateDiff {
			diff[common.HexToHash(slot)] = common.HexToHash(val)
		}
		overrides[common.HexToAddress(addrHex)] = gethclient.OverrideAccount{
			StateDiff: diff,
			Balance:   common.HexToHash(sd.Balance).Big(),
			Nonce:     common.HexToHash(sd.Nonce).Big().Uint64(),
		}
	}

	if len(overrides) == 0 {
		return titanPammState{}, nil
	}
	return titanPammState{
		Overrides:      overrides,
		BlockTimestamp: extractPammBlockTimestamp(overrides, t.cfg),
	}, nil
}

func extractPammBlockTimestamp(overrides map[common.Address]gethclient.OverrideAccount, cfg *Config) uint64 {
	registryState, ok := overrides[common.HexToAddress(cfg.PriorityUpdateRegistry)]
	if !ok {
		return 0
	}
	slot0, ok := registryState.StateDiff[priorityUpdateLaneSlot(common.HexToAddress(cfg.LensAddress), priorityUpdateLaneIndex)]
	if !ok {
		return 0
	}
	return uint64(binary.BigEndian.Uint32(slot0[:4]))
}

func priorityUpdateLaneSlot(target common.Address, laneIndex uint64) common.Hash {
	var encoded [64]byte
	copy(encoded[12:32], target.Bytes())
	binary.BigEndian.PutUint64(encoded[56:64], laneIndex)
	return crypto.Keccak256Hash(encoded[:])
}

// titanOverridesToMap serializes the full quoter override set (storage +
// balance + nonce) into Extra.SO. Every contract under the quoter is forwarded
// so downstream simulation reproduces the exact Titan state, including
// balance/nonce-only entries (e.g. funded EOAs).
func titanOverridesToMap(overrides map[common.Address]gethclient.OverrideAccount) map[string]kipseli.StateOverride {
	out := make(map[string]kipseli.StateOverride, len(overrides))
	for addr, acct := range overrides {
		entry := kipseli.StateOverride{}
		if len(acct.StateDiff) > 0 {
			entry.Storage = make(map[string]string, len(acct.StateDiff))
			for slot, val := range acct.StateDiff {
				entry.Storage[slot.Hex()] = val.Hex()
			}
		}
		if acct.Balance != nil && acct.Balance.Sign() != 0 {
			entry.Balance = "0x" + acct.Balance.Text(16)
		}
		if acct.Nonce != 0 {
			entry.Nonce = "0x" + strconv.FormatUint(acct.Nonce, 16)
		}
		if entry.Storage == nil && entry.Balance == "" && entry.Nonce == "" {
			continue
		}
		out[strings.ToLower(addr.Hex())] = entry
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
