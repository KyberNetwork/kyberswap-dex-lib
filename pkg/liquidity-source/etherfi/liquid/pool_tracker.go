package liquid

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
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}
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

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	// Fetch teller related data
	var (
		accountant     common.Address
		assetData      = make([]Asset, len(p.Tokens)-1)
		isTellerPaused bool
	)

	req.AddCall(&ethrpc.Call{
		ABI:    tellerABI,
		Target: staticExtra.Teller.String(),
		Method: "accountant",
		Params: []any{},
	}, []any{&accountant})

	req.AddCall(&ethrpc.Call{
		ABI:    tellerABI,
		Target: staticExtra.Teller.String(),
		Method: "isPaused",
		Params: []any{},
	}, []any{&isTellerPaused})

	for i, token := range p.Tokens[1:] {
		req.AddCall(&ethrpc.Call{
			ABI:    tellerABI,
			Target: staticExtra.Teller.String(),
			Method: "assetData",
			Params: []any{common.HexToAddress(token.Address)},
		}, []any{&assetData[i]})
	}

	if _, err := req.TryAggregate(); err != nil {
		return p, err
	}

	// Fetch accountant related data
	var rateInQuote = make([]*big.Int, len(p.Tokens)-1)

	req = t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	for i, token := range p.Tokens[1:] {
		req.AddCall(&ethrpc.Call{
			ABI:    accountantABI,
			Target: accountant.String(),
			Method: "getRateInQuoteSafe",
			Params: []any{common.HexToAddress(token.Address)},
		}, []any{&rateInQuote[i]})
	}

	if _, err := req.TryAggregate(); err != nil {
		return p, err
	}

	// Update pool extra data
	extra := Extra{
		IsTellerPaused: isTellerPaused,
		AssetData:      assetData,
		RateInQuote:    rateInQuote,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}
