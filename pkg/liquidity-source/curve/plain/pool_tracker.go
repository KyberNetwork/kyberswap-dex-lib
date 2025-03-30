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

	var (
		initialA, futureA, initialATime, futureATime, swapFee, adminFee, lpSupply, oracleRate *big.Int

		numTokens = len(p.Tokens)

		balances   = make([]*big.Int, numTokens)
		balancesV1 = make([]*big.Int, numTokens)

		// for pools that have non-standard rate multipliers
		storedRates   [shared.MaxTokenCount]*big.Int
		registryRates [shared.MaxTokenCount]*big.Int
	)

	/*
		all variants of Plain pools need these common info:
			- InitialA, FutureA, InitialATime, FutureATime: to calculate A coefficient
			- SwapFee, AdminFee
			- Balances: pool can store balances themselves or call to external contract, but the `balances` method already abstract that away
		some pool variants need additional info:
			Rates: some pools don't use standard rates:
				- if they expose `stored_rates` method, use that
				- if they have `oracle` method without argument, call that to get 2nd coin's rate
				- if call to main registry `get_rates` return non empty, use that
				- otherwise leave it empty to use standard rate
	*/

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, err
	}

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	req.SetFrom(nonZeroAddr) // poolMethodStoredRates behaves differently for tx.origin == 0

	req.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: p.Address,
		Method: poolMethodInitialA,
		Params: nil,
	}, []interface{}{&initialA})

	req.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: p.Address,
		Method: poolMethodFutureA,
		Params: nil,
	}, []interface{}{&futureA})

	req.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: p.Address,
		Method: poolMethodInitialATime,
		Params: nil,
	}, []interface{}{&initialATime})

	req.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: p.Address,
		Method: poolMethodFutureATime,
		Params: nil,
	}, []interface{}{&futureATime})

	req.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: p.Address,
		Method: poolMethodFee,
		Params: nil,
	}, []interface{}{&swapFee})

	req.AddCall(&ethrpc.Call{
		ABI:    curvePlainABI,
		Target: p.Address,
		Method: poolMethodAdminFee,
		Params: nil,
	}, []interface{}{&adminFee})

	req.AddCall(&ethrpc.Call{
		ABI:    shared.ERC20ABI,
		Target: staticExtra.LpToken,
		Method: shared.ERC20MethodTotalSupply,
		Params: nil,
	}, []interface{}{&lpSupply})

	req.AddCall(&ethrpc.Call{
		ABI:    numTokenDependedABIs[numTokens],
		Target: p.Address,
		Method: poolMethodStoredRates,
		Params: nil,
	}, []interface{}{&storedRates})

	if len(staticExtra.Oracle) > 0 {
		req.AddCall(&ethrpc.Call{
			ABI:    shared.OracleABI,
			Target: staticExtra.Oracle,
			Method: poolMethodLatestAnswer,
			Params: nil,
		}, []interface{}{&oracleRate})
	}

	if dataSourceAddresses, ok := shared.DataSourceAddresses[t.config.ChainCode]; ok {
		if mainRegistryAddress, ok := dataSourceAddresses[shared.CURVE_DATASOURCE_MAIN]; ok {
			req.AddCall(&ethrpc.Call{
				ABI:    shared.MainRegistryABI,
				Target: mainRegistryAddress,
				Method: mainRegistryMethodGetRates,
				Params: []interface{}{common.HexToAddress(p.Address)},
			}, []interface{}{&registryRates})
		}
	}

	for i := range p.Tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    curvePlainABI,
			Target: p.Address,
			Method: poolMethodBalances,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&balances[i]})

		req.AddCall(&ethrpc.Call{
			ABI:    getBalances128ABI,
			Target: p.Address,
			Method: poolMethodBalances,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&balancesV1[i]})
	}

	if res, err := req.TryBlockAndAggregate(); err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to aggregate call pool data")
		return entity.Pool{}, err
	} else if res.BlockNumber != nil {
		p.BlockNumber = res.BlockNumber.Uint64()
	}

	var extra = Extra{
		InitialA:     number.SetFromBig(initialA),
		FutureA:      number.SetFromBig(futureA),
		InitialATime: initialATime.Int64(),
		FutureATime:  futureATime.Int64(),
		SwapFee:      number.SetFromBig(swapFee),
		AdminFee:     number.SetFromBig(adminFee),
	}

	// first check `stored_rates`
	if checkValidCustomRates(&p, storedRates) {
		lg.Infof("use custom stored rate %v", storedRates)
		if err := t.updateRateMultipliers(lg, &extra, numTokens, storedRates[:numTokens]); err != nil {
			return entity.Pool{}, err
		}
	} else if oracleRate != nil && oracleRate.Sign() != 0 && numTokens == 2 {
		// then check if there is valid answer from oracle (only valid for 2 coins pool)
		lg.Infof("use custom oracle rate %v", oracleRate)
		if err := t.updateRateMultipliers(lg, &extra, 2, []*big.Int{bignumber.TenPowInt(18), oracleRate}); err != nil {
			return entity.Pool{}, err
		}
	} else {
		// check rates from main registry
		// `rates` from registry need to be multiplied with Precision first
		for i, token := range p.Tokens {
			if registryRates[i] != nil {
				registryRates[i].Mul(registryRates[i], bignumber.TenPowInt(18-token.Decimals))
			}
		}
		if checkValidCustomRates(&p, registryRates) {
			lg.Infof("use custom registry rate %v", registryRates)
			if err := t.updateRateMultipliers(lg, &extra, numTokens, registryRates[:numTokens]); err != nil {
				return entity.Pool{}, err
			}
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		lg.WithFields(logger.Fields{"error": err}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	var reserves = make(entity.PoolReserves, 0, len(balances)+1)
	for i := range balances {
		if balances[i] != nil {
			reserves = append(reserves, balances[i].String())
		} else if balancesV1[i] != nil {
			reserves = append(reserves, balancesV1[i].String())
		} else {
			reserves = append(reserves, "0")
		}
	}
	reserves = append(reserves, lpSupply.String())

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	return p, nil
}

func (t *PoolTracker) updateRateMultipliers(lg logger.Logger, extra *Extra, numTokens int, customRates []*big.Int) error {
	extra.RateMultipliers = make([]uint256.Int, numTokens)
	lg.Debugf("pool use stored rate %v", customRates)

	for i := 0; i < numTokens; i++ {
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
