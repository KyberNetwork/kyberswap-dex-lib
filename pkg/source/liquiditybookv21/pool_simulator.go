package liquiditybookv21

import (
	"math/big"
	"sort"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

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
	bins              []Bin
}

var _ = pool.RegisterFactory0(DexTypeLiquidityBookV21, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var (
		tokens   = make([]string, 2)
		reserves = make([]*big.Int, 2)

		extra Extra
	)

	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])

		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])
	}

	err := json.Unmarshal([]byte(entityPool.Extra), &extra)
	if err != nil {
		return nil, err
	}

	info := pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		SwapFee:    nil,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
		Checked:    false,
	}

	return &PoolSimulator{
		Pool:              pool.Pool{Info: info},
		blockTimestamp:    extra.RpcBlockTimestamp,
		staticFeeParams:   extra.StaticFeeParams,
		variableFeeParams: extra.VariableFeeParams,
		activeBinID:       extra.ActiveBinID,
		binStep:           extra.BinStep,
		bins:              extra.Bins,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := params.TokenAmountIn
	tokenOut := params.TokenOut
	err := p.validateTokens([]string{tokenAmountIn.Token, tokenOut})
	if err != nil {
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
			Amount: swapOutResult.AmountOut,
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: swapOutResult.Fee,
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
	tokenIn := params.TokenIn
	tokenAmountOut := params.TokenAmountOut
	err := p.validateTokens([]string{tokenIn, tokenAmountOut.Token})
	if err != nil {
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
			Amount: swapInResult.AmountIn,
		},
		RemainingTokenAmountOut: &pool.TokenAmount{
			Token:  tokenAmountOut.Token,
			Amount: bignumber.ZeroBI,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: swapInResult.Fee,
		},
		Gas: defaultGas,
		SwapInfo: SwapInfo{
			BinsReserveChanges: swapInResult.BinsReserveChanges,
			NewParameters:      swapInResult.Parameters,
			NewActiveID:        swapInResult.NewActiveID,
		},
	}, nil
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
	totalBinReserveChanges := make(map[uint32]binReserveChanges)
	for _, b := range swapInfo.BinsReserveChanges {
		changes, ok := totalBinReserveChanges[b.BinID]
		if !ok {
			changes = binReserveChanges{
				BinID:      b.BinID,
				AmountXIn:  new(big.Int),
				AmountXOut: new(big.Int),
				AmountYIn:  new(big.Int),
				AmountYOut: new(big.Int),
			}
		}
		changes.AmountXIn.Add(changes.AmountXIn, b.AmountXIn)
		changes.AmountXOut.Add(changes.AmountXOut, b.AmountXOut)
		changes.AmountYIn.Add(changes.AmountYIn, b.AmountYIn)
		changes.AmountYOut.Add(changes.AmountYOut, b.AmountYOut)

		totalBinReserveChanges[b.BinID] = changes
	}
	newBins := []Bin{}
	for _, b := range p.bins {
		newBin := Bin{
			ID:       b.ID,
			ReserveX: new(big.Int).Set(b.ReserveX),
			ReserveY: new(big.Int).Set(b.ReserveY),
		}

		changes, ok := totalBinReserveChanges[newBin.ID]
		if ok {
			newBin.ReserveX.Add(new(big.Int).Sub(newBin.ReserveX, changes.AmountXOut), changes.AmountXIn)
			newBin.ReserveY.Add(new(big.Int).Sub(newBin.ReserveY, changes.AmountYOut), changes.AmountYIn)
		}

		if !newBin.isEmpty() {
			newBins = append(newBins, newBin)
		}
	}
	sort.Slice(newBins, func(i, j int) bool {
		return newBins[i].ID < newBins[j].ID
	})
	p.bins = newBins
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
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
func (p *PoolSimulator) getSwapIn(amountOut *big.Int, swapForY bool) (*getSwapInResult, error) {
	var (
		amountsOutLeft     = new(big.Int).Set(amountOut)
		binStep            = p.binStep
		amountIn           = integer.Zero()
		swapFee            = integer.Zero()
		binsReserveChanges []binReserveChanges
	)

	parameters := p.copyParameters()
	id := parameters.ActiveBinID

	parameters = parameters.updateReferences(p.blockTimestamp)

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

			var amountOutOfBin *big.Int
			if binReserves.Cmp(amountsOutLeft) > 0 {
				amountOutOfBin = amountsOutLeft
			} else {
				amountOutOfBin = binReserves
			}

			parameters = parameters.updateVolatilityAccumulator(id)

			var amountInWithoutFee *big.Int

			if swapForY {
				amountInWithoutFee, err = shiftDivRoundUp(amountOutOfBin, scaleOffset, price)
			} else {
				amountInWithoutFee, err = mulShiftRoundUp(amountOutOfBin, price, scaleOffset)
			}
			if err != nil {
				return nil, err
			}

			totalFees := parameters.getTotalFee(binStep)

			feeAmount, err := getFeeAmount(amountInWithoutFee, totalFees)
			if err != nil {
				return nil, err
			}

			amountIn.Add(amountIn, new(big.Int).Add(amountInWithoutFee, feeAmount))
			amountsOutLeft.Sub(amountsOutLeft, amountOutOfBin)

			swapFee.Add(swapFee, feeAmount)

			newBinReserveChanges := newBinReserveChanges(
				id, !swapForY, amountIn, amountOutOfBin,
			)
			binsReserveChanges = append(binsReserveChanges, newBinReserveChanges)
		}

		if amountsOutLeft.Sign() == 0 {
			break
		}

		nextID, err := p.getNextNonEmptyBin(swapForY, id)
		if err != nil {
			return nil, ErrNotFoundBinID
		}

		id = nextID
	}

	parameters.ActiveBinID = id

	ret := getSwapInResult{
		AmountIn:           amountIn,
		Fee:                swapFee,
		BinsReserveChanges: binsReserveChanges,
		Parameters:         parameters,
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
func (p *PoolSimulator) getSwapOut(amountIn *big.Int, swapForY bool) (*getSwapOutResult, error) {
	var (
		amountsInLeft      = new(big.Int).Set(amountIn)
		binStep            = p.binStep
		amountOut          = new(big.Int)
		swapFee            = new(big.Int)
		binsReserveChanges []binReserveChanges
	)

	parameters := p.copyParameters()
	id := parameters.ActiveBinID

	parameters = parameters.updateReferences(p.blockTimestamp)

	for {
		binArrIdx, err := p.findBinArrIndex(id)
		if err != nil {
			return nil, err
		}
		binReserves := p.bins[binArrIdx]
		if !binReserves.isEmptyForSwap(!swapForY) {
			parameters = parameters.updateVolatilityAccumulator(id)

			amountsInWithFees, amountsOutOfBin, totalFees, err := binReserves.getAmounts(
				parameters, binStep, swapForY, id, amountsInLeft,
			)
			if err != nil {
				return nil, err
			}

			if amountsInWithFees.Sign() > 0 {
				amountsInLeft.Sub(amountsInLeft, amountsInWithFees)
				amountOut.Add(amountOut, amountsOutOfBin)
				swapFee.Add(swapFee, totalFees)

				pFee, err := scalarMulDivBasisPointRoundDown(
					totalFees,
					big.NewInt(int64(p.staticFeeParams.ProtocolShare)),
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

		if amountsInLeft.Sign() == 0 {
			break
		}

		nextID, err := p.getNextNonEmptyBin(swapForY, id)
		if err != nil {
			return nil, ErrNotFoundBinID
		}

		id = nextID
	}

	parameters.ActiveBinID = id

	ret := getSwapOutResult{
		AmountOut:          amountOut,
		Fee:                swapFee,
		BinsReserveChanges: binsReserveChanges,
		Parameters:         parameters,
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
