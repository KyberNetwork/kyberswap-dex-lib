package doppleramm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	if err := cfg.withDefaults(); err != nil {
		return nil, err
	}
	return &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient}, nil
}

type marketReserves struct {
	ReserveX  *big.Int `abi:"reserveX"`
	ReserveY  *big.Int `abi:"reserveY"`
	Liquidity *big.Int `abi:"liquidity"`
}

type quoteResult struct {
	ActualInputAmount  *big.Int `abi:"actualInputAmount"`
	ActualOutputAmount *big.Int `abi:"actualOutputAmount"`
}

type protocolFee struct {
	SwapFeeShareBps uint16         `abi:"swapFeeShareBps"`
	SwapBaseFeeBps  uint16         `abi:"swapBaseFeeBps"`
	Account         common.Address `abi:"account"`
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}
	marketID, ok := new(big.Int).SetString(staticExtra.MarketID, 0)
	if !ok {
		return p, ErrUnsupportedMarket
	}

	var reserves marketReserves
	var fee protocolFee
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: ammABI, Target: staticExtra.AMM, Method: "reserves", Params: []any{marketID}}, []any{&reserves}).
		AddCall(&ethrpc.Call{ABI: ammABI, Target: staticExtra.AMM, Method: "protocolFeeForMarket", Params: []any{marketID}}, []any{&fee}).
		TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	if len(resp.Result) != 2 || !resp.Result[0] || !resp.Result[1] {
		return p, ErrCallReverted
	}
	if reserves.ReserveX == nil || reserves.ReserveY == nil ||
		reserves.ReserveX.Sign() <= 0 || reserves.ReserveY.Sign() <= 0 {
		return p, ErrInsufficientLiquidity
	}

	lpFeeBps, err := t.deriveLpFee(ctx, staticExtra.Quoter, marketID, reserves, fee.SwapBaseFeeBps, resp.BlockNumber)
	if err != nil {
		return p, err
	}

	extra, err := json.Marshal(Extra{TokenX: staticExtra.TokenX, BaseFeeBps: fee.SwapBaseFeeBps, LpFeeBps: lpFeeBps})
	if err != nil {
		return p, err
	}
	p.Extra = string(extra)
	p.Reserves = entity.PoolReserves{reserves.ReserveX.String(), reserves.ReserveY.String()}
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()
	return p, nil
}

// probeDivisors are tried in order. A probe must move the output by more than a
// wei per basis point to pin the fee down: 1% of reserveX is enough for any
// market whose other side holds more than dust, and 10% is the fallback.
var probeDivisors = []int64{100, 10}

// deriveLpFee recovers the module's LP fee from quoter probes. The fee is fixed
// per market and lives in the component's config pointer, so a probe is the only
// source available to an off-chain integration.
func (t *PoolTracker) deriveLpFee(
	ctx context.Context,
	quoter string,
	marketID *big.Int,
	reserves marketReserves,
	baseFeeBps uint16,
	blockNumber *big.Int,
) (uint16, error) {
	var lastErr error
	for _, div := range probeDivisors {
		probeAmount := new(big.Int).Div(reserves.ReserveX, big.NewInt(div))
		if probeAmount.Sign() <= 0 {
			continue
		}

		var quote quoteResult
		resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber).
			AddCall(&ethrpc.Call{
				ABI:    quoterABI,
				Target: quoter,
				Method: "quoteSwapExactInput",
				Params: []any{marketID, true, probeAmount, []byte{}, []byte{}},
			}, []any{&quote}).
			TryAggregate()
		if err != nil {
			return 0, err
		}
		if len(resp.Result) != 1 || !resp.Result[0] || quote.ActualOutputAmount == nil {
			return 0, ErrCallReverted
		}

		lpFeeBps, err := deriveLpFeeBps(probeAmount, quote.ActualOutputAmount, reserves.ReserveX, reserves.ReserveY, baseFeeBps)
		if err == nil {
			return lpFeeBps, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = ErrFeeDerivation
	}
	return 0, lastErr
}

// deriveLpFeeBps solves the module LP fee from one (amountIn, amountOut) probe.
func deriveLpFeeBps(probeAmount, actualOut, reserveX, reserveY *big.Int, baseFeeBps uint16) (uint16, error) {
	if actualOut.Sign() <= 0 || actualOut.Cmp(reserveY) >= 0 {
		return 0, ErrFeeDerivation
	}

	// out = reserveY*a/(reserveX+a)  =>  a = out*reserveX/(reserveY-out)
	effIn := new(big.Int).Mul(actualOut, reserveX)
	effIn.Div(effIn, new(big.Int).Sub(reserveY, actualOut))

	afterBase := new(big.Int).Sub(probeAmount,
		new(big.Int).Div(new(big.Int).Mul(probeAmount, big.NewInt(int64(baseFeeBps))), big.NewInt(bpsDenominator)))
	if afterBase.Sign() <= 0 {
		return 0, ErrFeeDerivation
	}

	lpBps := new(big.Int).Sub(afterBase, effIn)
	lpBps.Mul(lpBps, big.NewInt(bpsDenominator))
	lpBps.Div(lpBps, afterBase)
	if lpBps.Sign() < 0 || !lpBps.IsInt64() || lpBps.Int64() > bpsDenominator {
		return 0, ErrFeeDerivation
	}

	// Integer floors in the derivation can land one bps off; accept only a
	// candidate that reproduces the probe exactly.
	probeIn, overflow := uint256.FromBig(probeAmount)
	reserveIn, overflowIn := uint256.FromBig(reserveX)
	reserveOut, overflowOut := uint256.FromBig(reserveY)
	if overflow || overflowIn || overflowOut {
		return 0, ErrOverflow
	}

	candidate := uint16(lpBps.Int64())
	// Integer floors in the derivation can land a bps off, and a probe too small
	// to move the output by a wei per bps leaves two candidates. Accept a single
	// value that reproduces the probe exactly, and let the caller retry with a
	// bigger probe when the answer is ambiguous.
	var found uint16
	matches := 0
	for _, c := range []uint16{candidate - 1, candidate, candidate + 1} {
		if c > bpsDenominator {
			continue
		}
		out, err := getAmountOut(probeIn, reserveIn, reserveOut, baseFeeBps, c)
		if err == nil && out.ToBig().Cmp(actualOut) == 0 {
			found = c
			matches++
		}
	}
	if matches != 1 {
		return 0, ErrFeeDerivation
	}
	return found, nil
}
