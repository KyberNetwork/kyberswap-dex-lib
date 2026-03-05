package printr

import (
	"context"
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

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("start getting new state of pool")

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	// Early exit: pool already marked graduated (bonding curve finished). Skip RPC; return nil so downstream does not retry.
	if len(p.Extra) > 0 {
		var existing Extra
		if _ = json.Unmarshal([]byte(p.Extra), &existing); existing.Graduated {
			return p, nil
		}
	}

	printrAddr := staticExtra.PrintrAddr
	tokenAddr := common.HexToAddress(staticExtra.Token)

	var (
		curveResult GetCurveResult
		tradingFee  uint16
		isPaused    bool
	)

	req := t.ethrpcClient.NewRequest().SetOverrides(overrides).SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    printrABI,
			Target: printrAddr,
			Method: printrMethodGetCurve,
			Params: []any{tokenAddr},
		}, []any{&curveResult}).
		AddCall(&ethrpc.Call{
			ABI:    printrABI,
			Target: printrAddr,
			Method: printrMethodTradingFee,
		}, []any{&tradingFee}).
		AddCall(&ethrpc.Call{
			ABI:    printrABI,
			Target: printrAddr,
			Method: printrMethodPaused,
		}, []any{&isPaused})

	_, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	reserve := uint256.MustFromBig(curveResult.Reserve)
	completionThreshold := uint256.MustFromBig(curveResult.CompletionThreshold)

	// Bonding curve graduated (completionThreshold == 0): pool is permanently not swappable.
	// Unlike a generic tracker error (where downstream often does not save), here we still
	// return updated pool with Reserves set to "0","0" so pool indexing does not feed this
	// pool to path-finding anymore—downstream should persist this state and treat
	// ErrTokenGraduated as "permanently disabled" for routing. Extra.Graduated is set so
	// the next tracker cycle can early-return without RPC.
	if completionThreshold.IsZero() {
		extra := Extra{
			Reserve:             reserve,
			CompletionThreshold: completionThreshold,
			TradingFee:          tradingFee,
			Paused:              isPaused,
			Graduated:           true,
		}
		extraBytes, _ := json.Marshal(extra)
		p.Extra = string(extraBytes)
		p.Timestamp = time.Now().Unix()
		p.Reserves = entity.PoolReserves{"0", "0"}
		return p, nil
	}

	extra := Extra{
		Reserve:             reserve,
		CompletionThreshold: completionThreshold,
		TradingFee:          tradingFee,
		Paused:              isPaused,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	// Calculate reserves for routing:
	// reserves[0] = curve.reserve (base currency available for sells)
	// reserves[1] = tokenReserve - tokens available for buys
	maxTokenSupply, _ := uint256.FromDecimal(staticExtra.MaxTokenSupply)
	virtualReserve, _ := uint256.FromDecimal(staticExtra.VirtualReserve)

	tokenReserve := computeTokenReserve(maxTokenSupply, staticExtra.TotalCurves, virtualReserve, reserve)
	issuedSupply := new(uint256.Int).Sub(
		new(uint256.Int).Div(maxTokenSupply, uint256.NewInt(uint64(staticExtra.TotalCurves))),
		tokenReserve,
	)
	buyableTokens := new(uint256.Int)
	if completionThreshold.Gt(issuedSupply) {
		buyableTokens.Sub(completionThreshold, issuedSupply)
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		curveResult.Reserve.String(),
		buyableTokens.ToBig().String(),
	}

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Info("finish getting new state of pool")

	return p, nil
}

// computeTokenReserve calculates the current token reserve from curve parameters.
func computeTokenReserve(
	maxTokenSupply *uint256.Int,
	totalCurves uint16,
	virtualReserve *uint256.Int,
	reserve *uint256.Int,
) *uint256.Int {
	initialTokenReserve := new(uint256.Int).Div(maxTokenSupply, uint256.NewInt(uint64(totalCurves)))
	curveConstant := new(uint256.Int).Mul(virtualReserve, initialTokenReserve)
	vPlusR := new(uint256.Int).Add(virtualReserve, reserve)
	return new(uint256.Int).Div(curveConstant, vPlusR)
}
