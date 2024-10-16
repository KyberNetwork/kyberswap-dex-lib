package clipper

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/samber/lo"
)

var (
	ErrInvalidTokenIn  = errors.New("invalid token in")
	ErrInvalidTokenOut = errors.New("invalid token out")
	ErrInvalidPair     = errors.New("invalid pair")

	basisPoint float64 = 10000
)

type PoolSimulator struct {
	pool.Pool
	extra Extra

	addressToToken map[string]PoolAsset
	symbolToToken  map[string]PoolAsset
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens: lo.Map(entityPool.Tokens,
					func(item *entity.PoolToken, index int) string { return item.Address }),
				Reserves: lo.Map(entityPool.Reserves,
					func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			},
		},
		extra: extra,
		addressToToken: lo.SliceToMap(extra.Assets,
			func(item PoolAsset) (string, PoolAsset) { return item.Address, item }),
		symbolToToken: lo.SliceToMap(extra.Assets,
			func(item PoolAsset) (string, PoolAsset) { return item.Symbol, item }),
	}, nil
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	tokenIndex := p.GetTokenIndex(address)
	if tokenIndex == -1 {
		return nil
	}

	token, ok := p.addressToToken[address]
	if !ok {
		return nil
	}

	swapToTokens := make([]string, 0, len(p.Pool.Info.Tokens)-1)

	for _, pairs := range p.extra.Pairs {
		var swapToSymbol string

		if pairs.Assets[0] == token.Symbol {
			swapToSymbol = pairs.Assets[1]
		}
		if pairs.Assets[1] == token.Symbol {
			swapToSymbol = pairs.Assets[0]
		}

		if swapToSymbol == "" {
			continue
		}

		swapToToken, ok := p.symbolToToken[swapToSymbol]
		if !ok {
			continue
		}

		swapToTokens = append(swapToTokens, swapToToken.Address)
	}

	return swapToTokens
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	// Find the pair that has the same token as the input & output token
	assetIn, ok := p.addressToToken[params.TokenAmountIn.Token]
	if !ok {
		return nil, ErrInvalidTokenIn
	}

	assetOut, ok := p.addressToToken[params.TokenOut]
	if !ok {
		return nil, ErrInvalidTokenOut
	}

	var pairInfo PoolPair
	for _, pair := range p.extra.Pairs {
		if (pair.Assets[0] == assetIn.Symbol && pair.Assets[1] == assetOut.Symbol) ||
			(pair.Assets[0] == assetOut.Symbol && pair.Assets[1] == assetIn.Symbol) {
			pairInfo = pair
			break
		}
	}

	if pairInfo == (PoolPair{}) {
		return nil, ErrInvalidPair
	}

	// We follow the recommend Closed Formed Solution, suggested by Clipper
	inX := new(big.Float).Quo(
		new(big.Float).SetInt(params.TokenAmountIn.Amount),
		bignumber.TenPowDecimals(assetIn.Decimals),
	)
	pX := new(big.Float).SetFloat64(assetIn.PriceInUSD)
	pY := new(big.Float).SetFloat64(assetOut.PriceInUSD)
	M := new(big.Float).SetFloat64((basisPoint - pairInfo.FeeInBasisPoints) / basisPoint)

	outY := new(big.Float).Quo(
		new(big.Float).Mul(
			new(big.Float).Mul(M, inX),
			pX,
		),
		pY,
	)
	outY.Mul(outY, bignumber.TenPowDecimals(assetOut.Decimals))

	amountOut, _ := outY.Int(nil)

	// TODO: Apply the final check FMV

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  assetOut.Address,
			Amount: amountOut,
		},

		// TODO: Define gas

		SwapInfo: SwapInfo{
			ChainID:           p.extra.ChainID,
			TimeInSeconds:     p.extra.TimeInSeconds,
			InputAmount:       params.TokenAmountIn.Amount,
			InputAssetSymbol:  assetIn.Symbol,
			OutputAssetSymbol: assetOut.Symbol,
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {}
