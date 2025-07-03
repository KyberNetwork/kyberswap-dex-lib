package angletransmuter

import (
	"encoding/json"
	"math/big"
	"strings"

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
		Tokens         []*entity.PoolToken
		StableToken    common.Address
		StableDecimals uint8

		Transmuter TransmuterState
		gas        Gas
	}
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

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
		Tokens:     p.Tokens,
		gas:        extra.Gas,
		Transmuter: extra.Transmuter,
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

	isMint := indexOut == len(s.Pool.Info.Tokens)-1
	var oracleValue *uint256.Int
	minRatio := uint256.NewInt(0)
	var err error
	collateral := strings.ToLower(tokenAmountIn.Token)
	if isMint {
		oracleValue, err = s._readMint(collateral)
		if err != nil {
			return nil, err
		}
	} else {
		collateral = strings.ToLower(tokenOut)
		oracleValue, minRatio, err = s._getBurnOracle(collateral)
		if err != nil {
			return nil, err
		}
	}

	collatStablecoinIssued := s.Transmuter.Collaterals[collateral].StablecoinsIssued
	otherStablecoinIssued := new(uint256.Int).Sub(s.Transmuter.TotalStablecoinIssued, collatStablecoinIssued)
	fees := s.Transmuter.Collaterals[collateral].Fees
	stablecoinCap := s.Transmuter.Collaterals[collateral].StablecoinCap
	var amountOut *uint256.Int
	if isMint {
		amountOut, err = _quoteMintExactInput(oracleValue, amountIn, fees, collatStablecoinIssued, otherStablecoinIssued, stablecoinCap, s.Tokens[indexIn].Decimals)
	} else {
		amountOut, err = _quoteBurnExactInput(oracleValue, minRatio, amountIn, fees, collatStablecoinIssued, otherStablecoinIssued, s.Tokens[indexOut].Decimals)
	}
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: big.NewInt(0),
		},
		Gas: int64(lo.Ternary(isMint, s.gas.Mint, s.gas.Burn)),
	}, nil
}

func (s *PoolSimulator) _getBurnOracle(collateral string) (*uint256.Int, *uint256.Int, error) {
	var oracleValue *uint256.Int
	minRatio := newBASE_18()
	for collat := range s.Transmuter.Collaterals {
		value, ratio, err := s._readBurn(collat)
		if err != nil {
			return nil, nil, err
		}
		if strings.EqualFold(collat, collateral) {
			oracleValue = value
		}
		if ratio.Cmp(minRatio) < 0 {
			minRatio = ratio
		}
	}
	return oracleValue, minRatio, nil
}

func (s *PoolSimulator) _readMint(collateral string) (*uint256.Int, error) {
	configOracle := s.Transmuter.Collaterals[collateral].Config
	if configOracle.OracleType == EXTERNAL {
		return newBASE_18(), nil
	}
	spot, target, err := s._readSpotAndTarget(collateral)
	if err != nil {
		return nil, err
	}

	if target.Cmp(spot) < 0 {
		spot = target
	}

	return spot, nil
}

func (s *PoolSimulator) _readBurn(collateral string) (*uint256.Int, *uint256.Int, error) {
	configOracle := s.Transmuter.Collaterals[collateral].Config
	if configOracle.OracleType == EXTERNAL {
		return newBASE_18(), newBASE_18(), nil
	}

	spot, target, err := s._readSpotAndTarget(collateral)
	if err != nil {
		return nil, nil, err
	}

	ratio, uB := newBASE_18(), newBASE_18()
	uB.Mul(target, uB.Sub(uB, configOracle.Hyperparameters.BurnRatioDeviation)).Div(uB, BASE_18)
	if spot.Cmp(uB) < 0 {
		ratio.Div(ratio.Mul(ratio, spot), target)
	} else if spot.Cmp(target) < 0 {
		spot = target
	}
	return spot, ratio, nil
}

func (s *PoolSimulator) _readSpotAndTarget(collateral string) (*uint256.Int, *uint256.Int, error) {
	configOracle := s.Transmuter.Collaterals[collateral].Config
	targetPrice, err := s._read(configOracle.TargetType, configOracle.TargetFeed, newBASE_18())
	if err != nil {
		return nil, nil, err
	}

	oracleValue, err := s._read(configOracle.OracleType, configOracle.OracleFeed, new(uint256.Int).Set(targetPrice))
	if err != nil {
		return nil, nil, err
	}
	lB, uB := new(uint256.Int), new(uint256.Int)
	lB.Mul(targetPrice, lB.Sub(BASE_18, configOracle.Hyperparameters.UserDeviation)).Div(lB, BASE_18)
	uB.Mul(targetPrice, uB.Add(BASE_18, configOracle.Hyperparameters.UserDeviation)).Div(uB, BASE_18)
	if lB.Cmp(oracleValue) < 0 && uB.Cmp(oracleValue) >= 0 {
		oracleValue = targetPrice
	}
	return oracleValue, targetPrice, nil
}

