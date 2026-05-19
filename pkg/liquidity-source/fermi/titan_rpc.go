package fermi

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
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
			logger.WithFields(logger.Fields{"url": url, "error": err.Error()}).
				Warn("fermi: titan RPC dial failed")
			continue
		}
		clients = append(clients, c)
	}
	return clients
}

func (t *PoolTracker) fetchStateOverrides(ctx context.Context) map[common.Address]gethclient.OverrideAccount {
	if len(t.titanClients) == 0 {
		return nil
	}
	timeout := t.config.Titan.Timeout
	if timeout == 0 {
		timeout = titanDefaultTimeout
	}

	for _, client := range t.titanClients {
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		overrides, err := t.doTitanRPC(callCtx, client)
		cancel()
		if err != nil {
			logger.WithFields(logger.Fields{"error": err.Error()}).
				Warn("fermi: titan RPC failed, trying next")
			continue
		}
		return overrides
	}
	return nil
}

func (t *PoolTracker) doTitanRPC(
	ctx context.Context,
	client *rpc.Client,
) (map[common.Address]gethclient.OverrideAccount, error) {
	var result map[string]json.RawMessage
	if err := client.CallContext(ctx, &result, "titan_getPammStateOverrides"); err != nil {
		return nil, err
	}

	quoterKey := strings.ToLower(common.HexToAddress(t.config.FermiSwapper).Hex())

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
