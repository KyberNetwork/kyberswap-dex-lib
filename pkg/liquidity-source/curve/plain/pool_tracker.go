package plain

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *shared.Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	lg := logger.WithFields(logger.Fields{
		"dexId":   config.DexID,
		"dexType": DexType,
	})

	err := shared.InitDataSourceAddresses(lg, config, ethrpcClient)
	if err != nil {
		return nil, err
	}

	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logger:       lg,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	lg := t.logger.WithFields(logger.Fields{"poolAddress": p.Address})
	lg.Info("Start updating state ...")
	defer func() { lg.Info("Finish updating state.") }()

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, err
	}

	mainRegistryAddress := t.mainRegistryAddress()
	numTokens := len(p.Tokens)
	d := &rpcData{
		balances:   make([]*big.Int, numTokens),
		balancesV1: make([]*big.Int, numTokens),
	}

	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).
		SetFrom(shared.AddrDummy) // poolMethodStoredRates behaves differently for tx.origin == 0
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) },
		p.Address, numTokens, staticExtra.LpToken, staticExtra.Oracle, mainRegistryAddress, d)

	if res, err := req.TryBlockAndAggregate(); err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to aggregate call pool data")
		return entity.Pool{}, err
	} else if res.BlockNumber != nil {
		p.BlockNumber = res.BlockNumber.Uint64()
	}

	return buildPoolState(lg, p, d)
}

func (t *PoolTracker) mainRegistryAddress() string {
	if dataSourceAddresses, ok := shared.DataSourceAddresses[t.config.ChainCode]; ok {
		if addr, ok := dataSourceAddresses[shared.CURVE_DATASOURCE_MAIN]; ok {
			return addr
		}
	}
	return ""
}

type rpcData struct {
	initialA, futureA, initialATime, futureATime, swapFee, adminFee, lpSupply *big.Int
	oracleRate                                                                 *big.Int
	storedRates, registryRates                                                 [shared.MaxTokenCount]*big.Int
	balances, balancesV1                                                       []*big.Int
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), poolAddress string, numTokens int,
	lpToken, oracle, mainRegistryAddress string, d *rpcData) {
	addFn(&ethrpc.Call{ABI: curvePlainABI, Target: poolAddress, Method: poolMethodInitialA}, []any{&d.initialA})
	addFn(&ethrpc.Call{ABI: curvePlainABI, Target: poolAddress, Method: poolMethodFutureA}, []any{&d.futureA})
	addFn(&ethrpc.Call{ABI: curvePlainABI, Target: poolAddress, Method: poolMethodInitialATime}, []any{&d.initialATime})
	addFn(&ethrpc.Call{ABI: curvePlainABI, Target: poolAddress, Method: poolMethodFutureATime}, []any{&d.futureATime})
	addFn(&ethrpc.Call{ABI: curvePlainABI, Target: poolAddress, Method: poolMethodFee}, []any{&d.swapFee})
	addFn(&ethrpc.Call{ABI: curvePlainABI, Target: poolAddress, Method: poolMethodAdminFee}, []any{&d.adminFee})
	addFn(&ethrpc.Call{ABI: curvePlainABI, Target: lpToken, Method: shared.ERC20MethodTotalSupply}, []any{&d.lpSupply})
	addFn(&ethrpc.Call{ABI: numTokenDependedABIs[numTokens], Target: poolAddress, Method: poolMethodStoredRates}, []any{&d.storedRates})
	if len(oracle) > 0 {
		addFn(&ethrpc.Call{ABI: shared.OracleABI, Target: oracle, Method: poolMethodLatestAnswer}, []any{&d.oracleRate})
	}
	if mainRegistryAddress != "" {
		addFn(&ethrpc.Call{
			ABI:    shared.MainRegistryABI,
			Target: mainRegistryAddress,
			Method: mainRegistryMethodGetRates,
			Params: []any{common.HexToAddress(poolAddress)},
		}, []any{&d.registryRates})
	}
	for i := range numTokens {
		addFn(&ethrpc.Call{ABI: curvePlainABI, Target: poolAddress, Method: poolMethodBalances, Params: []any{big.NewInt(int64(i))}}, []any{&d.balances[i]})
		addFn(&ethrpc.Call{ABI: getBalances128ABI, Target: poolAddress, Method: poolMethodBalances, Params: []any{big.NewInt(int64(i))}}, []any{&d.balancesV1[i]})
	}
}

func buildPoolState(lg logger.Logger, p entity.Pool, d *rpcData) (entity.Pool, error) {
	numTokens := len(p.Tokens)
	var extra = Extra{
		InitialA:     number.SetFromBig(d.initialA),
		FutureA:      number.SetFromBig(d.futureA),
		InitialATime: d.initialATime.Int64(),
		FutureATime:  d.futureATime.Int64(),
		SwapFee:      number.SetFromBig(d.swapFee),
		AdminFee:     number.SetFromBig(d.adminFee),
	}

	// first check `stored_rates`
	if checkValidCustomRates(&p, d.storedRates) {
		lg.Infof("use custom stored rate %v", d.storedRates)
		if err := updateRateMultipliers(lg, &extra, numTokens, d.storedRates[:numTokens]); err != nil {
			return entity.Pool{}, err
		}
	} else if d.oracleRate != nil && d.oracleRate.Sign() != 0 && numTokens == 2 {
		// then check if there is valid answer from oracle (only valid for 2 coins pool)
		lg.Infof("use custom oracle rate %v", d.oracleRate)
		if err := updateRateMultipliers(lg, &extra, 2, []*big.Int{bignumber.TenPowInt(18), d.oracleRate}); err != nil {
			return entity.Pool{}, err
		}
	} else {
		// check rates from main registry
		// `rates` from registry need to be multiplied with Precision first
		for i, token := range p.Tokens {
			if d.registryRates[i] != nil {
				d.registryRates[i].Mul(d.registryRates[i], bignumber.TenPowInt(18-token.Decimals))
			}
		}
		if checkValidCustomRates(&p, d.registryRates) {
			lg.Infof("use custom registry rate %v", d.registryRates)
			if err := updateRateMultipliers(lg, &extra, numTokens, d.registryRates[:numTokens]); err != nil {
				return entity.Pool{}, err
			}
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	var reserves = make(entity.PoolReserves, 0, len(d.balances)+1)
	for i := range d.balances {
		if d.balances[i] != nil {
			reserves = append(reserves, d.balances[i].String())
		} else if d.balancesV1[i] != nil {
			reserves = append(reserves, d.balancesV1[i].String())
		} else {
			reserves = append(reserves, "0")
		}
	}
	reserves = append(reserves, d.lpSupply.String())

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	return p, nil
}

func updateRateMultipliers(lg logger.Logger, extra *Extra, numTokens int,
	customRates []*big.Int) error {
	extra.RateMultipliers = make([]uint256.Int, numTokens)
	lg.Debugf("pool use stored rate %v", customRates)

	for i := range numTokens {
		if overflow := extra.RateMultipliers[i].SetFromBig(customRates[i]); overflow {
			lg.WithFields(logger.Fields{"storedRates": customRates}).Error("invalid stored rates")
			return ErrInvalidStoredRates
		}
	}
	return nil
}

func checkValidCustomRates(p *entity.Pool, customRates [8]*big.Int) bool {
	for i := range p.Tokens {
		standardRate := bignumber.TenPowInt(36 - p.Tokens[i].Decimals)

		// only compare if stored_rate from rpc is valid (not nil and not zero)
		if customRates[i] != nil && customRates[i].Sign() != 0 && customRates[i].Cmp(standardRate) != 0 {
			return true
		}
	}
	return false
}
