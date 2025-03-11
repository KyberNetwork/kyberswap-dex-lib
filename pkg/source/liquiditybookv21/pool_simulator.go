package liquiditybookv21

import (
	"math/big"
	"slices"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	blockTimestamp    uint64
	staticFeeParams   staticFeeParams
	variableFeeParams variableFeeParams
	activeBinID       uint32
	binStep           uint16
	bins              []BinU256
}

var _ = pool.RegisterFactory0(DexTypeLiquidityBookV21, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra ExtraU256
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
		}},
		blockTimestamp:    extra.RpcBlockTimestamp,
		staticFeeParams:   extra.StaticFeeParams,
		variableFeeParams: extra.VariableFeeParams,
		activeBinID:       extra.ActiveBinID,
		binStep:           extra.BinStep,
		bins:              extra.Bins,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut
	if err := p.validateTokens([]string{tokenAmountIn.Token, tokenOut}); err != nil {
		return nil, err
	}
	amountIn := tokenAmountIn.Amount
	swapForY := tokenAmountIn.Token == p.Info.Tokens[0]

	swapOutResult, err := p.getSwapOut(amountIn, swapForY)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: swapOutResult.Amount.ToBig(),
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: swapOutResult.Fee.ToBig(),
		},
		Gas: defaultGas,
		SwapInfo: SwapInfo{
			BinsReserveChanges: swapOutResult.BinsReserveChanges,
			NewParameters:      swapOutResult.Parameters,
			NewActiveID:        swapOutResult.NewActiveID,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenAmountOut := params.TokenIn, params.TokenAmountOut
	if err := p.validateTokens([]string{tokenIn, tokenAmountOut.Token}); err != nil {
		return nil, err
	}
	amountOut := tokenAmountOut.Amount
	swapForY := tokenIn == p.Info.Tokens[0]

	swapInResult, err := p.getSwapIn(amountOut, swapForY)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: swapInResult.Amount.ToBig(),
		},
		RemainingTokenAmountOut: &pool.TokenAmount{
			Token:  tokenAmountOut.Token,
			Amount: bignumber.ZeroBI,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: swapInResult.Fee.ToBig(),
		},
		Gas: defaultGas,
		SwapInfo: SwapInfo{
			BinsReserveChanges: swapInResult.BinsReserveChanges,
			NewParameters:      swapInResult.Parameters,
			NewActiveID:        swapInResult.NewActiveID,
		},
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = lo.Map(p.Info.Reserves, func(v *big.Int, i int) *big.Int {
		return new(big.Int).Set(v)
	})
	cloned.bins = slices.Clone(p.bins)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.WithFields(logger.Fields{
			"address": p.Info.Address,
		}).Warn("invalid swap info")
	}

	// update total reserves
	for idx, reserve := range p.Info.Reserves {
		if strings.EqualFold(p.Info.Tokens[idx], params.TokenAmountIn.Token) {
			p.Info.Reserves[idx].Add(reserve, params.TokenAmountIn.Amount)
		}
		if strings.EqualFold(p.Info.Tokens[idx], params.TokenAmountOut.Token) {
			p.Info.Reserves[idx].Sub(reserve, params.TokenAmountOut.Amount)
		}
	}

	// active bin ID
	p.activeBinID = swapInfo.NewActiveID

	// fee
	p.staticFeeParams = swapInfo.NewParameters.StaticFeeParams
	p.variableFeeParams = swapInfo.NewParameters.VariableFeeParams

	// update reserves of bins
	totalBinReserveChanges := make(map[uint32][2]*uint256.Int)
	for _, b := range swapInfo.BinsReserveChanges {
		changes, ok := totalBinReserveChanges[b.BinID]
		if !ok {
			totalBinReserveChanges[b.BinID] = [2]*uint256.Int{
				new(uint256.Int).Sub(b.AmountXIn, b.AmountXOut),
				new(uint256.Int).Sub(b.AmountYIn, b.AmountYOut)}
			continue
		}

		changes[0].Add(changes[0], b.AmountXIn).Sub(changes[0], b.AmountXOut)
		changes[1].Add(changes[1], b.AmountYIn).Sub(changes[1], b.AmountYOut)
	}
	newBins := p.bins[:0]
	for _, b := range p.bins {
		if changes, ok := totalBinReserveChanges[b.ID]; ok {
			b.ReserveX = changes[0].Add(b.ReserveX, changes[0])
			b.ReserveY = changes[1].Add(b.ReserveY, changes[1])
		}

		if !b.isEmpty() {
			newBins = append(newBins, b)
		}
	}
	p.bins = newBins
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) interface{} {
	return nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/LBPair.sol#L373
/**
 * @notice Simulates a swap in.
 * @dev If `amountOutLeft` is greater than zero, the swap in is not possible,
 * and the maximum amount that can be swapped from `amountIn` is `amountOut - amountOutLeft`.
 * @param amountOut The amount of token X or Y to swap in
 * @param swapForY Whether the swap is for token Y (true) or token X (false)
 * @return amountIn The amount of token X or Y that can be swapped in, including the fee
 * @return amountOutLeft The amount of token Y or X that cannot be swapped out
 * @return fee The fee of the swap
 */
func (p *PoolSimulator) getSwapIn(amountOut *big.Int, swapForY bool) (*swapResult, error) {
	amountsOutLeft, overflow := uint256.FromBig(amountOut)
	if overflow {
		return nil, ErrInvalidAmount
	}
	var (
		binStep            = p.binStep
		amountIn           uint256.Int
		swapFee            uint256.Int
		binsReserveChanges []binReserveChanges
	)

	params := p.copyParameters()
	id := params.ActiveBinID

	params = params.updateReferences(p.blockTimestamp)

	for {
		binArrIdx, err := p.findBinArrIndex(id)
		if err != nil {
			return nil, err
		}
		binReserves := p.bins[binArrIdx].decode(!swapForY)
		if binReserves.Sign() > 0 {
			price, err := getPriceFromID(id, binStep)
			if err != nil {
				return nil, err
			}

			var amountOutOfBin *uint256.Int
			if binReserves.Cmp(amountsOutLeft) > 0 {
				amountOutOfBin = amountsOutLeft
			} else {
				amountOutOfBin = binReserves
			}

			params = params.updateVolatilityAccumulator(id)

			var amountInWithoutFee *uint256.Int

			if swapForY {
				amountInWithoutFee, err = shiftDivRoundUp(amountOutOfBin, scaleOffset, price)
			} else {
				amountInWithoutFee, err = mulShiftRoundUp(amountOutOfBin, price, scaleOffset)
			}
			if err != nil {
				return nil, err
			}

			totalFees := params.getTotalFee(binStep)

			feeAmount, err := getFeeAmount(amountInWithoutFee, totalFees)
			if err != nil {
				return nil, err
			}

			amountIn.Add(&amountIn, new(uint256.Int).Add(amountInWithoutFee, feeAmount))
			amountsOutLeft.Sub(amountsOutLeft, amountOutOfBin)

			swapFee.Add(&swapFee, feeAmount)

			newBinReserveChanges := newBinReserveChanges(
				id, !swapForY, &amountIn, amountOutOfBin,
			)
			binsReserveChanges = append(binsReserveChanges, newBinReserveChanges)
		}

		if amountsOutLeft.IsZero() {
			break
		}

		nextID, err := p.getNextNonEmptyBin(swapForY, id)
		if err != nil {
			return nil, ErrNotFoundBinID
		}

		id = nextID
	}

	params.ActiveBinID = id

	ret := swapResult{
		Amount:             &amountIn,
		Fee:                &swapFee,
		BinsReserveChanges: binsReserveChanges,
		Parameters:         params,
		NewActiveID:        id,
	}

	return &ret, nil
}

// https://github.com/traderjoe-xyz/joe-v2/blob/main/src/LBPair.sol#L434
/**
 * @notice Simulates a swap out.
 * @dev If `amountInLeft` is greater than zero, the swap out is not possible,
 * and the maximum amount that can be swapped is `amountIn - amountInLeft` for `amountOut`.
 * @param amountIn The amount of token X or Y to swap in
 * @param swapForY Whether the swap is for token Y (true) or token X (false)
 * @return amountInLeft The amount of token X or Y that cannot be swapped in
 * @return amountOut The amount of token Y or X that can be swapped out
 * @return fee The fee of the swap
 */
func (p *PoolSimulator) getSwapOut(amountIn *big.Int, swapForY bool) (*swapResult, error) {
	amountsInLeft, overflow := uint256.FromBig(amountIn)
	if overflow {
		return nil, ErrInvalidAmount
	}
	var (
		binStep            = p.binStep
		amountOut          uint256.Int
		swapFee            uint256.Int
		binsReserveChanges []binReserveChanges
	)

	params := p.copyParameters()
	id := params.ActiveBinID

	params = params.updateReferences(p.blockTimestamp)

	for {
		binArrIdx, err := p.findBinArrIndex(id)
		if err != nil {
			return nil, err
		}
		binReserves := p.bins[binArrIdx]
		if !binReserves.isEmptyForSwap(!swapForY) {
			params = params.updateVolatilityAccumulator(id)

			amountsInWithFees, amountsOutOfBin, totalFees, err := binReserves.getAmounts(
				params, binStep, swapForY, id, amountsInLeft,
			)
			if err != nil {
				return nil, err
			}

			if amountsInWithFees.Sign() > 0 {
				amountsInLeft.Sub(amountsInLeft, amountsInWithFees)
				amountOut.Add(&amountOut, amountsOutOfBin)
				swapFee.Add(&swapFee, totalFees)

				pFee, err := scalarMulDivBasisPointRoundDown(
					totalFees,
					uint256.NewInt(uint64(p.staticFeeParams.ProtocolShare)),
				)
				if err != nil {
					return nil, err
				}
				amountsInWithFees.Sub(amountsInWithFees, pFee)
				newBinReserveChanges := newBinReserveChanges(
					id, !swapForY, amountsInWithFees, amountsOutOfBin,
				)
				binsReserveChanges = append(binsReserveChanges, newBinReserveChanges)
			}

		}

		if amountsInLeft.IsZero() {
			break
		}

		nextID, err := p.getNextNonEmptyBin(swapForY, id)
		if err != nil {
			return nil, ErrNotFoundBinID
		}

		id = nextID
	}

	params.ActiveBinID = id

	ret := swapResult{
		Amount:             &amountOut,
		Fee:                &swapFee,
		BinsReserveChanges: binsReserveChanges,
		Parameters:         params,
		NewActiveID:        id,
	}

	return &ret, nil
}

func (p *PoolSimulator) validateTokens(tokens []string) error {
	for _, t := range tokens {
		if p.GetTokenIndex(t) < 0 {
			return ErrInvalidToken
		}
	}
	return nil
}

func (p *PoolSimulator) copyParameters() *parameters {
	return &parameters{
		StaticFeeParams:   p.staticFeeParams,
		VariableFeeParams: p.variableFeeParams,
		ActiveBinID:       p.activeBinID,
	}
}

func (p *PoolSimulator) getNextNonEmptyBin(swapForY bool, id uint32) (uint32, error) {
	if swapForY {
		return p.findFirstRight(id)
	}

	return p.findFirstLeft(id)
}

func (p *PoolSimulator) findFirstRight(id uint32) (uint32, error) {
	idx, err := p.findBinArrIndex(id)
	if err != nil {
		return 0, err
	}
	if idx == 0 {
		return 0, ErrNotFoundBinID
	}
	return p.bins[idx-1].ID, nil
}

func (p *PoolSimulator) findFirstLeft(id uint32) (uint32, error) {
	idx, err := p.findBinArrIndex(id)
	if err != nil {
		return 0, err
	}
	if idx == uint32(len(p.bins)-1) {
		return 0, ErrNotFoundBinID
	}
	return p.bins[idx+1].ID, nil
}

func (p *PoolSimulator) findBinArrIndex(binID uint32) (uint32, error) {
	if len(p.bins) == 0 {
		return 0, ErrNotFoundBinID
	}

	var (
		l = 0
		r = len(p.bins)
	)

	for r-l > 1 {
		m := (r + l) >> 1
		if p.bins[m].ID <= binID {
			l = m
		} else {
			r = m
		}
	}

	if p.bins[l].ID != binID {
		return 0, ErrNotFoundBinID
	}

	return uint32(l), nil
}
