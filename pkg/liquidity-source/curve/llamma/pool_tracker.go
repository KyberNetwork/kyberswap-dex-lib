package llamma

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
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

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
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

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var (
		fee, adminFeesX, adminFeesY *big.Int
		basePrice, priceW           *big.Int
		dataBytes                   []byte

		collateralReserve *big.Int
		stableCoinReserve *big.Int
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(new(big.Int).SetUint64(22019234))
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodFee,
	}, []any{&fee})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodAdminFeesX,
	}, []any{&adminFeesX})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodAdminFeesY,
	}, []any{&adminFeesY})
	calls.AddCall(&ethrpc.Call{
		ABI:    curvePriceOracleABI,
		Target: staticExtra.PriceOracleAddress,
		Method: priceOracleMethodPriceW,
	}, []any{&priceW})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodGetBasePrice,
	}, []any{&basePrice})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaHelperABI,
		Target: t.config.HelperAddress,
		Method: curveLlammaHelperMethodGet,
		Params: []any{common.HexToAddress(p.Address)},
	}, []any{&dataBytes})
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
	resp, err := calls.Aggregate()
	if err != nil {
		return p, err
	}

	curveLlammaResult, err := decode(dataBytes)
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(&Extra{
		BasePrice:   uint256.MustFromBig(basePrice),
		Fee:         uint256.MustFromBig(fee),
		AdminFeesX:  uint256.MustFromBig(adminFeesX),
		AdminFeesY:  uint256.MustFromBig(adminFeesY),
		PriceOracle: uint256.MustFromBig(priceW),
		AdminFee:    curveLlammaResult.AdminFee,
		DynamicFee:  curveLlammaResult.DynamicFee,
		ActiveBand:  curveLlammaResult.ActiveBand,
		MinBand:     curveLlammaResult.MinBand,
		MaxBand:     curveLlammaResult.MaxBand,
		Bands:       curveLlammaResult.Bands,
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
		collateralReserve.Sub(collateralReserve, adminFeesX).String(),
		stableCoinReserve.Sub(stableCoinReserve, adminFeesY).String(),
	}

	var blockNumber uint64
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}
	p.BlockNumber = blockNumber

	lg.Info("Finish updating state of pool")

	return p, nil
}
