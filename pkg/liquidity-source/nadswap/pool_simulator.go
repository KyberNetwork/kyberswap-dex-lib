package nadswap

import (
	"math/big"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolSimulator struct {
	pool.Pool

	isMeme             bool
	quoteToken         common.Address
	creatorFeeRate     uint16
	dexProtocolFeeRate uint16
	feeRate            uint16 // creator + dexProtocol

	reserve0 *uint256.Int
	reserve1 *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	var se StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &se); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens: lo.Map(entityPool.Tokens, func(t *entity.PoolToken, _ int) string {
					return t.Address
				}),
				Reserves: lo.Map(entityPool.Reserves, func(r string, _ int) *big.Int {
					b, _ := new(big.Int).SetString(r, 10)
					return b
				}),
				BlockNumber: entityPool.BlockNumber,
			},
		},
		isMeme:             se.IsMemePair,
		quoteToken:         se.QuoteToken,
		creatorFeeRate:     se.CreatorFeeRate,
		dexProtocolFeeRate: se.DexProtocolFeeRate,
		feeRate:            se.CreatorFeeRate + se.DexProtocolFeeRate,
		reserve0:           extra.Reserve0.Clone(),
		reserve1:           extra.Reserve1.Clone(),
	}, nil
}

// tokenIsQuote compares a pool token address case-insensitively against the
// configured quote token. Used to detect buy direction (tokenIn == quoteToken)
// for meme pairs.
func (s *PoolSimulator) tokenIsQuote(token string) bool {
	return strings.EqualFold(token, s.quoteToken.Hex())
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	reserveIn, reserveOut := s.reserve0, s.reserve1
	if indexIn == 1 {
		reserveIn, reserveOut = s.reserve1, s.reserve0
	}

	var (
		amountOut *uint256.Int
		err       error
		gas       int64
		isBuy     bool
	)

	switch {
	case !s.isMeme:
		amountOut, err = getAmountOutGeneral(amountIn, reserveIn, reserveOut)
		gas = sellGas
	case s.tokenIsQuote(params.TokenAmountIn.Token):
		amountOut, err = getAmountOutMemeBuy(amountIn, reserveIn, reserveOut, s.feeRate)
		gas = buyGas
		isBuy = true
	default:
		amountOut, err = getAmountOutMemeSell(amountIn, reserveIn, reserveOut, s.feeRate)
		gas = sellGas
	}
	if err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return nil, ErrInsufficientOutput
	}

	// Compute post-swap reserves for SwapInfo, accounting for quote-token swap fees transferred to
	// FeeCollector before _update on meme pairs (see NadFunPair._collectFee).
	newReserveIn, newReserveOut := s.computeNewReserves(amountIn, amountOut, reserveIn, reserveOut, s.isMeme, isBuy)
	var newR0, newR1 *uint256.Int
	if indexIn == 0 {
		newR0, newR1 = newReserveIn, newReserveOut
	} else {
		newR0, newR1 = newReserveOut, newReserveIn
	}

	feeToken := params.TokenAmountIn.Token
	feeAmount := computeReportedFee(amountIn, s.isMeme, isBuy, s.feeRate)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: feeToken, Amount: feeAmount.ToBig()},
		Gas:            gas,
		SwapInfo:       SwapInfo{NewReserve0: newR0, NewReserve1: newR1},
	}, nil
}

// computeNewReserves projects post-swap reserves accounting for quote-token swap fees
// transferred to FeeCollector before _update (meme pairs only; general pairs and feeRate==0 use the simple delta).
//
//   - meme buy  (tokenIn == quoteToken): newReserveIn  = reserveIn  + amountIn  - swapFeeIn
//     newReserveOut = reserveOut - amountOut
//     where swapFeeIn  = floor(amountIn  * feeRate / BPS)
//   - meme sell (tokenIn != quoteToken): newReserveIn  = reserveIn  + amountIn
//     newReserveOut = reserveOut - amountOut - swapFeeOut
//     where swapFeeOut = floor(amountOut * feeRate / (BPS - LpFeeRate - feeRate))
//   - else: simple +amountIn / -amountOut
//
// Returns (newReserveIn, newReserveOut). All arithmetic uses floor (matches Solidity).
func (s *PoolSimulator) computeNewReserves(amountIn, amountOut, reserveIn, reserveOut *uint256.Int, isMeme, isBuy bool) (*uint256.Int, *uint256.Int) {
	newReserveIn := new(uint256.Int).Add(reserveIn, amountIn)
	newReserveOut := new(uint256.Int).Sub(reserveOut, amountOut)
	if !isMeme || s.feeRate == 0 {
		return newReserveIn, newReserveOut
	}
	uFee := uint256.NewInt(uint64(s.feeRate))
	if isBuy {
		// swapFeeIn = amountIn * feeRate / BPS (floor)
		var swapFeeIn uint256.Int
		big256.MulDivDown(&swapFeeIn, amountIn, uFee, uBPS)
		newReserveIn.Sub(newReserveIn, &swapFeeIn)
	} else {
		// swapFeeOut = amountOut * feeRate / (BPS - LpFeeRate - feeRate)
		var denom, swapFeeOut uint256.Int
		denom.Sub(uBPS, uLpFeeRate)
		denom.Sub(&denom, uFee)
		big256.MulDivDown(&swapFeeOut, amountOut, uFee, &denom)
		newReserveOut.Sub(newReserveOut, &swapFeeOut)
	}
	return newReserveIn, newReserveOut
}

