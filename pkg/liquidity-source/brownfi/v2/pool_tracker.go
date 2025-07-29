package brownfiv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, ErrInvalidToken
	}
	startTime := time.Now()
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	var reserveData GetReservesResult
	var fee, lambda uint64
	var kappa, minPriceAge, oPrice0, oPrice1 *big.Int
	req := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodGetReserves,
	}, []any{&reserveData}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodFee,
	}, []any{&fee}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2FactoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryMethodMinPriceAge,
	}, []any{&minPriceAge})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	if p.BlockNumber > resp.BlockNumber.Uint64() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":           p.Address,
					"pool_block_number": p.BlockNumber,
					"data_block_number": resp.BlockNumber.Uint64(),
				},
			).
			Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	if _, err = d.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodLambda,
	}, []any{&lambda}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodKappa,
	}, []any{&kappa}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2FactoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryMethodPriceOf,
		Params: []any{common.HexToAddress(p.Tokens[0].Address), minPriceAge},
	}, []any{&oPrice0}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2FactoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryMethodPriceOf,
		Params: []any{common.HexToAddress(p.Tokens[1].Address), minPriceAge},
	}, []any{&oPrice1}).TryAggregate(); err != nil {
		return p, err
	}

	logger.
		WithFields(
			logger.Fields{
				"pool_id":          p.Address,
				"old_reserve":      p.Reserves,
				"new_reserve":      reserveData,
				"old_block_number": p.BlockNumber,
				"new_block_number": resp.BlockNumber.Uint64(),
				"duration_ms":      time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	extra := Extra{
		Fee:     fee,
		Lambda:  lambda,
		Kappa:   uint256.MustFromBig(kappa),
		OPrices: [2]*uint256.Int{uint256.MustFromBig(oPrice0), uint256.MustFromBig(oPrice1)},
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{
		reserveData.Reserve0.String(),
		reserveData.Reserve1.String(),
	}
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	return p, nil
}
