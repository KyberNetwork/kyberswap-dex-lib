package midas

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client

	rvConfigs map[string]RvConfig
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		rvConfigs:    getRvConfig(config.MTokens),
	}, nil
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	lg := logger.WithFields(logger.Fields{
		"pool": p.Address,
	})

	lg.Infof("start getting new state of pool")
	defer func() {
		lg.Infof("finished getting new state of pool")
	}()

	currentDayNumber := time.Now().Unix() / oneDayInSecond

	mToken := p.Tokens[0].Address
	tokens := lo.Map(p.Tokens[1:], func(token *entity.PoolToken, _ int) string {
		return token.Address
	})

	var (
		vaultState *VaultStateRpcResult
		err        error
	)
	if staticExtra.IsDv {
		vaultState, err = t.getDvState(ctx, p.Address, mToken, tokens, currentDayNumber)
	} else {
		rvCfg, ok := t.rvConfigs[p.Address]
		if !ok {
			lg.Errorf("failed to find rvConfig")
			return p, nil
		}
		vaultState, err = t.getRvState(ctx, rvCfg, tokens, currentDayNumber)
	}
	if err != nil {
		return p, nil
	}

	extraBytes, err := json.Marshal(vaultState.ToVaultState(mToken, staticExtra.VaultType))
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)

	p.Reserves = []string{
		convertFromBase18(uint256.MustFromBig(vaultState.InstantDailyLimit), p.Tokens[0].Decimals).String(),
	}

	var tokenLimits []string
	for i := 0; i < len(tokens); i++ {
		tokenLimits = append(
			tokenLimits,
			convertFromBase18(uint256.MustFromBig(vaultState.TokensConfig[i].Allowance), p.Tokens[i+1].Decimals).String())
	}
	p.Reserves = append(p.Reserves, tokenLimits...)

	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) initVaultCalls(req *ethrpc.Request, vault string, tokens []string, result *VaultStateRpcResult,
	isDv bool, currentDayNumber int64) *ethrpc.Request {

	vaultAbi := lo.Ternary(isDv, DepositVaultABI, RedemptionVaultABI)
	fnSelector := lo.Ternary(isDv, depositInstantSelector, redeemInstantSelector)

	req.AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vGetPaymentTokensMethod,
	}, []any{&result.PaymentTokens}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vPausedMethod,
	}, []any{&result.Paused}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vFnPausedMethod,
		Params: []any{fnSelector},
	}, []any{&result.FnPaused}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vInstantDailyLimitMethod,
	}, []any{&result.InstantDailyLimit}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vDailyLimitsMethod,
		Params: []any{big.NewInt(currentDayNumber)},
	}, []any{&result.DailyLimits}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vInstantFeeMethod,
	}, []any{&result.InstantFee}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vMinAmountMethod,
	}, []any{&result.MinAmount}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vMTokenDataFeedMethod,
	}, []any{&result.MTokenDataFeed}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vWaivedFeeRestrictionMethod,
		Params: []any{common.HexToAddress(t.config.Executor)},
	}, []any{&result.WaivedFeeRestriction})

	result.TokensConfig = make([]TokenConfigRpcResult, len(tokens))
	for i, token := range tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    vaultAbi,
			Target: vault,
			Method: vTokensConfigMethod,
			Params: []any{common.HexToAddress(token)},
		}, []any{&result.TokensConfig[i]})
	}

	return req
}

func (t *PoolTracker) getDvState(ctx context.Context, vault, mToken string, tokens []string,
	currentDayNumber int64) (*VaultStateRpcResult, error) {
	lg := logger.WithFields(logger.Fields{"dv": vault})

	var result VaultStateRpcResult
	result.MToken = mToken

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req = t.initVaultCalls(req, vault, tokens, &result, true, currentDayNumber)
	req.AddCall(&ethrpc.Call{
		ABI:    DepositVaultABI,
		Target: vault,
		Method: dvMinMTokenAmountForFirstDepositMethod,
	}, []any{&result.MinMTokenAmountForFirstDeposit}).AddCall(&ethrpc.Call{
		ABI:    DepositVaultABI,
		Target: vault,
		Method: dvTotalMintedMethod,
		Params: []any{common.HexToAddress(t.config.Executor)},
	}, []any{&result.TotalMinted}).AddCall(&ethrpc.Call{
		ABI:    DepositVaultABI,
		Target: vault,
		Method: dvMaxSupplyCapMethod,
	}, []any{&result.MaxSupplyCap}).AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: mToken,
		Method: abi.Erc20TotalSupplyMethod,
	}, []any{&result.MTokenTotalSupply})

	resp, err := req.TryAggregate()
	if err != nil {
		lg.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate dv state, mToken %v, vault %v", mToken, vault)
		return nil, err
	}

	req = t.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		SetBlockNumber(resp.BlockNumber).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: result.MTokenDataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&result.MTokenRate})
	result.TokenRates = make([]*big.Int, len(tokens))
	for i, _ := range tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: result.TokensConfig[i].DataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&result.TokenRates[i]})
	}

	_, err = req.Aggregate()
	if err != nil {
		lg.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate dv data feed state, mToken %v", mToken)
		return nil, err
	}

	return &result, nil
}

