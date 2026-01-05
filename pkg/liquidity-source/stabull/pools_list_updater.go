package stabull

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	// For now, we use a simple initialization approach
	// TODO: Implement event-based pool discovery for NewCurve events
	// This requires scanning blockchain logs for newCurveTopic = "0xe7a19de9e8788cc07c144818f2945144acd6234f790b541aa1010371c8b2a73b"

	if d.hasInitialized {
		logger.WithFields(logger.Fields{
			"dex": DexType,
		}).Debug("skip since pools have been initialized")
		return nil, nil, nil
	}

	// Use configured pool addresses for initial implementation
	pools, err := d.initPools(ctx)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dex":   DexType,
			"error": err,
		}).Error("failed to initialize pools")
		return nil, nil, err
	}

	d.hasInitialized = true

	logger.WithFields(logger.Fields{
		"dex":   DexType,
		"pools": len(pools),
	}).Info("finished fetching pools")

	return pools, nil, nil
}

func (d *PoolsListUpdater) initPools(ctx context.Context) ([]entity.Pool, error) {
	// TODO: For production, scan NewCurve events from CurveFactory
	// For now, require pool addresses to be configured in Config

	if len(d.config.PoolAddresses) == 0 {
		logger.WithFields(logger.Fields{
			"dex": DexType,
		}).Warn("No pool addresses configured")
		return []entity.Pool{}, nil
	}

	var pools []entity.Pool
	for _, poolAddress := range d.config.PoolAddresses {
		pool, err := d.getNewPool(ctx, poolAddress)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dex":         DexType,
				"poolAddress": poolAddress,
				"error":       err,
			}).Warn("failed to fetch pool")
			continue
		}
		pools = append(pools, *pool)
	}

	return pools, nil
}

func (d *PoolsListUpdater) getNewPool(ctx context.Context, poolAddress string) (*entity.Pool, error) {
	var (
		token0Address   common.Address
		token1Address   common.Address
		liquidityResult struct {
			Total      *big.Int   `json:"total_"`
			Individual []*big.Int `json:"individual_"`
		}
		curveResult struct {
			Alpha   *big.Int `json:"alpha_"`
			Beta    *big.Int `json:"beta_"`
			Delta   *big.Int `json:"delta_"`
			Epsilon *big.Int `json:"epsilon_"`
			Lambda  *big.Int `json:"lambda_"`
		}
	)

	rpcRequest := d.ethrpcClient.NewRequest().SetContext(ctx)

	// Fetch token0 address (base token) using numeraires(0)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodNumeraires,
		Params: []interface{}{big.NewInt(0)},
	}, []interface{}{&token0Address})

	// Fetch token1 address (USDC) using numeraires(1)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodNumeraires,
		Params: []interface{}{big.NewInt(1)},
	}, []interface{}{&token1Address})

	// Fetch reserves using liquidity() method
	// liquidity() returns (uint256 total_, uint256[] individual_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodLiquidity,
		Params: []interface{}{},
	}, []interface{}{&liquidityResult.Total, &liquidityResult.Individual})

	// Fetch curve parameters (alpha, beta, delta, epsilon, lambda)
	// viewCurve() returns (uint256 alpha_, uint256 beta_, uint256 delta_, uint256 epsilon_, uint256 lambda_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodViewCurve,
		Params: []interface{}{},
	}, []interface{}{&curveResult.Alpha, &curveResult.Beta, &curveResult.Delta, &curveResult.Epsilon, &curveResult.Lambda})

	_, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, err
	}

	// Build curve parameters (convert to strings for JSON)
	curveParams := CurveParameters{
		Alpha:   curveResult.Alpha.String(),
		Beta:    curveResult.Beta.String(),
		Delta:   curveResult.Delta.String(),
		Epsilon: curveResult.Epsilon.String(),
		Lambda:  curveResult.Lambda.String(),
	}

	// TODO: Optionally fetch token metadata (symbol, decimals) using ERC20 calls
	// For now, we know token1 is always USDC (6 decimals)

	extra := Extra{
		CurveParams:     curveParams,
		BaseOracleRate:  "", // Oracle rates built into viewOriginSwap
		QuoteOracleRate: "", // Not needed for simulation
		OracleRate:      "",
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}

	// Convert reserves to strings
	reserves := make([]string, len(liquidityResult.Individual))
	for i, reserve := range liquidityResult.Individual {
		reserves[i] = reserve.String()
	}

	// TODO: Optionally fetch actual token symbols and decimals
	// For now using placeholders (we know token1 is always USDC)
	tokens := []*entity.PoolToken{
		{
			Address:   strings.ToLower(token0Address.Hex()),
			Symbol:    "TOKEN0", // TODO: Fetch via ERC20 symbol() call
			Decimals:  18,       // TODO: Fetch via ERC20 decimals() call
			Swappable: true,
		},
		{
			Address:   strings.ToLower(token1Address.Hex()),
			Symbol:    "USDC", // All Stabull pools have USDC as quote token
			Decimals:  6,      // USDC always has 6 decimals
			Swappable: true,
		},
	}

	return &entity.Pool{
		Address:   strings.ToLower(poolAddress),
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  reserves,
		Tokens:    tokens,
		Extra:     string(extraBytes),
	}, nil
}
