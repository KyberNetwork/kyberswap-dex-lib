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
	case PoolTypeMultiLP:
		p, err = t.getMultiLpPoolState(ctx, p, overrides)
	case PoolTypeWrapper:
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
// It reads the same inputs SynthereumMultiLpLiquidityPoolLib._calculateMint /
// _calculateRedeem use on-chain (the oracle Price, via the pool's own Finder ->
// SynthereumPriceFeed, and FeePercentage) so the simulator can reproduce mint/redeem
// exactly (PreciseUnitMath floor rounding) for any input size, rather than
// approximating from a single probed quote.
func (t *PoolTracker) getMultiLpPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	var (
		maxCapacity, totalSynthetic, feePercentage *big.Int
		finderAddr                                 common.Address
		priceIdentifier                            [32]byte
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
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
	req.AddCall(&ethrpc.Call{
		ABI:    multiLpPoolABI,
		Target: p.Address,
		Method: poolMethodSynthereumFinder,
	}, []any{&finderAddr})
	req.AddCall(&ethrpc.Call{
		ABI:    multiLpPoolABI,
		Target: p.Address,
		Method: poolMethodPriceFeedIdentifier,
	}, []any{&priceIdentifier})

	// TryBlockAndAggregate, not TryAggregate: both tolerate an individual call
	// reverting (which here only disables what depends on it, e.g. price resolution
	// below, rather than failing the whole refresh), but Multicall3.tryAggregate
	// returns only returnData, so Response.BlockNumber is left nil and the refresh
	// block cannot be propagated to entity.Pool. tryBlockAndAggregate returns the
	// block alongside the results.
	resp, err := req.TryBlockAndAggregate()
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

	// A zero finder/identifier means those two calls reverted under TryAggregate;
	// price resolution is skipped rather than dereferencing an unset value.
	if finderAddr != (common.Address{}) && priceIdentifier != ([32]byte{}) {
		if price, ok := t.getMultiLpPoolPrice(ctx, finderAddr, priceIdentifier, resp.BlockNumber, overrides); ok {
			extra.Price = price
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	// reserves: [collateral payable via redeem (gross, informational), synthetic mintable]
	collateralReserve := reserveZero
	if extra.Price != nil && !extra.Price.IsZero() {
		var r uint256.Int
		big256.MulDivDown(&r, extra.TotalSynth, extra.Price, big256.TenPow(36-p.Tokens[0].Decimals))
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

// getMultiLpPoolPrice resolves the pool's Finder -> SynthereumPriceFeed module and
// reads getLatestPrice(priceIdentifier), pinned to blockNumber (the block the rest
// of the pool's state was read at). Both hops are best-effort: a revert or a
// disagreeing Finder registration only disables mint/redeem pricing this refresh
// (MaxSynthCap/TotalSynth/FeePercentage still update), it does not fail the update.
func (t *PoolTracker) getMultiLpPoolPrice(
	ctx context.Context,
	finderAddr common.Address,
	priceIdentifier [32]byte,
	blockNumber *big.Int,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*uint256.Int, bool) {
	var priceFeedAddr common.Address
	finderReq := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		finderReq.SetBlockNumber(blockNumber)
	}
	if overrides != nil {
		finderReq.SetOverrides(overrides)
	}
	finderReq.AddCall(&ethrpc.Call{
		ABI:    finderABI,
		Target: finderAddr.Hex(),
		Method: finderMethodGetImplementationAddress,
		Params: []any{priceFeedInterfaceName},
	}, []any{&priceFeedAddr})
	// Direct eth_call, not multicall: see the getLatestPrice note below — the same
	// caller-identity constraint applies to the whole resolution chain, so both legs
	// are issued as plain calls from the zero address.
	if _, err := finderReq.Call(); err != nil || priceFeedAddr == (common.Address{}) {
		return nil, false
	}

	var price *big.Int
	priceReq := t.ethrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil {
		priceReq.SetBlockNumber(blockNumber)
	}
	if overrides != nil {
		priceReq.SetOverrides(overrides)
	}
	priceReq.AddCall(&ethrpc.Call{
		ABI:    priceFeedABI,
		Target: priceFeedAddr.Hex(),
		Method: priceFeedMethodGetLatestPrice,
		Params: []any{priceIdentifier},
	}, []any{&price})
	// SynthereumPriceFeed.getLatestPrice authenticates its caller: it resolves
	// PoolRegistry from the Finder and STATICCALLs back into msg.sender to check the
	// caller is a registered pool. Routed through Multicall3 that callback hits a
	// contract which does not implement the expected method and reverts, so the whole
	// read fails (tryAggregate yields success=false with empty returndata) even though
	// the identical call succeeds directly. Issued from the zero address the callback
	// lands on a non-contract and returns empty-success, so the price reads fine.
	// Hence Call() here — batching this into the pool's multicall silently disables
	// pricing and leaves multi-lp pools unquotable.
	if _, err := priceReq.Call(); err != nil || price == nil {
		return nil, false
	}

	return fromBig(price), true
}

// getWrapperPoolState refreshes the state of the fixed-rate wrapper. Wrapping is
// unbounded. Unwrapping is bounded on-chain by
// 'require(_synthTokenAmount <= totSynthToken_, "Synth tokens amount too high")'
// (FixedRateLendingWrapper.unwrap) — the wrapper's own totalSyntheticTokens() is
// therefore the binding, exact cap; the collateral redeemable from the ERC4626
// vault it deposits into (maxWithdraw(wrapper)) is tracked as a second, independent
// cap: the two diverge and either one binding causes an on-chain revert. Note it is
// maxWithdraw, not previewRedeem(balanceOf) -- previewRedeem values the wrapper's
// shares, but a Morpho vault lends most of its assets out, so the instantly
// redeemable amount is materially smaller (on Base today 453k vs 631k EURC) and
// unwrapping into that gap reverts NotEnoughLiquidity().
// Note: the wrapper keeps no idle collateral, so reading the collateral token
// balance of the wrapper would always return 0 — hence the vault-based read.
func (t *PoolTracker) getWrapperPoolState(
	ctx context.Context,
	p entity.Pool,
	staticExtra *StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	if staticExtra.Vault == "" {
		return p, errors.New("missing wrapper vault address")
	}

	var (
		maxWithdraw                *big.Int
		totalSynthetic, conversion *big.Int
		maxDeposit                 *big.Int
	)
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	req.AddCall(&ethrpc.Call{
		ABI:    vaultABI,
		Target: staticExtra.Vault,
		Method: vaultMethodMaxWithdraw,
		Params: []any{common.HexToAddress(p.Address)},
	}, []any{&maxWithdraw})
	req.AddCall(&ethrpc.Call{
		ABI:    vaultABI,
		Target: staticExtra.Vault,
		Method: vaultMethodMaxDeposit,
		Params: []any{common.HexToAddress(p.Address)},
	}, []any{&maxDeposit})
	req.AddCall(&ethrpc.Call{
		ABI:    wrapperABI,
		Target: p.Address,
		Method: wrapperMethodTotalSyntheticTokens,
	}, []any{&totalSynthetic})
	req.AddCall(&ethrpc.Call{
		ABI:    wrapperABI,
		Target: p.Address,
		Method: wrapperMethodConversionRate,
	}, []any{&conversion})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	if maxWithdraw == nil {
		return p, errors.New("failed to fetch wrapper vault withdrawable liquidity")
	}
	if totalSynthetic == nil || conversion == nil {
		return p, errors.New("failed to fetch wrapper accounting state")
	}

	extra := Extra{
		WrapperReserve:    fromBig(maxWithdraw),
		WrapperSynthCap:   fromBig(totalSynthetic),
		WrapperRate:       fromBig(conversion),
		WrapperMaxDeposit: fromBig(maxDeposit), // best-effort: nil leaves wrap() capacity-unchecked, matching pre-fix behavior
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	// reserves: [collateral payable via unwrap, synthetic mintable via wrap].
	// Reserves[0] mirrors the effective unwrap cap CalcAmountOut enforces (min of
	// WrapperSynthCap and the vault-derived reserve), not just the raw vault figure.
	// Reserves[1] uses the vault's real maxDeposit headroom (converted to synthetic
	// units) when available, falling back to a large placeholder otherwise -- wrap
	// has no on-chain cap beyond the vault's deposit capacity.
	unwrapCap := extra.WrapperSynthCap
	scalingFactor := big256.TenPow(p.Tokens[1].Decimals - p.Tokens[0].Decimals)
	if extra.WrapperReserve != nil {
		var vaultCap uint256.Int
		if _, overflow := vaultCap.MulOverflow(extra.WrapperReserve, scalingFactor); !overflow &&
			(unwrapCap == nil || vaultCap.Lt(unwrapCap)) {
			unwrapCap = &vaultCap
		}
	}
	wrapCapacity := defaultSynthReserve
	if extra.WrapperMaxDeposit != nil {
		var synthCap uint256.Int
		if _, overflow := synthCap.MulOverflow(extra.WrapperMaxDeposit, scalingFactor); !overflow {
			wrapCapacity = synthCap.Dec()
		}
	}
	unwrapCapReserve := reserveZero
	if unwrapCap != nil {
		var collateralEquivalent uint256.Int
		collateralEquivalent.Div(unwrapCap, scalingFactor)
		unwrapCapReserve = collateralEquivalent.Dec()
	}
	p.Reserves = entity.PoolReserves{unwrapCapReserve, wrapCapacity}
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
