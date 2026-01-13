package stabull

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}

type TokenPair struct {
	BaseToken  string
	QuoteToken string
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
	var (
		dexID     = d.config.DexID
		startTime = time.Now()
	)

	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	// Stabull uses factory.getCurve(base, quote) to discover pools
	// We query the factory with known token pairs to find all deployed pools
	pairAddresses, err := d.discoverPoolsFromFactory(ctx)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("discoverPoolsFromFactory failed")

		return nil, metadataBytes, err
	}

	pools, err := d.initPools(ctx, pairAddresses)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"pools_len":   len(pools),
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, metadataBytes, nil
}

// getAllPairsLength gets number of pools from the factory contract
// TODO: Stabull factory doesn't have curvesLength method - needs event-based discovery
// Pools should be discovered via NewCurve events instead of indexed enumeration
func (d *PoolsListUpdater) getAllPairsLength(ctx context.Context) (int, error) {
	// Stabull factory uses event-based pool discovery (NewCurve events)
	// There is no curvesLength() method in the factory contract
	// Return 0 for now - proper implementation should use GetNewPoolsType to discover via events
	logger.WithFields(logger.Fields{
		"dex": DexType,
	}).Warn("getAllPairsLength not supported - Stabull uses event-based pool discovery")

	return 0, nil
}

// discoverPoolsFromFactory queries the factory with known token pairs to find deployed pools
func (d *PoolsListUpdater) discoverPoolsFromFactory(ctx context.Context) ([]common.Address, error) {
	// Token pairs that have known Stabull pools
	// These pairs match the official Stabull documentation
	tokenPairs := d.getKnownTokenPairs()

	logger.WithFields(logger.Fields{
		"dex":        DexType,
		"chainID":    d.config.ChainID,
		"pairsCount": len(tokenPairs),
	}).Info("discovering pools from factory")

	var poolAddresses []common.Address

	// Query factory in batches to avoid RPC limits
	const batchSize = 50
	for i := 0; i < len(tokenPairs); i += batchSize {
		end := i + batchSize
		if end > len(tokenPairs) {
			end = len(tokenPairs)
		}

		batch := tokenPairs[i:end]
		addresses, err := d.queryFactoryBatch(ctx, batch)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dex":   DexType,
				"batch": i / batchSize,
				"error": err,
			}).Warn("factory batch query failed")
			continue
		}

		logger.WithFields(logger.Fields{
			"dex":          DexType,
			"batch":        i / batchSize,
			"poolsInBatch": len(addresses),
		}).Info("factory batch query succeeded")

		poolAddresses = append(poolAddresses, addresses...)
	}

	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"pools_found": len(poolAddresses),
	}).Info("discovered pools from factory")

	return poolAddresses, nil
}

