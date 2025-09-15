package midas

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	config.MTokens = lo.MapEntries(config.MTokens, func(k string, v MTokenConfig) (string, MTokenConfig) {
		return strings.ToLower(k), v
	})

	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("start getting new state of pool")

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var (
		paymentTokens []common.Address
		tokenConfig   struct {
			DataFeed  common.Address
			Fee       *big.Int
			Allowance *big.Int
			Stable    bool
		}
		depositInstantFnPaused bool
		redeemInstantFnPaused  bool
		dailyLimits            *big.Int
		instantDailyLimit      *big.Int
		instantFee             *big.Int
		minAmount              *big.Int
		tokenRate              *big.Int
		mTokenRate             *big.Int
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	currentDayNumber := time.Now().Unix() / oneDayInSecond
	depositVault := t.config.MTokens[p.Tokens[0].Address].DepositVault
	redemptionVault := t.config.MTokens[p.Address].RedemptionVault

	req.SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    DepositVaultABI,
			Target: depositVault,
			Method: depositVaultGetPaymentTokensMethod,
		}, []any{&paymentTokens}).
		AddCall(&ethrpc.Call{
			ABI:    DepositVaultABI,
			Target: depositVault,
			Method: depositVaultGetPaymentTokensMethod,
		}, []any{&paymentTokens}).
		AddCall(&ethrpc.Call{
			ABI:    DepositVaultABI,
			Target: depositVault,
			Method: depositVaultTokensConfigMethod,
			Params: []any{common.HexToAddress(p.Tokens[1].Address)},
		}, []any{&tokenConfig}).
		AddCall(&ethrpc.Call{
			ABI:    DepositVaultABI,
			Target: depositVault,
			Method: vaultFnPausedMethod,
			Params: []any{depositInstantSelector},
		}, []any{&depositInstantFnPaused}).
		AddCall(&ethrpc.Call{
			ABI:    DepositVaultABI,
			Target: depositVault,
			Method: depositVaultInstantDailyLimitMethod,
		}, []any{&instantDailyLimit}).
		AddCall(&ethrpc.Call{
			ABI:    DepositVaultABI,
			Target: depositVault,
			Method: depositVaultDailyLimitsMethod,
			Params: []any{big.NewInt(currentDayNumber)},
		}, []any{&dailyLimits}).
		AddCall(&ethrpc.Call{
			ABI:    DepositVaultABI,
			Target: depositVault,
			Method: depositVaultInstantFeeMethod,
		}, []any{&instantFee}).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: staticExtra.DataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&tokenRate}).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: staticExtra.MTokenDataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&mTokenRate}).
		AddCall(&ethrpc.Call{
			ABI:    DepositVaultABI,
			Target: depositVault,
			Method: depositVaultMinAmountMethod,
		}, []any{&minAmount}).
		AddCall(&ethrpc.Call{
			ABI:    RedemptionVaultABI,
			Target: redemptionVault,
			Method: vaultFnPausedMethod,
			Params: []any{redeemInstantSelector},
		}, []any{&redeemInstantFnPaused})

	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	tokenRemoved := true
	for _, token := range paymentTokens {
		if strings.EqualFold(token.String(), p.Tokens[1].Address) {
			tokenRemoved = false
			break
		}
	}

	extra := Extra{
		TokenRemoved: tokenRemoved,
		TokenConfig: &TokenConfig{
			DataFeed:  tokenConfig.DataFeed,
			Fee:       uint256.MustFromBig(tokenConfig.Fee),
			Allowance: uint256.MustFromBig(tokenConfig.Allowance),
			Stable:    tokenConfig.Stable,
		},
		DepositInstantFnPaused: depositInstantFnPaused,
		InstantDailyLimit:      uint256.MustFromBig(instantDailyLimit),
		DailyLimits:            uint256.MustFromBig(dailyLimits),
		InstantFee:             uint256.MustFromBig(instantFee),
		TokenRate:              uint256.MustFromBig(tokenRate),
		MTokenRate:             uint256.MustFromBig(mTokenRate),
		MinAmount:              uint256.MustFromBig(minAmount),
		RedeemInstantFnPaused:  redeemInstantFnPaused,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{instantDailyLimit.String(), instantDailyLimit.String()}
	p.Timestamp = time.Now().Unix()

	return p, nil
}
