package llamma

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
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

func (t *PoolTracker) FetchStateFromRPC(ctx context.Context, pool entity.Pool, blockNumber uint64) ([]byte, error) {
	// TODO implement me
	panic("implement me")
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

	rpcState, err := t.fetchRPCState(ctx, p, new(big.Int).SetUint64(22014994))
	if err != nil {
		lg.Error("failed to fetch state from RPC")
		return p, err
	}

	extraBytes, err := json.Marshal(&Extra{
		Fee:         uint256.MustFromBig(rpcState.Fee),
		AdminFee:    uint256.MustFromBig(rpcState.AdminFee),
		AdminFeesX:  uint256.MustFromBig(rpcState.AdminFeesX),
		AdminFeesY:  uint256.MustFromBig(rpcState.AdminFeesY),
		BasePrice:   uint256.MustFromBig(rpcState.BasePrice),
		PriceOracle: uint256.MustFromBig(rpcState.PriceOracle),
		ActiveBand:  int256.MustFromBig(rpcState.ActiveBand),
		MinBand:     int256.MustFromBig(rpcState.MinBand),
		MaxBand:     int256.MustFromBig(rpcState.MaxBand),
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

func (t *PoolTracker) fetchRPCState(ctx context.Context, p entity.Pool, blockNumber *big.Int) (FetchRPCResult, error) {
	var (
		collateralReserve, stableCoinReserve  *big.Int
		fee, adminFee, adminFeesX, adminFeesY *big.Int
		basePrice, priceOracle                *big.Int
		activeBand, minBand, maxBand          *big.Int
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		calls.SetBlockNumber(blockNumber)
	}
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
		Method: llammaMethodGetBasePrice,
	}, []any{&basePrice})
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

	bandsX := make([]*big.Int, maxBand.Int64()-minBand.Int64()+1)
	bandsY := make([]*big.Int, maxBand.Int64()-minBand.Int64()+1)

	// TODO: should we use multicall here?
	bandCalls := t.ethrpcClient.NewRequest().SetContext(ctx)
	for i := minBand.Int64(); i <= maxBand.Int64(); i += 1 {
		bandCalls.AddCall(&ethrpc.Call{
			ABI:    llammaABI,
			Target: p.Address,
			Method: llammaMethodBandsX,
			Params: []any{big.NewInt(i)},
		}, []any{&bandsX[i-minBand.Int64()]})
		bandCalls.AddCall(&ethrpc.Call{
			ABI:    llammaABI,
			Target: p.Address,
			Method: llammaMethodBandsY,
			Params: []any{big.NewInt(i)},
		}, []any{&bandsY[i-minBand.Int64()]})
	}
	_, err := bandCalls.Aggregate()
	if err != nil {
		return FetchRPCResult{}, err
	}

	collateralReserve.Sub(collateralReserve, adminFeesX)
	stableCoinReserve.Sub(stableCoinReserve, adminFeesY)

	fmt.Println(p.Address, "minBand: ", minBand, "maxBand: ", maxBand, activeBand)
	fmt.Println("balance of collateral: ", collateralReserve, "adminFeesX: ", adminFeesX)
	fmt.Println("balance of stable coin: ", stableCoinReserve, "adminFeesY: ", adminFeesY)
	totalBandx := big.NewInt(0)
	totalBandy := big.NewInt(0)
	for i := minBand.Int64(); i <= maxBand.Int64(); i += 1 {
		totalBandx.Add(totalBandx, bandsX[i-minBand.Int64()])
		totalBandy.Add(totalBandy, bandsY[i-minBand.Int64()])
	}
	fmt.Println("totalBandx: ", totalBandx)
	fmt.Println("totalBandy: ", totalBandy)

	return FetchRPCResult{
		Fee:         fee,
		AdminFee:    adminFee,
		AdminFeesX:  adminFeesX,
		AdminFeesY:  adminFeesY,
		BasePrice:   basePrice,
		PriceOracle: priceOracle,
		ActiveBand:  activeBand,
		MinBand:     minBand,
		MaxBand:     maxBand,
		BandsX: sliceToMapWithIndex(bandsX, func(i int, v *big.Int) (int64, *big.Int) {
			return int64(i) + minBand.Int64(), v
		}),
		BandsY: sliceToMapWithIndex(bandsY, func(i int, v *big.Int) (int64, *big.Int) {
			return int64(i) + minBand.Int64(), v
		}),
		CollateralReserves: collateralReserve,
		StableCoinReserves: stableCoinReserve,
		BlockNumber:        calls.BlockNumber,
	}, nil
}

func sliceToMapWithIndex[T any, K comparable, V any](
	slice []T,
	iteratee func(i int, item T) (K, V),
) map[K]V {
	out := make(map[K]V, len(slice))
	for i, e := range slice {
		k, v := iteratee(i, e)
		out[k] = v
	}
	return out
}
