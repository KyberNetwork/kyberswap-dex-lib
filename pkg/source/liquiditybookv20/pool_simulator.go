package liquiditybookv20

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	blockTimestamp uint64
	feeParams      feeParameters
	activeBinID    uint32
	bins           []bin
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
		Pool:           pool.Pool{Info: info},
		blockTimestamp: extra.RpcBlockTimestamp,
		feeParams:      extra.FeeParameters,
		activeBinID:    extra.ActiveBinID,
		bins:           extra.Bins,
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

	return nil, nil
}

func (p *PoolSimulator) getSwapOut(amountIn *big.Int, swapForY bool) (*getSwapOutResult, error) {
	// All fields are value type, so we can copy directly.
	fp := p.feeParams

	id := p.activeBinID

	fp.updateVariableFeeParameters(p.blockTimestamp, id)
	amountOut := bignumber.ZeroBI

	for {
		binArrIdx, err := p.findBinArrIndex(id)
		if err != nil {
			return nil, err
		}
		bin := p.bins[binArrIdx]
		


	}
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {}

func (t *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) validateTokens(tokens []string) error {
	for _, t := range tokens {
		if p.GetTokenIndex(t) < 0 {
			return ErrInvalidToken
		}
	}
	return nil
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
