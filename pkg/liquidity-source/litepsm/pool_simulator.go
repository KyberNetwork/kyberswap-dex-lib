package litepsm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	*Extra
	*StaticExtra
	DaiBal, GemBal       *uint256.Int
	TokenDecimalDiff     int8
	To18ConversionFactor *uint256.Int
}

var _ = pool.RegisterFactory0(DexTypeLitePSM, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	tokenDecimalDiff := int8(entityPool.Tokens[0].Decimals) - int8(entityPool.Tokens[1].Decimals)
	to18ConversionFactor := big256.TenPow(uint64(tokenDecimalDiff))

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
		Extra:                &extra,
		StaticExtra:          &staticExtra,
		DaiBal:               big256.New(entityPool.Reserves[0]),
		GemBal:               big256.New(entityPool.Reserves[1]),
		TokenDecimalDiff:     tokenDecimalDiff,
		To18ConversionFactor: to18ConversionFactor,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	gem, calcFunc, gas := tokenAmountIn.Token, p.sellGem, int64(gasSell)
	if p.IsGem(tokenOut) {
		gem, calcFunc, gas = tokenOut, p.buyGem, gasBuy
	}
	daiAmt, fee, err := calcFunc(amountIn)
	if err != nil {
		return nil, err
	}
	if p.GemJoin != nil {
		gas += gasJoin
	}
	if p.Dai != nil {
		gas += gasWrap
	}
	if p.IsMint {
		gas += gasMint
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: daiAmt.ToBig()},
		Fee:            &pool.TokenAmount{Token: gem, Amount: fee.ToBig()},
		Gas:            gas,
	}, nil
}

func (p *PoolSimulator) CloneBalance() pool.IPoolSimulator {
	cloned := *p
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	inputAmount, outputAmount := big256.FromBig(input.Amount), big256.FromBig(output.Amount)

	if p.IsGem(output.Token) {
		p.updateBalanceBuyingGem(inputAmount, outputAmount)
		return
	}

	p.updateBalanceSellingGem(inputAmount, outputAmount)
}

func (p *PoolSimulator) GetMetaInfo(_, tokenOut string) any {
	isBuyGem := p.IsGem(tokenOut)
	var approvalAddress string
	if isBuyGem || p.GemJoin == nil || p.Dai != nil { // p.Dai != nil means PSM wrapper
		approvalAddress = p.GetAddress()
	} else {
		approvalAddress = hexutil.Encode(p.GemJoin[:])
	}
	return MetaInfo{
		IsBuyGem:         isBuyGem,
		TokenDecimalDiff: p.TokenDecimalDiff,
		PrecisionDecimal: Precision,
		ApprovalAddress:  approvalAddress,
	}
}

func (p *PoolSimulator) IsGem(token string) bool {
	return p.GetTokenIndex(token) == 1
}
