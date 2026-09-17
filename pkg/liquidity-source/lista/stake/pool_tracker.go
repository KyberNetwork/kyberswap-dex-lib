package stake

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
)

// rateProbeAmount is the fixed input passed to convertBnbToSnBnb/convertSnBnbToBnb, both of which
// are linear in their argument (amount * totalShares / totalPooledBnb, one floor division on-chain);
// probing with 1e18 yields a per-1e18-wei rate the simulator can reuse via a single mul-div.
var rateProbeAmount = big.NewInt(1e18)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
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
	startTime := time.Now()
	logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address}).
		Info("Start getting new pool state")

	var (
		paused           bool
		depositRate      *big.Int
		withdrawRate     *big.Int
		feeRate          *big.Int
		amountToDelegate *big.Int
		minBnb           *big.Int
		whitelistOff     bool
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}
	calls.AddCall(&ethrpc.Call{
		ABI: StakeManagerABI, Target: p.Address, Method: "paused",
	}, []any{&paused})
	calls.AddCall(&ethrpc.Call{
		ABI: StakeManagerABI, Target: p.Address, Method: "convertBnbToSnBnb",
		Params: []any{rateProbeAmount},
	}, []any{&depositRate})
	calls.AddCall(&ethrpc.Call{
		ABI: StakeManagerABI, Target: p.Address, Method: "convertSnBnbToBnb",
		Params: []any{rateProbeAmount},
	}, []any{&withdrawRate})
	calls.AddCall(&ethrpc.Call{
		ABI: StakeManagerABI, Target: p.Address, Method: "instantWithdrawFeeRate",
	}, []any{&feeRate})
	calls.AddCall(&ethrpc.Call{
		ABI: StakeManagerABI, Target: p.Address, Method: "amountToDelegate",
	}, []any{&amountToDelegate})
	calls.AddCall(&ethrpc.Call{
		ABI: StakeManagerABI, Target: p.Address, Method: "minBnb",
	}, []any{&minBnb})
	calls.AddCall(&ethrpc.Call{
		ABI: StakeManagerABI, Target: p.Address, Method: "instantWhitelistOff",
	}, []any{&whitelistOff})

	resp, err := calls.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address, "error": err}).
			Error("Failed to get new pool state")
		return p, err
	}

	extra := Extra{
		Paused:                  paused,
		DepositRate:             uint256.MustFromBig(depositRate),
		WithdrawRate:            uint256.MustFromBig(withdrawRate),
		InstantWithdrawFeeRate:  uint256.MustFromBig(feeRate),
		MinBnb:                  uint256.MustFromBig(minBnb),
		InstantWithdrawEligible: whitelistOff,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	// Reserves[0] (WBNB) doubles as the withdraw-direction liquidity cap: instantWithdraw pays out
	// of this same buffer on-chain (amountToDelegate -= bnbAmount, checked-underflow reverts).
	// Reserves[1] (slisBNB) is a large sentinel -- deposit always mints, so it's never capped.
	p.Reserves = entity.PoolReserves{amountToDelegate.String(), defaultDepositReserve}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = resp.BlockNumber.Uint64()

	logger.WithFields(logger.Fields{
		"dex_id": t.config.DexID, "pool_id": p.Address, "type": DexType,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Finished getting new pool state")

	return p, nil
}
