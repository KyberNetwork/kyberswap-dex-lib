package etherfiebtc

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolListUpdater struct {
	ethrpcClient   *ethrpc.Client
	config         *Config
	hasInitialized bool
}

func NewPoolListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		config:         config,
		ethrpcClient:   ethrpcClient,
		hasInitialized: false,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		logger.Debug("skip since pool has been initialized")
		return nil, nil, nil
	}

	logger.WithFields(logger.Fields{
		"exchange": u.config.DexID,
	}).Info("Started getting new pool")

	byteData, ok := bytesByPath[u.config.PoolPath]
	if !ok {
		logger.Errorf("misconfigured poolPath")
		return nil, nil, errors.New("misconfigured poolPath")
	}
	var initialPool InitialPool
	if err := json.Unmarshal(byteData, &initialPool); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal poolData")
		return nil, nil, err
	}

	tokens := make(entity.PoolTokens, 0, len(initialPool.Tokens))
	reserves := make(entity.PoolReserves, 0, len(initialPool.Tokens))
	for _, token := range initialPool.Tokens {
		tokenEntity := entity.PoolToken{
			Address:   strings.ToLower(token.Address),
			Name:      token.Name,
			Symbol:    token.Symbol,
			Decimals:  token.Decimals,
			Swappable: true,
		}
		tokens = append(tokens, &tokenEntity)
		reserves = append(reserves, defaultReserves)
	}

	teller := strings.ToLower(initialPool.Teller)
	accountant := strings.ToLower(initialPool.Accountant)

	staticExtra := StaticExtra{
		Accountant: accountant,
		Base:       initialPool.Base,
		Decimals:   initialPool.Decimals,
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return nil, nil, err
	}

	extra, blockNumber, err := getExtra(ctx, u.ethrpcClient, teller, accountant, tokens, nil)
	if err != nil {
		return nil, nil, err
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	poolEntity := entity.Pool{
		Address:     strings.ToLower(initialPool.Teller),
		Exchange:    u.config.DexID,
		Type:        DexType,
		Timestamp:   time.Now().Unix(),
		Reserves:    reserves,
		Tokens:      tokens,
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
		BlockNumber: blockNumber,
	}

	u.hasInitialized = true

	logger.WithFields(logger.Fields{
		"exchange": u.config.DexID,
		"address":  poolEntity.Address,
	}).Info("Finished getting new pool")

	return []entity.Pool{poolEntity}, nil, nil
}

func getExtra(
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	teller, accountant string,
	tokens entity.PoolTokens,
	overrides map[common.Address]gethclient.OverrideAccount,
) (Extra, uint64, error) {
	var (
		isTellerPaused  bool
		shareLockPeriod uint64
		accountantState AccountantState
	)
	assetData := make([]Asset, len(tokens))
	rateProviderData := make([]RateProviderData, len(tokens))

	calls := ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    tellerABI,
		Target: teller,
		Method: tellerMethodIsPaused,
	}, []interface{}{&isTellerPaused})
	calls.AddCall(&ethrpc.Call{
		ABI:    tellerABI,
		Target: teller,
		Method: tellerMethodShareLockPeriod,
	}, []interface{}{&shareLockPeriod})
	for i := range tokens {
		tokenAddress := common.HexToAddress(tokens[i].Address)
		calls.AddCall(&ethrpc.Call{
			ABI:    tellerABI,
			Target: teller,
			Params: []interface{}{tokenAddress},
			Method: tellerMethodAssetData,
		}, []interface{}{&assetData[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    accountantABI,
			Target: accountant,
			Params: []interface{}{tokenAddress},
			Method: accountantMethodRateProviderData,
		}, []interface{}{&rateProviderData[i]})
	}
	calls.AddCall(&ethrpc.Call{
		ABI:    accountantABI,
		Target: accountant,
		Method: accountantMethodAccountantState,
	}, []interface{}{&accountantState})

	resp, err := calls.Aggregate()
	if err != nil {
		return Extra{}, 0, err
	}
	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	assets := make(map[string]Asset)
	rateProviders := make(map[string]RateProviderData)
	for i, token := range tokens {
		assets[token.Address] = assetData[i]
		rateProviders[token.Address] = rateProviderData[i]
	}

	return Extra{
		IsTellerPaused:  isTellerPaused,
		ShareLockPeriod: shareLockPeriod,
		Assets:          assets,
		AccountantState: accountantState,
		RateProviders:   rateProviders,
	}, resp.BlockNumber.Uint64(), nil
}
