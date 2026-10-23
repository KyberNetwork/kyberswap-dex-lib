package thogprop

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
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

// One atomic call returns the full registry, portfolio, native balance, block
// context and all 126 packed words. No hard-coded token order is retained.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	data, err := thogAMMABI.Pack(methodGetPoolData, uint8(0), uint8(32))
	if err != nil {
		return p, err
	}
	address := common.HexToAddress(p.Address)
	// An explicit nonzero gasPrice preserves BASEFEE in eth_call implementations
	// that zero it for fee-free simulations. This does not submit a transaction.
	raw, err := t.ethrpcClient.GetETHClient().CallContract(ctx, ethereum.CallMsg{To: &address, Data: data, GasPrice: big.NewInt(1)}, nil)
	if err != nil {
		return p, err
	}
	snapshot, err := decodeSnapshot(raw)
	if err != nil {
		return p, err
	}
	state, err := snapshotState(snapshot)
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
