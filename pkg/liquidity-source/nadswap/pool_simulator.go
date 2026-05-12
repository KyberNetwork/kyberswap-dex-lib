package nadswap

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

	// Compute post-swap reserves for SwapInfo. In NadFunPair the input-side LP fee is *kept in the pool*
	// (k-invariant uses balance adjusted only by LP fee), but for off-chain quote we model reserves
	// updating to reserveIn += amountIn and reserveOut -= amountOut (matching how aggregators chain
	// quotes for routing).
	newReserveIn := new(uint256.Int).Add(reserveIn, amountIn)
	newReserveOut := new(uint256.Int).Sub(reserveOut, amountOut)
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

// computeReportedFee returns the input-side fee for reporting (informational only).
//   - general / meme buy: amountIn * (LP + feeRate) / BPS
//   - meme sell:           amountIn * LP / BPS  (swap fee is on output and already netted)
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

// UpdateBalance is a placeholder satisfying IPoolSimulator. The real implementation
// (consuming SwapInfo.NewReserve0/NewReserve1) lands in Task 8.
func (s *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

// GetMetaInfo returns pool metadata for the swap router. Currently exposes block number only.
func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return struct {
		BlockNumber uint64 `json:"blockNumber"`
	}{
		BlockNumber: s.Info.BlockNumber,
	}
}
