package ramses

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
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

var _ = pooltrack.RegisterFactoryCE(DexTypeRamses, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var (
		reserve                         Reserves
		stableFee, volatileFee, pairFee *big.Int
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: poolMethodGetReserves,
	}, []any{&reserve})

	calls.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: poolMethodStableFee,
	}, []any{&stableFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: poolMethodVolatileFee,
	}, []any{&volatileFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: poolMethodPairFee,
		Params: []any{common.HexToAddress(p.Address)},
	}, []any{&pairFee})

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to aggregate to get pool data")

		return entity.Pool{}, err
	}

	staticExtra, err := extractStaticExtra(p.StaticExtra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to extract static extra")

		return entity.Pool{}, err
	}

	var swapFee = pairFee.Int64()
	if pairFee.Int64() == 0 {
		swapFee = volatileFee.Int64()
		if staticExtra.Stable {
			swapFee = stableFee.Int64()
		}
	}

	p.Reserves = entity.PoolReserves{reserve.Reserve0.String(), reserve.Reserve1.String()}
	p.SwapFee = float64(swapFee) / bps
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func extractStaticExtra(s string) (staticExtra StaticExtra, err error) {
	err = json.Unmarshal([]byte(s), &staticExtra)

	return
}
