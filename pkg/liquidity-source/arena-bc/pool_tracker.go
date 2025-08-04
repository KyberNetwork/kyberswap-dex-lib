package arenabc

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, params, nil)
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

	tokenId := staticExtra.TokenId
	tokenManager := t.config.TokenManager

	var (
		isPaused              bool
		canDeployLp           bool
		tokenParams           TokenParametersResult
		tokenBalance          *big.Int
		maxTokensForSale      *big.Int
		allowedTokenSupply    *big.Int
		protocolFeeBasisPoint uint8
		referralFeeBasisPoint uint8
		tokenSupply           *big.Int
	)

	req := t.ethrpcClient.NewRequest()
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    tokenManagerABI,
			Target: tokenManager,
			Method: tokenManagerMethodPaused,
		}, []any{&isPaused}).
		AddCall(&ethrpc.Call{
			ABI:    tokenManagerABI,
			Target: tokenManager,
			Method: tokenManagerMethodCanDeployLp,
		}, []any{&canDeployLp}).
		AddCall(&ethrpc.Call{
			ABI:    tokenManagerABI,
			Target: tokenManager,
			Method: tokenManagerMethodTokenParams,
			Params: []any{tokenId},
		}, []any{&tokenParams}).
		AddCall(&ethrpc.Call{
			ABI:    tokenManagerABI,
			Target: tokenManager,
			Method: tokenManagerMethodTokenBalanceOf,
			Params: []any{tokenId},
		}, []any{&tokenBalance}).
		AddCall(&ethrpc.Call{
			ABI:    tokenManagerABI,
			Target: tokenManager,
			Method: tokenManagerMethodGetMaxTokensForSale,
			Params: []any{tokenId},
		}, []any{&maxTokensForSale}).
		AddCall(&ethrpc.Call{
			ABI:    tokenManagerABI,
			Target: tokenManager,
			Method: tokenManagerMethodTokenSupply,
			Params: []any{tokenId},
		}, []any{&allowedTokenSupply}).
		AddCall(&ethrpc.Call{
			ABI:    tokenManagerABI,
			Target: tokenManager,
			Method: tokenManagerMethodProtocolFeeBasisPoint,
		}, []any{&protocolFeeBasisPoint}).
		AddCall(&ethrpc.Call{
			ABI:    tokenManagerABI,
			Target: tokenManager,
			Method: tokenManagerMethodReferralFeeBasisPoint,
		}, []any{&referralFeeBasisPoint}).
		AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: p.Tokens[1].Address,
			Method: abi.Erc20TotalSupplyMethod,
		}, []any{&tokenSupply})

	_, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	extra := Extra{
		IsPaused:              isPaused,
		CanDeployLp:           canDeployLp,
		TokenParams:           tokenParams.ToTokenParameters(),
		TokenBalance:          uint256.MustFromBig(tokenBalance),
		MaxTokensForSale:      uint256.MustFromBig(maxTokensForSale),
		AllowedTokenSupply:    uint256.MustFromBig(allowedTokenSupply),
		ProtocolFeeBasisPoint: protocolFeeBasisPoint,
		ReferralFeeBasisPoint: referralFeeBasisPoint,
		TokenSupply:           uint256.MustFromBig(tokenSupply),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = calculateReserves(&extra)

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Info("finish getting new state of pool")

	return p, nil
}

func calculateReserves(e *Extra) entity.PoolReserves {
	maxReward := integralFloor(
		new(uint256.Int).Div(e.TokenSupply, granularityScaler),
		new(uint256.Int),
		uint256.NewInt(uint64(e.TokenParams.A)),
		uint256.NewInt(uint64(e.TokenParams.B)),
		e.TokenParams.CurveScaler,
	)
	fee := getFee(
		maxReward,
		uint256.NewInt(uint64(e.ProtocolFeeBasisPoint)),
		uint256.NewInt(uint64(e.TokenParams.CreatorFeeBasisPoints)),
		uint256.NewInt(uint64(e.ReferralFeeBasisPoint)),
	)

	return entity.PoolReserves{
		maxReward.Sub(maxReward, fee).String(),
		e.MaxTokensForSale.String(),
	}
}
