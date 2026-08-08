package synthereum

import (
	"context"
	"errors"
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
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
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
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	if len(p.Tokens) != 2 {
		return p, errors.New("invalid pool tokens")
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var err error
	switch staticExtra.PoolType {
	case poolTypeMultiLP:
		p, err = t.getMultiLpPoolState(ctx, p, overrides)
	case poolTypeWrapper:
		p, err = t.getWrapperPoolState(ctx, p, &staticExtra, overrides)
	default:
		return p, ErrUnsupportedPoolType
	}
	if err != nil {
		return p, err
	}

	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}

// getMultiLpPoolState refreshes the state of a SynthereumMultiLpLiquidityPool.
// It probes the on-chain quoter with 1 whole unit of each input token; mint/redeem
// outputs are linear in the input amount for a fixed oracle price and fee, so the
// simulator can price any amount from these probes.
func (t *PoolTracker) getMultiLpPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	mintProbeIn := big256.TenPow(p.Tokens[0].Decimals)   // 1 whole collateral unit
	redeemProbeIn := big256.TenPow(p.Tokens[1].Decimals) // 1 whole synthetic unit

	var (
		mintOut, mintFee            *big.Int
		redeemOut, redeemFee        *big.Int
		maxCapacity, totalSynthetic *big.Int
		feePercentage               *big.Int
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	req.AddCall(&ethrpc.Call{
		ABI:    multiLpPoolABI,
		Target: p.Address,
		Method: poolMethodGetMintTradeInfo,
		Params: []any{mintProbeIn.ToBig()},
	}, []any{&mintOut, &mintFee})
	req.AddCall(&ethrpc.Call{
		ABI:    multiLpPoolABI,
		Target: p.Address,
		Method: poolMethodGetRedeemTradeInfo,
		Params: []any{redeemProbeIn.ToBig()},
	}, []any{&redeemOut, &redeemFee})
	req.AddCall(&ethrpc.Call{
		ABI:    multiLpPoolABI,
		Target: p.Address,
		Method: poolMethodMaxTokensCapacity,
	}, []any{&maxCapacity})
	req.AddCall(&ethrpc.Call{
		ABI:    multiLpPoolABI,
		Target: p.Address,
		Method: poolMethodTotalSyntheticTokens,
	}, []any{&totalSynthetic})
	req.AddCall(&ethrpc.Call{
		ABI:    multiLpPoolABI,
		Target: p.Address,
		Method: poolMethodFeePercentage,
	}, []any{&feePercentage})

	// TryAggregate: the trade-info probes may revert individually (e.g. the redeem
	// probe when outstanding synthetic supply is below 1 whole unit); in that case
	// the corresponding outputs stay nil and that trade side is disabled.
	resp, err := req.TryAggregate()
	if err != nil {
		return p, err
	}

	if maxCapacity == nil || totalSynthetic == nil || feePercentage == nil {
		return p, errors.New("failed to fetch multi-lp pool state")
	}

	extra := Extra{
		FeePercentage: fromBig(feePercentage),
		MaxSynthCap:   fromBig(maxCapacity),
		TotalSynth:    fromBig(totalSynthetic),
	}
	if mintOut != nil && mintFee != nil {
		extra.MintProbeIn = mintProbeIn
		extra.MintProbeOut = fromBig(mintOut)
		extra.MintProbeFee = fromBig(mintFee)
	}
	if redeemOut != nil && redeemFee != nil {
		extra.RedeemProbeIn = redeemProbeIn
		extra.RedeemProbeOut = fromBig(redeemOut)
		extra.RedeemProbeFee = fromBig(redeemFee)
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	// reserves: [collateral payable via redeem, synthetic mintable]
	collateralReserve := reserveZero
	if extra.RedeemProbeOut != nil && extra.RedeemProbeIn != nil && !extra.RedeemProbeIn.IsZero() {
		var r uint256.Int
		big256.MulDivDown(&r, extra.TotalSynth, extra.RedeemProbeOut, extra.RedeemProbeIn)
		collateralReserve = r.Dec()
	}
	p.Reserves = entity.PoolReserves{collateralReserve, extra.MaxSynthCap.Dec()}
	p.Extra = string(extraBytes)
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()

	return p, nil
}

// getWrapperPoolState refreshes the state of the fixed-rate wrapper. Wrapping is
// unbounded; unwrapping is bounded by the collateral the wrapper can redeem from
// the ERC4626 vault it deposits into: previewRedeem(balanceOf(wrapper)).
// Note: the wrapper keeps no idle collateral, so reading the collateral token
// balance of the wrapper would always return 0.
func (t *PoolTracker) getWrapperPoolState(
	ctx context.Context,
	p entity.Pool,
	staticExtra *StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	if staticExtra.Vault == "" {
		return p, errors.New("missing wrapper vault address")
	}

	var shares *big.Int
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	req.AddCall(&ethrpc.Call{
		ABI:    vaultABI,
		Target: staticExtra.Vault,
		Method: vaultMethodBalanceOf,
		Params: []any{common.HexToAddress(p.Address)},
	}, []any{&shares})

	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}
	if shares == nil {
		return p, errors.New("failed to fetch wrapper vault shares")
	}

	assets := big.NewInt(0)
	if shares.Sign() > 0 {
		var previewedAssets *big.Int
		previewReq := t.ethrpcClient.NewRequest().SetContext(ctx)
		if resp.BlockNumber != nil {
			// pin to the same block as the balanceOf call
			previewReq.SetBlockNumber(resp.BlockNumber)
		}
		if overrides != nil {
			previewReq.SetOverrides(overrides)
		}
		previewReq.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: staticExtra.Vault,
			Method: vaultMethodPreviewRedeem,
			Params: []any{shares},
		}, []any{&previewedAssets})
		if _, err = previewReq.Aggregate(); err != nil {
			return p, err
		}
		if previewedAssets == nil {
			return p, errors.New("failed to preview wrapper vault redeem")
		}
		assets = previewedAssets
	}

	extra := Extra{
		WrapperReserve: fromBig(assets),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	// reserves: [collateral payable via unwrap, placeholder for the unbounded wrap side]
	p.Reserves = entity.PoolReserves{extra.WrapperReserve.Dec(), defaultSynthReserve}
	p.Extra = string(extraBytes)
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func fromBig(b *big.Int) *uint256.Int {
	if b == nil {
		return nil
	}
	u, overflow := uint256.FromBig(b)
	if overflow {
		return nil
	}
	return u
}
