package thogprop

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, client *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: cfg, ethrpcClient: client}, nil
}

type snapshotHeader struct {
	Number     *hexutil.Big `json:"number"`
	Hash       common.Hash  `json:"hash"`
	ParentHash common.Hash  `json:"parentHash"`
	BaseFee    *hexutil.Big `json:"baseFeePerGas"`
}

// The pool image and issuer reads share one canonical block hash. A reorg or
// failed issuer read fails the refresh rather than mixing different states.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	client := t.ethrpcClient.GetETHClient().Client()
	var header *snapshotHeader
	if err := client.CallContext(ctx, &header, "eth_getBlockByNumber", "latest", false); err != nil {
		return p, err
	}
	if header == nil || header.Number == nil || !(*big.Int)(header.Number).IsUint64() || header.BaseFee == nil || header.Hash == (common.Hash{}) {
		return p, ErrInvalidState
	}
	block := rpc.BlockNumberOrHashWithHash(header.Hash, true)
	data, err := thogAMMABI.Pack(methodGetPoolData, uint8(0), uint8(maxTokens))
	if err != nil {
		return p, err
	}
	address := common.HexToAddress(p.Address)
	// Nonzero simulation gasPrice preserves BASEFEE. It is unrelated to the
	// eventual swap's effective transaction gas price.
	call := func(to common.Address, data []byte) map[string]any {
		return map[string]any{"to": to, "data": hexutil.Bytes(data), "gasPrice": "0x1"}
	}
	var raw hexutil.Bytes
	if err = client.CallContext(ctx, &raw, "eth_call", call(address, data), block); err != nil {
		return p, err
	}
	snapshot, err := decodeSnapshot(raw)
	if err != nil {
		return p, err
	}
	if snapshot.BlockNumber == nil || snapshot.BaseFeePerGas == nil || snapshot.BlockNumber.Cmp((*big.Int)(header.Number)) != 0 || snapshot.BaseFeePerGas.Cmp((*big.Int)(header.BaseFee)) != 0 || snapshot.ParentBlockHash != [32]byte(header.ParentHash) {
		return p, ErrInvalidState
	}
	var issuers []common.Address
	for _, token := range snapshot.Tokens {
		if isXStock(hexutil.Encode(token.Token[:])) {
			issuers = append(issuers, token.Token)
		}
	}
	sharesData, err := thogAMMABI.Pack("sharesOf", address)
	if err != nil {
		return p, err
	}
	multiplierData, err := thogAMMABI.Pack("getCurrentMultiplier")
	if err != nil {
		return p, err
	}
	results := make([]hexutil.Bytes, 2*len(issuers))
	batch := make([]rpc.BatchElem, len(results))
	for i, issuer := range issuers {
		for j, calldata := range [][]byte{sharesData, multiplierData} {
			index := 2*i + j
			batch[index] = rpc.BatchElem{Method: "eth_call", Args: []any{call(issuer, calldata), block}, Result: &results[index]}
		}
	}
	if len(batch) > 0 {
		if err = client.BatchCallContext(ctx, batch); err != nil {
			return p, err
		}
	}
	shares := make(map[string]ShareBalance, len(issuers))
	for i, issuer := range issuers {
		for j := range 2 {
			if batch[2*i+j].Error != nil {
				return p, fmt.Errorf("thog-prop: issuer %s: %w", issuer, batch[2*i+j].Error)
			}
		}
		if len(results[2*i]) != 32 || len(results[2*i+1]) != 96 {
			return p, ErrInvalidState
		}
		var holding ShareBalance
		holding.Shares.SetBytes(results[2*i])
		holding.Multiplier.SetBytes(results[2*i+1][:32])
		shares[hexutil.Encode(issuer[:])] = holding
	}
	state, err := snapshotState(snapshot, shares)
	if err != nil {
		return p, err
	}
	extra, err := json.Marshal(&Extra{State: state})
	if err != nil {
		return p, err
	}
	tokens := make([]*entity.PoolToken, len(state.Tokens))
	reserves := make(entity.PoolReserves, len(tokens))
	for i, token := range state.Tokens {
		tokens[i] = &entity.PoolToken{Address: token.Address, Decimals: token.Decimals, Swappable: true}
		reserves[i] = state.Balances[i].Dec()
	}
	p.Tokens = tokens
	p.Reserves = reserves
	p.Extra = string(extra)
	p.BlockNumber = state.BlockNumber
	p.Timestamp = time.Now().Unix()
	return p, nil
}

// Refresh even when no publication or transfer event arrives. Price age advances
// with the snapshot block and must not freeze when the maker stops posting.
// The pool-service schedule is in whole Unix seconds; head-driven refreshes can
// run sooner. Tracker failures remain explicit errors for the caller to handle.
func (t *PoolTracker) NextUpdateAt(p entity.Pool) int64 { return p.Timestamp + 1 }

var _ pool.IPoolTrackerWithSchedule = (*PoolTracker)(nil)
