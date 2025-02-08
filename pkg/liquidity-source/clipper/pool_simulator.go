package clipper

import (
	"math"
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	extra Extra

	addressToToken map[string]PoolAsset
	symbolToToken  map[string]PoolAsset
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

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

	// We used to use Closed Formed Solution, suggested by Clipper.
	// However, it's not accurate, leading Build route often fails.
	// So we switch to the Accurate Calculation.

	inX, _ := params.TokenAmountIn.Amount.Float64()
	inX /= math.Pow10(int(assetIn.Decimals))

	pX := assetIn.PriceInUSD

	if inX*pX < 0.1 {
		return nil, ErrMinAmountInNotEnough
	}

	qX, _ := assetIn.Quantity.Float64()
	qX /= math.Pow10(int(assetIn.Decimals))
	wX := float64(assetIn.ListingWeight)

	pY := assetOut.PriceInUSD
	qY, _ := assetOut.Quantity.Float64()
	qY /= math.Pow10(int(assetOut.Decimals))
	wY := float64(assetOut.ListingWeight)

	M := (basisPoint - pairInfo.FeeInBasisPoints) / basisPoint

	k := p.extra.K

	first := math.Pow(pX*qX, 1-k) / math.Pow(wX, k)
	second := math.Pow(pY*qY, 1-k) / math.Pow(wY, k)
	third := math.Pow(pX*(qX+M*inX), 1-k) / math.Pow(wX, k)

	numerator := (first + second - third) * math.Pow(wY, k)
	numerator = math.Pow(numerator, 1/(1-k))

	outY := qY - numerator/pY
	outY *= math.Pow10(int(assetOut.Decimals))
	if math.IsNaN(outY) {
		return nil, ErrAmountOutNaN
	}

	amountOut, _ := big.NewFloat(outY).Int(nil)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  assetOut.Address,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  assetOut.Address,
			Amount: bignumber.ZeroBI,
		},

		Gas: defaultGas,

		SwapInfo: SwapInfo{
			ChainID:           p.extra.ChainID,
			TimeInSeconds:     p.extra.TimeInSeconds,
			InputAmount:       params.TokenAmountIn.Amount.String(),
			InputAssetSymbol:  assetIn.Symbol,
			OutputAssetSymbol: assetOut.Symbol,
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} { return nil }