func (s *PoolSimulator) _read(oracleType OracleReadType, oracleFeed OracleFeed, baseValue *uint256.Int) (*uint256.Int, error) {
	switch oracleType {
	case CHAINLINK_FEEDS:
		if !oracleFeed.IsChainLink || !oracleFeed.Chainlink.Active {
			return nil, ErrInvalidOracle
		}
		price := s._quoteAmount(OracleQuoteType(oracleFeed.Chainlink.QuoteType), baseValue)
		for i := range oracleFeed.Chainlink.CircuitChainlink {
			// price, err = s._readChainlink(oracleFeed, address, price)
			// if err != nil {
			// 	return nil, err
			// }
			// TODO: check staled rate
			if oracleFeed.Chainlink.CircuitChainIsMultiplied[i] == 1 {
				// (_quoteAmount * uint256(ratio)) / (10 ** decimals);
				price.Mul(
					price,
					oracleFeed.Chainlink.Answers[i],
				).Div(
					price,
					new(uint256.Int).Exp(U10, uint256.NewInt(uint64(oracleFeed.Chainlink.ChainlinkDecimals[i]))),
				)
			} else {
				// (_quoteAmount * (10 ** decimals)) / uint256(ratio);
				price.Mul(
					price,
					new(uint256.Int).Exp(U10, uint256.NewInt(uint64(oracleFeed.Chainlink.ChainlinkDecimals[i]))),
				).Div(
					price,
					oracleFeed.Chainlink.Answers[i],
				)
			}
		}
		return price, nil
	case STABLE:
		return newBASE_18(), nil
	case NO_ORACLE:
		return baseValue, nil
	case WSTETH:
		return nil, ErrUnimplemented
	case CBETH:
		return nil, ErrUnimplemented
	case RETH:
		return nil, ErrUnimplemented
	case SFRXETH:
		return nil, ErrUnimplemented
	case PYTH:
		if !oracleFeed.IsPyth || !oracleFeed.Pyth.Active {
			return nil, ErrInvalidOracle
		}
		price := s._quoteAmount(OracleQuoteType(oracleFeed.Pyth.QuoteType), baseValue)
		for i := range oracleFeed.Pyth.FeedIds {
			normalizedPrice := oracleFeed.Pyth.PythState[i].Price
			isNormalizerExpoNeg := oracleFeed.Pyth.PythState[i].Expo.Sign() < 0
			normalizer := new(uint256.Int).Exp(U10, new(uint256.Int).Abs(oracleFeed.Pyth.PythState[i].Expo))

			if oracleFeed.Pyth.IsMultiplied[i] == 1 && isNormalizerExpoNeg {
				price.Div(price.Mul(price, normalizedPrice), normalizer)
			} else if oracleFeed.Pyth.IsMultiplied[i] == 1 && !isNormalizerExpoNeg {
				price.Mul(price.Mul(price, normalizedPrice), normalizer)
			} else if oracleFeed.Pyth.IsMultiplied[i] == 0 && isNormalizerExpoNeg {
				price.Div(price.Mul(price, normalizer), normalizedPrice)
			} else {
				price.Div(price, new(uint256.Int).Mul(normalizedPrice, normalizer))
			}
		}
		return price, nil
	case MAX:
		return oracleFeed.Max, nil
	case MORPHO_ORACLE:
		if !oracleFeed.IsMorpho || !oracleFeed.Morpho.Active {
			return nil, ErrInvalidOracle
		}
		return new(uint256.Int).Div(oracleFeed.Morpho.Price, oracleFeed.Morpho.NormalizationFactor), nil
	default:
		return baseValue, nil
	}
}

func (s *PoolSimulator) _quoteAmount(quoteType OracleQuoteType, baseValue *uint256.Int) *uint256.Int {
	if quoteType == UNIT {
		return newBASE_18()
	}
	return baseValue
}

func (p *PoolSimulator) CanSwapFrom(address string) []string { return p.CanSwapTo(address) }

func (p *PoolSimulator) CanSwapTo(address string) []string {
	if !lo.Contains(p.Info.Tokens, address) || len(p.Info.Tokens) < 2 {
		return nil
	}
	if lo.IndexOf(p.Info.Tokens, address) == len(p.Info.Tokens)-1 { // agToken
		return lo.Subset(p.Info.Tokens, 0, uint(len(p.Info.Tokens)-1))
	}
	return []string{p.Info.Tokens[len(p.Info.Tokens)-1]}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	return nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if strings.EqualFold(params.TokenAmountIn.Token, p.Info.Tokens[len(p.Info.Tokens)-1]) { // agToken
		p.Transmuter.Collaterals[params.TokenAmountOut.Token].StablecoinsIssued.Sub(
			p.Transmuter.Collaterals[params.TokenAmountOut.Token].StablecoinsIssued,
			uint256.MustFromBig(params.TokenAmountIn.Amount),
		)
		p.Transmuter.TotalStablecoinIssued.Sub(
			p.Transmuter.TotalStablecoinIssued,
			uint256.MustFromBig(params.TokenAmountIn.Amount),
		)
	} else {
		p.Transmuter.Collaterals[params.TokenAmountIn.Token].StablecoinsIssued.Add(
			p.Transmuter.Collaterals[params.TokenAmountIn.Token].StablecoinsIssued,
			uint256.MustFromBig(params.TokenAmountOut.Amount),
		)
		p.Transmuter.TotalStablecoinIssued.Add(
			p.Transmuter.TotalStablecoinIssued,
			uint256.MustFromBig(params.TokenAmountOut.Amount),
		)
	}
}