func (d *PoolsListUpdater) getKnownTokenPairs() []TokenPair {
	// All known token pairs from Stabull
	// Token addresses verified from working pool discovery test
	var pairs []TokenPair

	switch d.config.ChainID {
	case 137: // Polygon
		// All tokens paired with Native USDC (0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359)
		usdcPolygon := "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"
		pairs = []TokenPair{
			{"0xd2a530170D71a9Cfe1651Fb468E2B98F7Ed7456b", usdcPolygon}, // AUDF/USDC
			{"0x4ed141110f6eeeaba9a1df36d8c26f684d2475dc", usdcPolygon}, // BRZ/USDC
			{"0x12050c705152931cFEe3DD56c52Fb09Dea816C23", usdcPolygon}, // COPM/USDC
			{"0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063", usdcPolygon}, // DAI/USDC
			{"0xE111178A87A3BFf0c8d18DECBa5798827539Ae99", usdcPolygon}, // EURS/USDC
			{"0xFbBE4b730e1e77d02dC40fEdF9438E2802eab3B5", usdcPolygon}, // NZDS/USDC
			{"0x9cFb3B1b217b41C4E748774368099Dd8Dd7E89A1", usdcPolygon}, // OFD/USDC
			{"0x553d3D295e0f695B9228246232eDF400ed3560B5", usdcPolygon}, // PAXG/USDC
			{"0x87a25dc121Db52369F4a9971F664Ae5e372CF69A", usdcPolygon}, // PHPC/USDC
			{"0x4Fb71290Ac171E1d144F7221D882BECAc7196EB5", usdcPolygon}, // TRYB/USDC
			{"0xc2132D05D31c914a87C6611C10748AEb04B58e8F", usdcPolygon}, // USDT/USDC
			{"0xDC3326e71D45186F113a2F448984CA0e8D201995", usdcPolygon}, // XSGD/USDC
			{"0x02567e4b14b25549331fCEe2B56c647A8bAB16FD", usdcPolygon}, // ZCHF/USDC
		}
	case 8453: // Base
		// All tokens paired with Native USDC (0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913)
		usdcBase := "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
		pairs = []TokenPair{
			{"0x449b3317a6d1efb1bc3ba0700c9eaa4ffff4ae65", usdcBase}, // AUDD/USDC
			{"0xE9185Ee218cae427aF7B9764A011bb89FeA761B4", usdcBase}, // BRZ/USDC
			{"0x60a3e35cc302bfa44cb288bc5a4f316fdb1adb42", usdcBase}, // EURC/USDC
			{"0x269cae7dc59803e5c596c95756faeebb6030e0af", usdcBase}, // MXNe/USDC
			{"0xFb8718a69aed7726AFb3f04D2Bd4bfDE1BdCb294", usdcBase}, // TRYB/USDC
			{"0xb755506531786C8aC63B756BaB1ac387bACB0C04", usdcBase}, // ZARP/USDC
			{"0xd4dd9e2f021bb459d5a5f6c24c12fe09c5d45553", usdcBase}, // ZCHF/USDC
			{"0x7479791022eb1030bbc3b09f6575c5db4ddc0b90", usdcBase}, // OFD/USDC
		}
	case 1: // Ethereum
		// All tokens paired with USDC (0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48)
		usdcEthereum := "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
		pairs = []TokenPair{
			{"0x4cCe605eD955295432958d8951D0B176C10720d5", usdcEthereum}, // AUDD/USDC
			{"0xdB25f211AB05b1c97D595516F45794528a807ad8", usdcEthereum}, // EURS/USDC
			{"0xc08512927d12348f6620a698105e1baac6ecd911", usdcEthereum}, // GYEN/USDC
			{"0xda446fad08277b4d2591536f204e018f32b6831c", usdcEthereum}, // NZDS/USDC
			{"0x2c537e5624e4af88a7ae4060c022609376c8d0eb", usdcEthereum}, // TRYB/USDC
		}
	}

	return pairs
}

func (d *PoolsListUpdater) queryFactoryBatch(ctx context.Context, pairs []TokenPair) ([]common.Address, error) {
	rpcRequest := d.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	// Prepare result storage for all pool addresses
	poolAddressResults := make([]common.Address, len(pairs))

	// Add all getCurve calls to batch
	for i, pair := range pairs {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    stabullFactoryABI,
			Target: d.config.FactoryAddress,
			Method: factoryMethodGetCurve,
			Params: []interface{}{
				common.HexToAddress(pair.BaseToken),
				common.HexToAddress(pair.QuoteToken),
			},
		}, []interface{}{&poolAddressResults[i]})
	}

	logger.WithFields(logger.Fields{
		"dex":     DexType,
		"calls":   len(pairs),
		"factory": d.config.FactoryAddress,
	}).Info("executing factory batch RPC")

	resp, err := rpcRequest.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dex":   DexType,
			"error": err,
		}).Error("factory batch RPC failed")
		return nil, fmt.Errorf("factory batch RPC failed: %w", err)
	}

	logger.WithFields(logger.Fields{
		"dex":         DexType,
		"results":     len(resp.Result),
		"firstResult": len(resp.Result) > 0,
	}).Info("factory batch RPC completed")

	var poolAddresses []common.Address
	for i, isSuccess := range resp.Result {
		if !isSuccess {
			logger.WithFields(logger.Fields{
				"dex":   DexType,
				"index": i,
				"pair":  fmt.Sprintf("%s/%s", pairs[i].BaseToken, pairs[i].QuoteToken),
			}).Warn("factory call failed for pair")
			continue
		}

		poolAddress := poolAddressResults[i]
		if poolAddress == (common.Address{}) {
			// No pool deployed for this pair
			logger.WithFields(logger.Fields{
				"dex":  DexType,
				"pair": fmt.Sprintf("%s/%s", pairs[i].BaseToken, pairs[i].QuoteToken),
			}).Debug("no pool deployed for pair")
			continue
		}

		logger.WithFields(logger.Fields{
			"dex":  DexType,
			"pair": fmt.Sprintf("%s/%s", pairs[i].BaseToken, pairs[i].QuoteToken),
			"pool": poolAddress.Hex(),
		}).Info("found pool")

		poolAddresses = append(poolAddresses, poolAddress)
	}

	return poolAddresses, nil
}