func (t *PoolTracker) getRvState(ctx context.Context, rvCfg RvConfig, tokens []string,
	currentDayNumber int64) (*VaultStateRpcResult, error) {
	lg := logger.WithFields(logger.Fields{
		"rv": rvCfg.Address,
	})

	var result VaultStateRpcResult
	result.MToken = rvCfg.MToken

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req = t.initVaultCalls(req, rvCfg.Address, tokens, &result, true, currentDayNumber)

	result.TokenBalances = make([]*big.Int, len(tokens))
	for i, token := range tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: token,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(rvCfg.Address)},
		}, []any{&result.TokenBalances[i]})
	}
	resp, err := req.Aggregate()
	if err != nil {
		lg.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate rv state, type %v", rvCfg.RvType)
		return nil, err
	}

	switch rvCfg.RvType {
	case redemptionVault:
	case redemptionVaultSwapper:
		var (
			liquidityProvider     common.Address
			mTbillRedemptionVault common.Address
		)
		if _, err = t.ethrpcClient.
			NewRequest().
			SetContext(ctx).
			SetBlockNumber(resp.BlockNumber).
			AddCall(&ethrpc.Call{
				ABI:    RedemptionVaultABI,
				Target: rvCfg.Address,
				Method: rvSwapperLiquidityProviderMethod,
			}, []any{&liquidityProvider}).
			AddCall(&ethrpc.Call{
				ABI:    RedemptionVaultABI,
				Target: rvCfg.Address,
				Method: rvSwapperMTbillRedemptionVaultMethod,
			}, []any{&mTbillRedemptionVault}).
			Aggregate(); err != nil {
			lg.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to get liquidityProvider and mTbillRedemptionVault")
			return nil, err
		}

		mTbillRv := strings.ToLower(mTbillRedemptionVault.String())
		mTbillRvCfg, ok := t.rvConfigs[mTbillRv]
		if !ok {
			return nil, fmt.Errorf("rvConfig not found %v", mTbillRv)
		}

		req = t.ethrpcClient.
			NewRequest().
			SetContext(ctx).
			SetBlockNumber(resp.BlockNumber).
			AddCall(&ethrpc.Call{
				ABI:    abi.Erc20ABI,
				Target: t.rvConfigs[mTbillRv].MToken,
				Method: abi.Erc20BalanceOfMethod,
				Params: []any{liquidityProvider},
			}, []any{&result.MToken2Balance})
		_, err = req.Aggregate()
		if err != nil {
			lg.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to get liquidityProvider and mTbillRedemptionVault")
			return nil, err
		}

		req = t.ethrpcClient.
			NewRequest().
			SetContext(ctx).
			SetBlockNumber(resp.BlockNumber).
			AddCall(&ethrpc.Call{
				ABI:    dataFeedABI,
				Target: result.MTokenDataFeed.String(),
				Method: dataFeedGetDataInBase18Method,
			}, []any{&result.MTokenRate})

		result.TokenRates = make([]*big.Int, len(tokens))
		for i := range tokens {
			if eth.IsZeroAddress(result.TokensConfig[i].DataFeed) {
				continue
			}

			req.AddCall(&ethrpc.Call{
				ABI:    dataFeedABI,
				Target: result.TokensConfig[i].DataFeed.String(),
				Method: dataFeedGetDataInBase18Method,
			}, []any{&result.TokenRates[i]})
		}
		_, err = req.Aggregate()
		if err != nil {
			lg.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to aggregate data feed rates for rv swapper")
			return nil, err
		}

		result.SwapperVaultType = mTbillRvCfg.RvType
		result.MTbillRedemptionVault, err = t.getRvState(ctx, mTbillRvCfg, tokens, currentDayNumber)
		if err != nil {
			lg.WithFields(logger.Fields{
				"error": err,
			}).Warnf("failed to aggregate mTbillRedemptionVault state for rv swapper, mTbillRedemptionVault %v, type %v",
				mTbillRedemptionVault, mTbillRvCfg.RvType)
		}

	case redemptionVaultUstb:
		var ustbRedemption common.Address
		req = t.ethrpcClient.
			NewRequest().
			SetContext(ctx).
			SetBlockNumber(resp.BlockNumber).
			AddCall(&ethrpc.Call{
				ABI:    redemptionVaultWithUstbABI,
				Target: rvCfg.Address,
				Method: rvUstbUstbRedemptionMethod,
			}, []any{&ustbRedemption})
		_, err = req.Call()
		if err != nil {
			lg.Errorf("failed to get ustbRedemption for rv ustb")
			return nil, err
		}

		var superstateToken common.Address
		req = t.ethrpcClient.
			NewRequest().
			SetContext(ctx).
			SetBlockNumber(resp.BlockNumber).
			AddCall(&ethrpc.Call{
				ABI:    redemptionABI,
				Target: ustbRedemption.String(),
				Method: redemptionSuperstateTokenMethod,
			}, []any{&superstateToken})
		_, err = req.Call()
		if err != nil {
			lg.Errorf("failed to get superstate token for redemption %v", ustbRedemption)
			return nil, err
		}

		req.AddCall(&ethrpc.Call{
			ABI:    redemptionABI,
			Target: ustbRedemption.String(),
			Method: redemptionUsdcMethod,
		}, []any{&result.Redemption.Usdc}).
			AddCall(&ethrpc.Call{
				ABI:    redemptionABI,
				Target: ustbRedemption.String(),
				Method: redemptionRedemptionFeeMethod,
			}, []any{&result.Redemption.RedemptionFee}).
			AddCall(&ethrpc.Call{
				ABI:    abi.Erc20ABI,
				Target: superstateToken.String(),
				Method: abi.Erc20BalanceOfMethod,
				Params: []any{common.HexToAddress(rvCfg.Address)},
			}, []any{&result.Redemption.UstbBalance}).
			AddCall(&ethrpc.Call{
				ABI:    redemptionABI,
				Target: ustbRedemption.String(),
				Method: redemptionGetChainlinkPriceMethod,
			}, []any{&result.Redemption.ChainlinkPrice}).
			AddCall(&ethrpc.Call{
				ABI:    redemptionABI,
				Target: ustbRedemption.String(),
				Method: redemptionChainlinkFeedPrecisionMethod,
			}, []any{&result.Redemption.ChainLinkFeedPrecision}).
			AddCall(&ethrpc.Call{
				ABI:    redemptionABI,
				Target: ustbRedemption.String(),
				Method: redemptionSuperstateTokenPrecisionMethod,
			}, []any{&result.Redemption.SuperstateTokenPrecision})
		_, err = req.Aggregate()
		if err != nil {
			lg.Errorf("failed to aggregate redemption state %v", ustbRedemption)
			return nil, err
		}

		req = t.ethrpcClient.
			NewRequest().
			SetContext(ctx).
			SetBlockNumber(resp.BlockNumber).
			AddCall(&ethrpc.Call{
				ABI:    dataFeedABI,
				Target: result.MTokenDataFeed.String(),
				Method: dataFeedGetDataInBase18Method,
			}, []any{&result.MTokenRate})

		result.TokenRates = make([]*big.Int, len(tokens))
		for i := range tokens {
			if eth.IsZeroAddress(result.TokensConfig[i].DataFeed) {
				continue
			}

			req.AddCall(&ethrpc.Call{
				ABI:    dataFeedABI,
				Target: result.TokensConfig[i].DataFeed.String(),
				Method: dataFeedGetDataInBase18Method,
			}, []any{&result.TokenRates[i]})
		}
		_, err = req.TryAggregate()
		if err != nil {
			lg.WithFields(logger.Fields{
				"error": err,
			}).Error("failed to aggregate data feed rates for rv ustb")
			return nil, err
		}
	default:
		return nil, ErrNotSupported
	}

	return &result, nil
}
