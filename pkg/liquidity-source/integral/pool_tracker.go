package integral

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
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

var _ = pooltrack.RegisterFactoryCE(DexTypeIntegral, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (u *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return u.getNewPoolState(ctx, p, params, nil)
}

func (u *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return u.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (u *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("%s: Start getting new state of pool (address: %s)", u.config.DexID, p.Address)

	var (
		poolState PoolState

		token0LimitMaxMultiplier *big.Int
		token1LimitMaxMultiplier *big.Int

		token0 = common.HexToAddress(p.Tokens[0].Address)
		token1 = common.HexToAddress(p.Tokens[1].Address)

		isPairEnabled bool

		pairInfo         PriceByPair
		invertedPairInfo PriceByPair
	)

	rpcRequest := u.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		rpcRequest.SetOverrides(overrides)
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerGetPoolStateMethod,
		Params: []any{token0, token1},
	}, []any{&poolState})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerIsPairEnabledMethod,
		Params: []any{common.HexToAddress(p.Address)},
	}, []any{&isPairEnabled})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerGetTokenLimitMaxMultiplierMethod,
		Params: []any{token0},
	}, []any{&token0LimitMaxMultiplier})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerGetTokenLimitMaxMultiplierMethod,
		Params: []any{token1},
	}, []any{&token1LimitMaxMultiplier})

	if _, err := rpcRequest.Aggregate(); err != nil {
		logger.Errorf("%s: failed to fetch basic pool data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return p, err
	}

	if !isPairEnabled {
		var extra Extra
		_ = json.Unmarshal([]byte(p.Extra), &extra)

		extra.IsEnabled = false
		extraBytes, err := json.Marshal(extra)
		if err != nil {
			logger.Errorf("%s: failed to marshal extra data for disabled pool (address: %s, error: %v)", u.config.DexID, p.Address, err)
			return entity.Pool{}, err
		}

		p.Extra = string(extraBytes)
		p.Timestamp = time.Now().Unix()
		p.Reserves = []string{"0", "0"}

		return p, nil
	}

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerGetPairByAddressMethod,
		Params: []any{common.HexToAddress(p.Address), false}, // get Price when swap X -> Y
	}, []any{&pairInfo})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    relayerABI,
		Target: u.config.RelayerAddress,
		Method: relayerGetPairByAddressMethod,
		Params: []any{common.HexToAddress(p.Address), true}, // get Price when swap Y -> X
	}, []any{&invertedPairInfo})
	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.Errorf("%s: failed to fetch decimals data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return p, err
	}

	extra := Extra{
		RelayerAddress:           u.config.RelayerAddress,
		IsEnabled:                isPairEnabled,
		SwapFee:                  number.SetFromBig(poolState.Fee),
		Price:                    number.SetFromBig(pairInfo.Price),
		InvertedPrice:            number.SetFromBig(invertedPairInfo.Price),
		Token0LimitMin:           number.SetFromBig(poolState.LimitMin0),
		Token0LimitMax:           number.SetFromBig(poolState.LimitMax0),
		Token1LimitMin:           number.SetFromBig(poolState.LimitMin1),
		Token1LimitMax:           number.SetFromBig(poolState.LimitMax1),
		Token0LimitMaxMultiplier: number.SetFromBig(token0LimitMaxMultiplier),
		Token1LimitMaxMultiplier: number.SetFromBig(token1LimitMaxMultiplier),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.Errorf("%s: failed to marshal extra data (address: %s, error: %v)", u.config.DexID, p.Address, err)
		return p, err
	}

	reserve0 := new(uint256.Int).Set(extra.Token0LimitMax)
	reserve1 := new(uint256.Int).Set(extra.Token1LimitMax)

	if token0LimitMaxMultiplier.Sign() != 0 {
		reserve0, _ = new(uint256.Int).MulDivOverflow(extra.Token0LimitMax, precision, extra.Token0LimitMaxMultiplier)
	}
	if token1LimitMaxMultiplier.Sign() != 0 {
		reserve1, _ = new(uint256.Int).MulDivOverflow(extra.Token1LimitMax, precision, extra.Token1LimitMaxMultiplier)
	}

	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)
	p.Reserves = []string{reserve0.String(), reserve1.String()}

	fee, _ := poolState.Fee.Float64()
	p.SwapFee = fee / precision.Float64()

	logger.Infof("%s: Pool state updated successfully (address: %s)", u.config.DexID, p.Address)

	return p, nil
}