func (d *PoolsListUpdater) initPools(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	var pools []entity.Pool
	for _, poolAddress := range pairAddresses {
		pool, err := d.getNewPool(ctx, poolAddress.Hex())
		if err != nil {
			logger.WithFields(logger.Fields{
				"dex":         DexType,
				"poolAddress": poolAddress.Hex(),
				"error":       err,
			}).Warn("failed to fetch pool")
			continue
		}
		pools = append(pools, *pool)
	}

	return pools, nil
}

func (d *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{
		Offset: newOffset,
	}

	return json.Marshal(metadata)
}

func (d *PoolsListUpdater) getBatchSize(length int, limit int, offset int) int {
	if length <= 0 {
		return 0
	}

	if offset >= length {
		return 0
	}

	if limit <= 0 {
		limit = length
	}

	batchSize := min(length-offset, limit)

	return batchSize
}

func (d *PoolsListUpdater) getNewPool(ctx context.Context, poolAddress string) (*entity.Pool, error) {
	var (
		token0Address   common.Address
		token1Address   common.Address
		token0Decimals  uint8
		liquidityResult struct {
			Total      *big.Int
			Individual []*big.Int
		}
		curveResult struct {
			Alpha   *big.Int
			Beta    *big.Int
			Delta   *big.Int
			Epsilon *big.Int
			Lambda  *big.Int
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
	}, []interface{}{&liquidityResult})

	// Fetch curve parameters (alpha, beta, delta, epsilon, lambda)
	// viewCurve() returns (uint256 alpha_, uint256 beta_, uint256 delta_, uint256 epsilon_, uint256 lambda_)
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    stabullPoolABI,
		Target: poolAddress,
		Method: poolMethodViewCurve,
		Params: []interface{}{},
	}, []interface{}{&curveResult})

	// Execute first batch of calls to get token addresses and pool data
	_, err := rpcRequest.Aggregate()
	if err != nil {
		return nil, err
	}

	// Now that we have token addresses, fetch token0 decimals in a second RPC call
	rpcRequest2 := d.ethrpcClient.NewRequest().SetContext(ctx)
	rpcRequest2.AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: token0Address.Hex(),
		Method: abi.Erc20DecimalsMethod,
		Params: []interface{}{},
	}, []interface{}{&token0Decimals})

	_, err = rpcRequest2.Aggregate()
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

	// Token metadata: token1 is always USDC (6 decimals)
	// Token0 decimals fetched via ERC20 call
	// Fallback to 18 decimals if fetch fails
	if token0Decimals == 0 {
		token0Decimals = 18
	}

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

	// Token metadata
	// Stabull pools always have USDC as token1 (quote currency)
	// Token0 is the fiat-backed stablecoin (e.g., AUDS, NZDS, etc.)
	// Symbol is kept generic as ERC20 symbol() method is not always reliable
	tokens := []*entity.PoolToken{
		{
			Address:   strings.ToLower(token0Address.Hex()),
			Symbol:    "TOKEN0",       // Generic symbol (actual symbol can vary: AUDS, NZDS, etc.)
			Decimals:  token0Decimals, // Fetched via ERC20 decimals() call
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
