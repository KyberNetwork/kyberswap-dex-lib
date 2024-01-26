package uniswapv2

import (
	"errors"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidReserve          = errors.New("invalid reserve")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientLiquidity   = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrInvalidK                = errors.New("K")
)

type (
	PoolSimulator struct {
		poolpkg.Pool
		fee          *uint256.Int
		feePrecision *uint256.Int

		gas Gas
	}

	Gas struct {
		Swap int64
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		fee:          uint256.NewInt(extra.Fee),
		feePrecision: uint256.NewInt(extra.FeePrecision),
		gas:          defaultGas,
	}, nil
}

func NewPoolSimulatorV2(entityPool entity.Pool) (*PoolSimulator, error) {
	sim := &PoolSimulator{}
	err := InitPoolSimulator(entityPool, sim)
	if err != nil {
		return nil, err
	}
	return sim, nil
}

const NUM_TOKEN = 2

func InitPoolSimulator(entityPool entity.Pool, sim *PoolSimulator) error {
	if len(entityPool.Tokens) != NUM_TOKEN || len(entityPool.Reserves) != NUM_TOKEN {
		return errors.New("Invalid number of token")
	}
	// in case the caller mess thing up
	if sim == nil {
		return errors.New("Invalid simulator instance")
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return err
	}

	sim.Pool.Info.Address = entityPool.Address
	sim.Pool.Info.ReserveUsd = entityPool.ReserveUsd
	sim.Pool.Info.Exchange = entityPool.Exchange
	sim.Pool.Info.Type = entityPool.Type
	sim.Pool.Info.BlockNumber = entityPool.BlockNumber
	sim.gas = defaultGas

	// try to re-use existing array if possible, if not then allocate new one
	if cap(sim.Pool.Info.Tokens) >= NUM_TOKEN {
		sim.Pool.Info.Tokens = sim.Pool.Info.Tokens[:NUM_TOKEN]
	} else {
		sim.Pool.Info.Tokens = make([]string, NUM_TOKEN)
	}
	for i := range entityPool.Tokens {
		sim.Pool.Info.Tokens[i] = entityPool.Tokens[i].Address
	}

	if cap(sim.Pool.Info.Reserves) >= NUM_TOKEN {
		sim.Pool.Info.Reserves = sim.Pool.Info.Reserves[:NUM_TOKEN]
	} else {
		sim.Pool.Info.Reserves = make([]*big.Int, NUM_TOKEN)
	}
	var tmp uint256.Int
	for i := range entityPool.Reserves {
		// still not sure why, but uint256.SetFromDecimal doesn't use `strings.NewReader` like bigInt.SetString
		// so it's cheaper to convert string to uint256 then to bigInt
		// (in the far future if we can replace pool's reserves with uint256 then we can remove the last step)
		err := tmp.SetFromDecimal(entityPool.Reserves[i])
		if err != nil {
			return err
		}
		if sim.Pool.Info.Reserves[i] == nil {
			sim.Pool.Info.Reserves[i] = new(big.Int)
		}
		utils.FillBig(&tmp, sim.Pool.Info.Reserves[i])
	}

	if sim.fee == nil {
		sim.fee = new(uint256.Int)
	}
	sim.fee.SetUint64(extra.Fee)

	if sim.feePrecision == nil {
		sim.feePrecision = new(uint256.Int)
	}
	sim.feePrecision.SetUint64(extra.FeePrecision)

	return nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	if amountIn.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	if reserveIn.Cmp(number.Zero) <= 0 || reserveOut.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountOut := s.getAmountOut(amountIn, reserveIn, reserveOut)
	if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	// NOTE: Intentionally comment out, since kAfter should always smaller than kBefore.
	// balanceIn := new(uint256.Int).Add(reserveIn, amountIn)
	// balanceOut := new(uint256.Int).Sub(reserveOut, amountOut)

	// balanceInAdjusted := new(uint256.Int).Sub(
	// 	new(uint256.Int).Mul(balanceIn, s.feePrecision),
	// 	new(uint256.Int).Mul(amountIn, s.fee),
	// )
	// balanceOutAdjusted := new(uint256.Int).Mul(balanceOut, s.feePrecision)

	// kBefore := new(uint256.Int).Mul(new(uint256.Int).Mul(reserveIn, reserveOut), new(uint256.Int).Mul(s.feePrecision, s.feePrecision))
	// kAfter := new(uint256.Int).Mul(balanceInAdjusted, balanceOutAdjusted)

	// if kAfter.Cmp(kBefore) < 0 {
	// 	return nil, ErrInvalidK
	// }

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas: s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Pool.Info.Reserves[indexIn] = new(big.Int).Add(s.Pool.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Pool.Info.Reserves[indexOut] = new(big.Int).Sub(s.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		Fee:          s.fee.Uint64(),
		FeePrecision: s.feePrecision.Uint64(),
		BlockNumber:  s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) getAmountOut(amountIn, reserveIn, reserveOut *uint256.Int) *uint256.Int {
	amountInWithFee := new(uint256.Int).Mul(amountIn, new(uint256.Int).Sub(s.feePrecision, s.fee))
	numerator := new(uint256.Int).Mul(amountInWithFee, reserveOut)
	denominator := new(uint256.Int).Add(new(uint256.Int).Mul(reserveIn, s.feePrecision), amountInWithFee)

	return new(uint256.Int).Div(numerator, denominator)
}