// computeReportedFee returns the input-side fee for reporting (informational only).
//   - general:   amountIn * LpFeeRate / BPS  (feeRate is 0 for general pools)
//   - meme buy:  amountIn * (LpFeeRate + feeRate) / BPS
//   - meme sell: amountIn * LpFeeRate / BPS  (swap fee is on output and already netted)
func computeReportedFee(amountIn *uint256.Int, isMeme, isBuy bool, feeRate uint16) *uint256.Int {
	rate := uint64(LpFeeRate)
	if isMeme && isBuy {
		rate += uint64(feeRate)
	}
	var fee uint256.Int
	fee.Mul(amountIn, uint256.NewInt(rate))
	fee.Div(&fee, uBPS)
	return &fee
}

func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	indexIn := s.GetTokenIndex(params.TokenIn)
	indexOut := s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	amountOut, overflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	reserveIn, reserveOut := s.reserve0, s.reserve1
	if indexIn == 1 {
		reserveIn, reserveOut = s.reserve1, s.reserve0
	}

	var (
		amountIn *uint256.Int
		err      error
		gas      int64
		isBuy    bool
	)

	switch {
	case !s.isMeme:
		amountIn, err = getAmountInGeneral(amountOut, reserveIn, reserveOut)
		gas = sellGas
	case s.tokenIsQuote(params.TokenIn):
		amountIn, err = getAmountInMemeBuy(amountOut, reserveIn, reserveOut, s.feeRate)
		gas = buyGas
		isBuy = true
	default:
		amountIn, err = getAmountInMemeSell(amountOut, reserveIn, reserveOut, s.feeRate)
		gas = sellGas
	}
	if err != nil {
		return nil, err
	}

	newReserveIn, newReserveOut := s.computeNewReserves(amountIn, amountOut, reserveIn, reserveOut, s.isMeme, isBuy)
	var newR0, newR1 *uint256.Int
	if indexIn == 0 {
		newR0, newR1 = newReserveIn, newReserveOut
	} else {
		newR0, newR1 = newReserveOut, newReserveIn
	}

	feeAmount := computeReportedFee(amountIn, s.isMeme, isBuy, s.feeRate)

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: params.TokenIn, Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: params.TokenIn, Amount: feeAmount.ToBig()},
		Gas:           gas,
		SwapInfo:      SwapInfo{NewReserve0: newR0, NewReserve1: newR1},
	}, nil
}

// UpdateBalance applies the post-swap reserves carried in SwapInfo to the simulator.
// It is a no-op if SwapInfo is missing or has the wrong type, mirroring how other
// v2-fork integrations defensively guard against routing/state mismatches.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	if si.NewReserve0 != nil {
		s.reserve0 = si.NewReserve0
	}
	if si.NewReserve1 != nil {
		s.reserve1 = si.NewReserve1
	}
	if len(s.Info.Reserves) == 2 {
		s.Info.Reserves[0] = s.reserve0.ToBig()
		s.Info.Reserves[1] = s.reserve1.ToBig()
	}
}

// CloneState returns a deep copy of the simulator suitable for speculative routing.
// Reserves are cloned so mutations on the returned simulator do not affect the
// original; immutable scalar configuration is copied by value via struct copy.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	cloned.reserve0 = s.reserve0.Clone()
	cloned.reserve1 = s.reserve1.Clone()
	return &cloned
}

// GetMetaInfo returns a payload describing pool state at the time of quoting.
// Exposes block number only, matching other v2-fork integrations in this repo.
func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return struct {
		BlockNumber uint64 `json:"blockNumber"`
	}{
		BlockNumber: s.Info.BlockNumber,
	}
}
