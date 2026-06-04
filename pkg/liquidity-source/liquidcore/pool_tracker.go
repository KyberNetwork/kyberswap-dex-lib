package liquidcore

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
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context,
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
	logger.Infof("getting new state for pool %v", p.Address)
	defer logger.Infof("finished getting new state for pool %v", p.Address)

	var (
		reserves struct {
			Reserve0 *big.Int
			Reserve1 *big.Int
		}
		spotPrices struct {
			ForwardPrice uint64
			InversePrice uint64
		}
		calibAmtOut *big.Int
	)

	// Calibration: call estimateSwap(token1→token0) with 5% of the previous reserve1.
	// The contract's internal oracle for swaps may differ from getSpotPrices (which normalizes
	// by a USD stablecoin price). Back-calculating SpotPrice from estimateSwap gives the
	// oracle value the swap formula actually needs.
	// Fee effect: calibAmtOut is net-of-fee, so derived SpotPrice underestimates by ~fee_bps/scale
	// (e.g., 0.025% for baseFee=25). This is further refined by derivedSpotPrice via fixed-point iteration over CalcSwap.
	calibAmtIn := calibrationAmountIn(p)

	var token0Addr, token1Addr common.Address
	if len(p.Tokens) >= 2 {
		token0Addr = common.HexToAddress(p.Tokens[0].Address)
		token1Addr = common.HexToAddress(p.Tokens[1].Address)
	}

	resp, err := t.ethrpcClient.R().SetContext(ctx).SetOverrides(overrides).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "getReserves",
		}, []any{&reserves}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "getSpotPrices",
		}, []any{&spotPrices}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "estimateSwap",
			Params: []any{token1Addr, token0Addr, calibAmtIn},
		}, []any{&calibAmtOut}).
		TryBlockAndAggregate()
	if err != nil {
		logger.Errorf("failed to aggregate pool state: %v", err)
		return p, err
	}

	// Derive SpotPrice from calibration result, falling back to getSpotPrices on failure.
	// SP = calibAmtOut * scale * 10^dec1 / (calibAmtIn * 10^dec0)
	// (isFromToken0=false formula inverted)
	spotPrice := derivedSpotPrice(p, reserves, spotPrices.ForwardPrice, calibAmtIn, calibAmtOut)

	extraBytes, err := json.Marshal(Extra{
		SpotPrice: spotPrice,
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Reserves = entity.PoolReserves{reserves.Reserve0.String(), reserves.Reserve1.String()}

	return p, nil
}

// calibrationAmountIn returns 5% of the previous reserve1, clamped to at least 1e12.
func calibrationAmountIn(p entity.Pool) *big.Int {
	amt := new(big.Int)
	if len(p.Reserves) >= 2 {
		r1 := new(big.Int)
		if _, ok := r1.SetString(p.Reserves[1], 10); ok && r1.Sign() > 0 {
			amt.Mul(r1, bignumber.Five)
			amt.Div(amt, bignumber.B100)
		}
	}
	if amt.Sign() == 0 {
		amt.SetUint64(1e12) // fallback for newly-listed pools with no prior reserve
	}
	return amt
}

// derivedSpotPrice computes SpotPrice from the calibration estimateSwap call.
// Inverts: rawOut = amtIn * decOutScale * SP / (scale * decInScale)
// => SP = calibAmtOut * scale * decInScale / (calibAmtIn * decOutScale)
// Then refines SP via fixed-point iteration over CalcSwap: sp *= amtOut/res.AmountOut
// until CalcSwap reproduces the expected output (up to 6 iterations).
// Falls back to forwardPrice if calibration gave no useful result.
func derivedSpotPrice(p entity.Pool, reserves struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
}, forwardPrice uint64, calibAmtIn, calibAmtOut *big.Int) *uint256.Int {
	lg := log.Ctx(context.Background())
	if calibAmtOut == nil || calibAmtOut.Sign() <= 0 || len(p.Tokens) < 2 {
		lg.Debug().Str("pool", p.Address).
			Stringer("calibAmtOut", calibAmtOut).
			Uint64("calibAmtIn", calibAmtIn.Uint64()).
			Msg("calibration failed, falling back to forward price")
		return uint256.NewInt(forwardPrice)
	}

	dec0, dec1 := p.Tokens[0].Decimals, p.Tokens[1].Decimals
	decInScale, decOutScale := big256.TenPow(dec1), big256.TenPow(dec0)
	scale := u1e6
	if dec0+dec1 > 25 {
		scale = u1e10
	}

	// calibAmtOut * scale * decInScale / (calibAmtIn * decOutScale)
	amtIn, amtOut := big256.FromBig(calibAmtIn), big256.FromBig(calibAmtOut)
	var num, denom uint256.Int
	sp := big256.MulDivDown(&num, num.Mul(amtOut, scale), decInScale, denom.Mul(amtIn, decOutScale))
	if sp.Sign() <= 0 {
		return uint256.NewInt(forwardPrice)
	}

	poolState := &PoolState{
		Token0:    p.Tokens[0].Address,
		Decimals0: dec0,
		Decimals1: dec1,
		Reserve0:  big256.FromBig(reserves.Reserve0),
		Reserve1:  big256.FromBig(reserves.Reserve1),
		SpotPrice: sp,
	}
	var lastRes *uint256.Int
	for range 6 {
		res, err := CalcSwap(poolState, p.Tokens[1].Address, amtIn)
		if err != nil {
			return uint256.NewInt(forwardPrice)
		}
		lg.Debug().Str("pool", p.Address).
			Stringer("res.AmountOut", res.AmountOut).
			Stringer("amtOut", amtOut).
			Msg("calibration result")
		if amtOut.Eq(res.AmountOut) || (lastRes != nil && lastRes.Eq(res.AmountOut)) {
			return sp
		}
		big256.MulDivDown(sp, sp, amtOut, res.AmountOut)
		lastRes = res.AmountOut
	}

	return sp
}
