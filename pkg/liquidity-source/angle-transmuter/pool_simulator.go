package angletransmuter

import (
	"encoding/json"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		pool.Pool
		StableToken    common.Address
		StableDecimals uint8

		Transmuter TransmuterState
		OraclePyth map[string]PythState
		gas        Gas
	}
)

// var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	tokens := lo.Map(p.Tokens, func(e *entity.PoolToken, _ int) string { return e.Address })
	reserves := lo.Map(p.Reserves, func(e string, _ int) *big.Int { return bignum.NewBig(e) })

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: p.BlockNumber,
		}},
		gas: extra.Gas,
	}, nil
}

// https://github.com/AngleProtocol/angle-transmuter/blob/6e1f2eb1f961d6c3b1cdaefe068d967c33c41936/contracts/transmuter/facets/Swapper.sol#L177
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = params.TokenAmountIn
		tokenOut      = params.TokenOut
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

	return nil, nil

}
