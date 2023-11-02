package liquiditybookv21

import (
	"encoding/json"
	"math/big"
	"sort"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/logger"

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
	bins              []bin
}

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

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
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
			p.Info.Reserves[idx] = new(big.Int).Add(reserve, params.TokenAmountIn.Amount)
		}
		if strings.EqualFold(p.Info.Tokens[idx], params.TokenAmountOut.Token) {
			p.Info.Reserves[idx] = new(big.Int).Sub(reserve, params.TokenAmountIn.Amount)
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
				AmountXIn:  integer.Zero(),
				AmountXOut: integer.Zero(),
				AmountYIn:  integer.Zero(),
				AmountYOut: integer.Zero(),
			}
		}
		changes.AmountXIn = new(big.Int).Add(changes.AmountXIn, b.AmountXIn)
		changes.AmountXOut = new(big.Int).Add(changes.AmountXOut, b.AmountXOut)
		changes.AmountYIn = new(big.Int).Add(changes.AmountYIn, b.AmountYIn)
		changes.AmountYOut = new(big.Int).Add(changes.AmountYOut, b.AmountYOut)

		totalBinReserveChanges[b.BinID] = changes
	}
	newBins := []bin{}
	for _, b := range p.bins {
		newBin := bin{
			ID:          b.ID,
			ReserveX:    new(big.Int).Set(b.ReserveX),
			ReserveY:    new(big.Int).Set(b.ReserveY),
			TotalSupply: new(big.Int).Set(b.TotalSupply),
		}

		changes, ok := totalBinReserveChanges[newBin.ID]
		if ok {
			newBin.ReserveX = new(big.Int).Add(new(big.Int).Sub(newBin.ReserveX, changes.AmountXOut), changes.AmountXIn)
			newBin.ReserveY = new(big.Int).Add(new(big.Int).Sub(newBin.ReserveY, changes.AmountYOut), changes.AmountYIn)
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

func (t *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) getSwapOut(amountIn *big.Int, swapForY bool) (*getSwapOutResult, error) {
	var (
		amountsInLeft      = amountIn
		binStep            = p.binStep
		amountOut          = integer.Zero()
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
		bin := p.bins[binArrIdx]
		if !bin.isEmptyForSwap(!swapForY) {
			parameters = parameters.updateVolatilityAccumulator(id)

			amountsInWithFees, amountsOutOfBin, totalFees, err := bin.getAmounts(
				parameters, binStep, swapForY, id, amountsInLeft,
			)
			if err != nil {
				return nil, err
			}

			if amountsInWithFees.Cmp(bignumber.ZeroBI) > 0 {
				amountsInLeft = new(big.Int).Sub(amountsInLeft, amountsInWithFees)
				amountOut = new(big.Int).Add(amountOut, amountsOutOfBin)
				swapFee = new(big.Int).Add(swapFee, totalFees)

				pFee, err := scalarMulDivBasisPointRoundDown(
					totalFees,
					big.NewInt(int64(p.staticFeeParams.ProtocolShare)),
				)
				if err != nil {
					return nil, err
				}
				amountsInWithFees = new(big.Int).Sub(amountsInWithFees, pFee)
				newBinReserveChanges := newBinReserveChanges(
					id, !swapForY, amountsInWithFees, amountsOutOfBin,
				)
				binsReserveChanges = append(binsReserveChanges, newBinReserveChanges)
			}

		}

		if amountsInLeft.Cmp(integer.Zero()) == 0 {
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
