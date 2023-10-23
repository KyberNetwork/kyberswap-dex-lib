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
) (pool.TokenAmount, error) {
	err := p.validateTokens([]string{tokenAmountIn.Token, tokenOut})
	if err != nil {
		return pool.TokenAmount{}, err
	}
	swapForY := tokenAmountIn.Token == p.Info.Tokens[0]

	amountsInLeft := tokenAmountIn.Amount
	binStep := p.binStep

	parameters := p.newParameters()
	id := parameters.ActiveBinID
	parameters = parameters.updateReferences(p.blockTimestamp)

	for {
		bin := p.bins[id]
		if !bin.isEmpty(!swapForY) {
			parameters = parameters.updateVolatilityAccumulator(id)

			// (bytes32 amountsInWithFees, bytes32 amountsOutOfBin, bytes32 totalFees) =
			// binReserves.getAmounts(parameters, binStep, swapForY, id, amountsInLeft);

		}
	}

	return pool.TokenAmount{}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {

}

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

func (p *PoolSimulator) newParameters() *parameters {
	return &parameters{
		StaticFeeParams:   p.staticFeeParams,
		VariableFeeParams: p.variableFeeParams,
		ActiveBinID:       p.activeBinID,
	}
}
