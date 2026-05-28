package pamm

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"

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

// fetchStateOverrides returns Titan pricing overrides keyed by RouterAddress.
// Returns nil on no configured clients or on full failure (caller falls back to
// the balance-only override).
func (t *PoolTracker) fetchStateOverrides(ctx context.Context) map[common.Address]gethclient.OverrideAccount {
	if len(t.titanClients) == 0 {
		return nil
	}
	timeout := t.cfg.Titan.Timeout
	if timeout == 0 {
		timeout = titanDefaultTimeout
	}

	for _, client := range t.titanClients {
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		overrides, err := t.doTitanRPC(callCtx, client)
		cancel()
		if err != nil {
			logger.WithFields(logger.Fields{"error": err.Error()}).Warn("titan RPC failed, trying next")
			continue
		}
		return overrides
	}
	return nil
}

func (t *PoolTracker) doTitanRPC(ctx context.Context, client *rpc.Client) (map[common.Address]gethclient.OverrideAccount, error) {
	var result map[string]json.RawMessage
	if err := client.CallContext(ctx, &result, "titan_getPammStateOverrides"); err != nil {
		return nil, err
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
		return nil, nil
	}

	var payload titanQuoterPayload
	if err := json.Unmarshal(quoterRaw, &payload); err != nil || len(payload.StateOverride) == 0 {
		return nil, nil
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
		return nil, nil
	}
	return overrides, nil
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
