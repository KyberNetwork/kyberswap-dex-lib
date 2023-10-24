package liquiditybookv21

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// return param when CalcAmountOut
// update will set the param to the pool

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

	amountsInLeft, amountOut, fee, err := p.getSwapOut(amountIn, swapForY)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: fee,
		},
		Gas: 0, // FIXME: change to real gas
		SwapInfo: SwapInfo{
			AmountsInLeft: amountsInLeft,
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {

}

func (t *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) getSwapOut(amountIn *big.Int, swapForY bool) (*big.Int, *big.Int, *big.Int, error) {
	amountsInLeft := amountIn
	binStep := p.binStep

	parameters := p.copyParameters()
	id := parameters.ActiveBinID
	parameters = parameters.updateReferences(p.blockTimestamp)

	amountOut := big.NewInt(0)
	fee := big.NewInt(0)

	for {
		bin := p.bins[id]
		if !bin.isEmpty(!swapForY) {
			parameters = parameters.updateVolatilityAccumulator(id)

			amountsInWithFees, amountsOutOfBin, totalFees, err := bin.getAmounts(
				parameters, binStep, swapForY, id, amountsInLeft,
			)
			if err != nil {
				return nil, nil, nil, err
			}

			if amountsInWithFees.Cmp(bignumber.ZeroBI) > 0 {
				amountsInLeft = new(big.Int).Sub(amountsInLeft, amountsInWithFees)
				amountOut = new(big.Int).Add(amountOut, amountsOutOfBin)
				fee = new(big.Int).Add(fee, totalFees)
			}
		}

		// TODO: save sub binStep

		if amountsInLeft.Cmp(bignumber.ZeroBI) == 0 {
			break
		}

		nextID, err := p.getNextNonEmptyBin(swapForY, id)
		if err != nil {
			if err == ErrNotFoundBinID {
				break
			}
			return nil, nil, nil, err
		}

		id = nextID
	}

	return amountsInLeft, amountOut, fee, nil
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
	idx, err := p.findBinIndex(id)
	if err != nil {
		return 0, err
	}
	if idx == 0 {
		return 0, ErrNotFoundBinID
	}
	return p.bins[idx-1].ID, nil
}

func (p *PoolSimulator) findFirstLeft(id uint32) (uint32, error) {
	idx, err := p.findBinIndex(id)
	if err != nil {
		return 0, err
	}
	if idx == uint32(len(p.bins)-1) {
		return 0, ErrNotFoundBinID
	}
	return p.bins[idx+1].ID, nil
}

func (p *PoolSimulator) findBinIndex(binID uint32) (uint32, error) {
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
