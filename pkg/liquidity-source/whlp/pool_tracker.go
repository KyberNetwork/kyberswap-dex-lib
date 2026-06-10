package whlp

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryE0(DexType, NewPoolTracker)

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var rateInQuote *big.Int

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    accountantABI,
		Target: staticExtra.Accountant.Hex(),
		Method: "getRateInQuoteSafe",
		Params: []any{usdt0Address},
	}, []any{&rateInQuote})

	if _, err := req.TryAggregate(); err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(Extra{RateInQuote: rateInQuote})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}
