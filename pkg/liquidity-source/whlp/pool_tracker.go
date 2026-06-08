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
	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
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

	var (
		rateInQuote     *big.Int
		accountantState struct {
			PayoutAddress                  common.Address
			FeesOwedInBase                 *big.Int
			TotalSharesLastUpdate          *big.Int
			ExchangeRate                   *big.Int
			AllowedExchangeRateChangeUpper uint16
			AllowedExchangeRateChangeLower uint16
			LastUpdateTimestamp            uint64
			IsPaused                       bool
			MinimumUpdateDelayInSeconds    uint32
			ManagementFee                  uint16
		}
	)

	req.AddCall(&ethrpc.Call{
		ABI:    accountantABI,
		Target: staticExtra.Accountant.Hex(),
		Method: "getRateInQuoteSafe",
		Params: []any{staticExtra.QuoteAsset},
	}, []any{&rateInQuote})

	req.AddCall(&ethrpc.Call{
		ABI:    accountantABI,
		Target: staticExtra.Accountant.Hex(),
		Method: "accountantState",
		Params: []any{},
	}, []any{&accountantState})

	if _, err := req.TryAggregate(); err != nil {
		return p, err
	}

	extra := Extra{
		RateInQuote:        rateInQuote,
		IsAccountantPaused: accountantState.IsPaused,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	return p, nil
}
