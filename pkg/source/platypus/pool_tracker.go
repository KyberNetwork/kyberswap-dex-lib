package platypus

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	ethClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryE0(DexTypePlatypus, NewPoolTracker)

func NewPoolTracker(ethClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		ethClient: ethClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[Platypus] Start getting new pool's state")

	// Get pool's state.
	poolState, blockNumber, err := t.getPoolState(ctx, p.Address)
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("Fail to get pool's state")
		return entity.Pool{}, err
	}

	// Get assets' address.
	assetAddresses, err := t.getAssetAddresses(ctx, p.Address, poolState.TokenAddresses, blockNumber)
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"tokens":  poolState.TokenAddresses,
			"error":   err,
		}).Errorf("Fail to get address of assets")
		return entity.Pool{}, err
	}

	// Get assets' state.
	assetStates, err := t.getAssetStates(ctx, assetAddresses, blockNumber)
	if err != nil {
		logger.WithFields(logger.Fields{
			"assetAddresses": assetAddresses,
			"error":          err,
		}).Errorf("Fail to get asset states")
		return entity.Pool{}, err
	}

	p.Type = getPoolTypeByPriceOracle(hexutil.Encode(poolState.PriceOracle[:]))

	sAvaxRate := big.NewInt(0)
	if p.Type == PoolTypePlatypusAvax {
		sAvaxRate, err = t.getSAvaxRate(ctx, addressStakedAvax, blockNumber)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("Fail to get staked avax rate")
			return entity.Pool{}, err
		}
	}

	extra := newExtra(poolState, assetStates, sAvaxRate)

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"extra":   extra,
			"error":   err,
		}).Errorf("Fail to marshal pool's extra")
		return entity.Pool{}, err
	}

	reserves := make([]string, 0, len(assetStates))
	for _, assetState := range assetStates {
		reserves = append(reserves, assetState.Cash.String())
	}

	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.Tokens = newPoolTokens(poolState.TokenAddresses)
	p.Timestamp = time.Now().Unix()
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}

	return p, nil
}

func (t *PoolTracker) getPoolState(ctx context.Context, address string) (PoolState, *big.Int, error) {
	var state PoolState
	var c1, haircutRate, retentionRatio, slippageParamK, slippageParamN, xThreshold *uint256.Int

	request := t.ethClient.NewRequest().
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetC1,
			Params: nil,
		}, []any{&c1}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetHaircutRate,
			Params: nil,
		}, []any{&haircutRate}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetPriceOracle,
			Params: nil,
		}, []any{&state.PriceOracle}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetRetentionRatio,
			Params: nil,
		}, []any{&retentionRatio}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetSlippageParamK,
			Params: nil,
		}, []any{&slippageParamK}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetSlippageParamN,
			Params: nil,
		}, []any{&slippageParamN}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetTokenAddresses,
			Params: nil,
		}, []any{&state.TokenAddresses}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetXThreshold,
			Params: nil,
		}, []any{&xThreshold}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodPaused,
			Params: nil,
		}, []any{&state.Paused})

	response, err := request.Aggregate()
	if err != nil {
		return PoolState{}, nil, err
	}

	state.C1 = u256ToBig(c1)
	state.HaircutRate = u256ToBig(haircutRate)
	state.RetentionRatio = u256ToBig(retentionRatio)
	state.SlippageParamK = u256ToBig(slippageParamK)
	state.SlippageParamN = u256ToBig(slippageParamN)
	state.XThreshold = u256ToBig(xThreshold)

	return state, response.BlockNumber, nil
}

func (t *PoolTracker) getAssetAddresses(
	ctx context.Context,
	poolAddress string,
	tokenAddresses []common.Address,
	blockNumber *big.Int,
) ([]common.Address, error) {
	assetAddresses := make([]common.Address, len(tokenAddresses))
	request := t.ethClient.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		request.SetBlockNumber(blockNumber)
	}
	for i, tokenAddress := range tokenAddresses {
		request.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodAssetOf,
			Params: []any{tokenAddress},
		}, []any{&assetAddresses[i]})
	}

	if _, err := request.Aggregate(); err != nil {
		return nil, err
	}

	return assetAddresses, nil
}

func (t *PoolTracker) getAssetStates(
	ctx context.Context, addresses []common.Address,
	blockNumber *big.Int,
) ([]AssetState, error) {
	states := make([]AssetState, len(addresses))
	cashes := make([]*uint256.Int, len(addresses))
	liabilities := make([]*uint256.Int, len(addresses))
	request := t.ethClient.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		request.SetBlockNumber(blockNumber)
	}
	for i, addr := range addresses {
		address := addr.Hex()
		request.
			AddCall(&ethrpc.Call{
				ABI:    assetABI,
				Target: address,
				Method: assetMethodCash,
				Params: nil,
			}, []any{&cashes[i]}).
			AddCall(&ethrpc.Call{
				ABI:    assetABI,
				Target: address,
				Method: assetMethodDecimals,
				Params: nil,
			}, []any{&states[i].Decimals}).
			AddCall(&ethrpc.Call{
				ABI:    assetABI,
				Target: address,
				Method: assetMethodLiability,
				Params: nil,
			}, []any{&liabilities[i]}).
			AddCall(&ethrpc.Call{
				ABI:    assetABI,
				Target: address,
				Method: assetMethodUnderlyingToken,
				Params: nil,
			}, []any{&states[i].UnderlyingToken}).
			AddCall(&ethrpc.Call{
				ABI:    assetABI,
				Target: address,
				Method: assetMethodAggregateAccount,
			}, []any{&states[i].AggregateAccount})
	}

	if _, err := request.Aggregate(); err != nil {
		return nil, err
	}

	for i := range states {
		states[i].Cash = u256ToBig(cashes[i])
		states[i].Liability = u256ToBig(liabilities[i])
	}

	return states, nil
}

func (t *PoolTracker) getSAvaxRate(ctx context.Context, address string, blockNumber *big.Int) (*big.Int, error) {
	var rate *uint256.Int
	request := t.ethClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    stakedAvaxABI,
			Target: address,
			Method: stakedAvaxMethodGetPooledAvaxByShares,
			Params: []any{bOne},
		}, []any{&rate})
	if blockNumber != nil {
		request.SetBlockNumber(blockNumber)
	}
	if _, err := request.Aggregate(); err != nil {
		return nil, err
	}

	return u256ToBig(rate), nil
}

func u256ToBig(v *uint256.Int) *big.Int {
	if v == nil {
		return nil
	}

	return v.ToBig()
}
