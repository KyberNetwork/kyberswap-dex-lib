package llamma

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	// "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client

	logger logger.Logger
}

var _ = pooltrack.RegisterFactoryCEG0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolTracker {
	return &PoolTracker{
		config:        config,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
		logger: logger.WithFields(logger.Fields{
			"dexId":   config.DexID,
			"dexType": DexType,
		}),
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := t.logger.WithFields(logger.Fields{"poolAddress": p.Address})

	rpcState, err := t.fetchRPCState(ctx, p)
	if err != nil {
		lg.Error("failed to fetch state from RPC")
	}

	extraBytes, err := json.Marshal(&Extra{
		Fee:         uint256.MustFromBig(rpcState.Fee),
		PriceOracle: uint256.MustFromBig(rpcState.PriceOracle),
		ActiveBand:  uint256.MustFromBig(rpcState.ActiveBand),
		MinBand:     uint256.MustFromBig(rpcState.MinBand),
		MaxBand:     uint256.MustFromBig(rpcState.MaxBand),
		BandsX: lo.MapEntries(rpcState.BandsX, func(k int64, v *big.Int) (int64, *uint256.Int) {
			return k, uint256.MustFromBig(v)
		}),
		BandsY: lo.MapEntries(rpcState.BandsY, func(k int64, v *big.Int) (int64, *uint256.Int) {
			return k, uint256.MustFromBig(v)
		}),
	})
	if err != nil {
		lg.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		rpcState.CollateralReserves.String(),
		rpcState.StableCoinReserves.String(),
	}

	var blockNumber uint64
	if rpcState.BlockNumber != nil {
		blockNumber = rpcState.BlockNumber.Uint64()
	}
	p.BlockNumber = blockNumber

	lg.Info("Finish updating state of pool")

	return p, nil
}

func (t *PoolTracker) fetchRPCState(ctx context.Context, p entity.Pool) (FetchRPCResult, error) {
	var (
		collateralReserve, stableCoinReserve  *big.Int
		fee, adminFee, adminFeesX, adminFeesY *big.Int
		priceOracle                           *big.Int
		activeBand, minBand, maxBand          *big.Int
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    llammaABI,
		Target: p.Address,
		Method: llammaMethodFee,
	}, []any{&fee})

	calls.AddCall(&ethrpc.Call{
		ABI:    llammaABI,
		Target: p.Address,
		Method: llammaMethodAdminFee,
	}, []any{&adminFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    llammaABI,
		Target: p.Address,
		Method: llammaMethodAdminFeesX,
	}, []any{&adminFeesX})

	calls.AddCall(&ethrpc.Call{
		ABI:    llammaABI,
		Target: p.Address,
		Method: llammaMethodAdminFeesY,
	}, []any{&adminFeesY})

	calls.AddCall(&ethrpc.Call{
		ABI:    llammaABI,
		Target: p.Address,
		Method: llammaMethodPriceOracle,
	}, []any{&priceOracle})

	calls.AddCall(&ethrpc.Call{
		ABI:    llammaABI,
		Target: p.Address,
		Method: llammaMethodActiveBand,
	}, []any{&activeBand})

	calls.AddCall(&ethrpc.Call{
		ABI:    llammaABI,
		Target: p.Address,
		Method: llammaMethodMinBand,
	}, []any{&minBand})

	calls.AddCall(&ethrpc.Call{
		ABI:    llammaABI,
		Target: p.Address,
		Method: llammaMethodMaxBand,
	}, []any{&maxBand})

	calls.AddCall(&ethrpc.Call{
		ABI:    shared.ERC20ABI,
		Target: p.Tokens[0].Address,
		Method: shared.ERC20MethodBalanceOf,
		Params: []interface{}{common.HexToAddress(p.Address)},
	}, []any{&collateralReserve})

	calls.AddCall(&ethrpc.Call{
		ABI:    shared.ERC20ABI,
		Target: p.Tokens[1].Address,
		Method: shared.ERC20MethodBalanceOf,
		Params: []interface{}{common.HexToAddress(p.Address)},
	}, []any{&stableCoinReserve})
	if _, err := calls.Aggregate(); err != nil {
		return FetchRPCResult{}, err
	}

	bandsX := make(map[int64]*big.Int, maxBand.Int64()-minBand.Int64()+1)
	bandsY := make(map[int64]*big.Int, maxBand.Int64()-minBand.Int64()+1)

	// TODO: should we use multicall here?
	bandCalls := t.ethrpcClient.NewRequest().SetContext(ctx)
	for i := minBand.Int64(); i <= maxBand.Int64(); i += 1 {
		var bandX, bandY big.Int
		bandsX[i] = &bandX
		bandsY[i] = &bandY
		bandCalls.AddCall(&ethrpc.Call{
			ABI:    llammaABI,
			Target: p.Address,
			Method: llammaMethodBandsX,
			Params: []any{big.NewInt(i)},
		}, []any{&bandX})
		bandCalls.AddCall(&ethrpc.Call{
			ABI:    llammaABI,
			Target: p.Address,
			Method: llammaMethodBandsY,
			Params: []any{big.NewInt(i)},
		}, []any{&bandY})
	}
	_, err := bandCalls.Aggregate()
	if err != nil {
		return FetchRPCResult{}, err
	}

	collateralReserve.Sub(collateralReserve, adminFeesX)
	stableCoinReserve.Sub(stableCoinReserve, adminFeesY)

	return FetchRPCResult{
		Fee:                fee,
		AdminFee:           adminFee,
		PriceOracle:        priceOracle,
		ActiveBand:         activeBand,
		MinBand:            minBand,
		MaxBand:            maxBand,
		BandsX:             bandsX,
		BandsY:             bandsY,
		CollateralReserves: collateralReserve,
		StableCoinReserves: stableCoinReserve,
		BlockNumber:        calls.BlockNumber,
	}, nil
}
