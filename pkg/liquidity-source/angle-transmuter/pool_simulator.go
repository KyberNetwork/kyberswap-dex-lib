package angletransmuter

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		pool.Pool
		Decimals   []uint8
		Transmuter TransmuterState
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
		Decimals:   lo.Map(p.Tokens, func(e *entity.PoolToken, _ int) uint8 { return e.Decimals }),
		Transmuter: extra.Transmuter,
	}, nil
}

// https://github.com/AngleProtocol/angle-transmuter/blob/6e1f2eb1f961d6c3b1cdaefe068d967c33c41936/contracts/transmuter/facets/Swapper.sol#L177
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenOut
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	if amountIn.Sign() <= 0 {
		return nil, ErrInvalidSwap
	}

	isMint := indexOut == len(s.Info.Tokens)-1
	collateral := lo.Ternary(isMint, tokenIn, tokenOut)

	collatInfo := s.Transmuter.Collaterals[collateral]
	otherStablecoinIssued := new(uint256.Int).Sub(s.Transmuter.TotalStablecoinIssued, collatInfo.StablecoinsIssued)

	var amountOut *uint256.Int
	if isMint {
		if !collatInfo.IsMintLive {
			return nil, ErrMintPaused
		}

		oracleValue, err := s._readMint(collateral)
		if err != nil {
			return nil, err
		}

		amountOut, err = _quoteMintExactInput(oracleValue, amountIn, collatInfo.Fees, collatInfo.StablecoinsIssued,
			otherStablecoinIssued, collatInfo.StablecoinCap, s.Decimals[indexIn])
		if err != nil {
			return nil, err
		}

		if err = s.checkHardCaps(&collatInfo, amountOut); err != nil {
			return nil, err
		}
	} else {
		if !collatInfo.IsBurnLive {
			return nil, ErrBurnPaused
		}

		oracleValue, minRatio, err := s._getBurnOracle(collateral)
		if err != nil {
			return nil, err
		}

		amountOut, err = _quoteBurnExactInput(oracleValue, minRatio, amountIn, collatInfo.Fees,
			collatInfo.StablecoinsIssued, otherStablecoinIssued, s.Decimals[indexOut])
		if err != nil {
			return nil, err
		}

		if err = s.checkAmounts(&collatInfo, amountOut); err != nil {
			return nil, err
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: big.NewInt(0),
		},
		Gas: lo.Ternary(isMint, defaultMintGas, defaultBurnGas),
	}, nil
}

func (s *PoolSimulator) checkAmounts(collatInfo *CollateralState, amountOut *uint256.Int) error {
	if collatInfo.IsManaged {
		return ErrUnsupportedBurnCollateral
	}

	if amountOut.Gt(collatInfo.Balance) {
		return ErrInsufficientBalance
	}

	return nil
}

func (s *PoolSimulator) checkHardCaps(collatInfo *CollateralState, amountOut *uint256.Int) error {
	if !shouldCheckHardCaps(s.GetExchange()) {
		return nil
	}

	if collatInfo.StablecoinCap != nil &&
		new(uint256.Int).Add(amountOut, collatInfo.StablecoinsFromCollateral).Gt(collatInfo.StablecoinCap) {
		return ErrInvalidSwap
	}

	return nil
}

func (s *PoolSimulator) _getBurnOracle(collateral string) (*uint256.Int, *uint256.Int, error) {
	var oracleValue *uint256.Int
	minRatio := newBASE18()
	for collat := range s.Transmuter.Collaterals {
		value, ratio, err := s._readBurn(collat)
		if err != nil {
			return nil, nil, err
		}
		if strings.EqualFold(collat, collateral) {
			oracleValue = value
		}
		if ratio.Lt(minRatio) {
			minRatio = ratio
		}
	}
	return oracleValue, minRatio, nil
}

func (s *PoolSimulator) _readMint(collateral string) (*uint256.Int, error) {
	configOracle := s.Transmuter.Collaterals[collateral].Config
	if configOracle.OracleType == EXTERNAL {
		return newBASE18(), nil
	}
	spot, target, err := s._readSpotAndTarget(collateral)
	if err != nil {
		return nil, err
	}

	if target.Lt(spot) {
		spot = target
	}

	return spot, nil
}

func (s *PoolSimulator) _readBurn(collateral string) (*uint256.Int, *uint256.Int, error) {
	configOracle := s.Transmuter.Collaterals[collateral].Config
	if configOracle.OracleType == EXTERNAL {
		return newBASE18(), newBASE18(), nil
	}

	spot, target, err := s._readSpotAndTarget(collateral)
	if err != nil {
		return nil, nil, err
	}

	ratio, uB := newBASE18(), newBASE18()
	uB.MulDivOverflow(target, uB.Sub(uB, configOracle.Hyperparameters.BurnRatioDeviation), BASE_18)
	if spot.Lt(uB) {
		ratio.MulDivOverflow(ratio, spot, target)
	} else if spot.Lt(target) {
		spot = target
	}
	return spot, ratio, nil
}

func (s *PoolSimulator) _readSpotAndTarget(collateral string) (*uint256.Int, *uint256.Int, error) {
	configOracle := s.Transmuter.Collaterals[collateral].Config
	targetPrice, err := s._read(configOracle.TargetType, configOracle.TargetFeed, newBASE18())
	if err != nil {
		return nil, nil, err
	}

	oracleValue, err := s._read(configOracle.OracleType, configOracle.OracleFeed, new(uint256.Int).Set(targetPrice))
	if err != nil {
		return nil, nil, err
	}
	lB, uB := new(uint256.Int), new(uint256.Int)
	lB.MulDivOverflow(targetPrice, lB.Sub(BASE_18, configOracle.Hyperparameters.UserDeviation), BASE_18)
	uB.MulDivOverflow(targetPrice, uB.Add(BASE_18, configOracle.Hyperparameters.UserDeviation), BASE_18)
	if lB.Lt(oracleValue) && !uB.Lt(oracleValue) {
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
				price.MulDivOverflow(price, oracleFeed.Chainlink.Answers[i], u256.TenPow(oracleFeed.Chainlink.ChainlinkDecimals[i]))
			} else {
				// (_quoteAmount * (10 ** decimals)) / uint256(ratio);
				price.MulDivOverflow(price, u256.TenPow(oracleFeed.Chainlink.ChainlinkDecimals[i]), oracleFeed.Chainlink.Answers[i])
			}
		}
		return price, nil
	case STABLE:
		return newBASE18(), nil
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
				price.MulDivOverflow(price, normalizedPrice, normalizer)
			} else if oracleFeed.Pyth.IsMultiplied[i] == 1 && !isNormalizerExpoNeg {
				price.Mul(price.Mul(price, normalizedPrice), normalizer)
			} else if oracleFeed.Pyth.IsMultiplied[i] == 0 && isNormalizerExpoNeg {
				price.MulDivOverflow(price, normalizer, normalizedPrice)
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
		return newBASE18()
	}
	return baseValue
}

func (s *PoolSimulator) CanSwapFrom(address string) []string { return s.CanSwapTo(address) }

func (s *PoolSimulator) CanSwapTo(address string) []string {
	tokenIndex := s.GetTokenIndex(address)
	if tokenIndex < 0 || len(s.Info.Tokens) < 2 {
		return nil
	}
	if tokenIndex == len(s.Info.Tokens)-1 { // agToken
		return s.Info.Tokens[:len(s.Info.Tokens)-1]
	}
	return []string{s.Info.Tokens[len(s.Info.Tokens)-1]}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenAmountOut.Token
	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)
	amountOut := uint256.MustFromBig(params.TokenAmountOut.Amount)

	isMint := tokenOut == s.Info.Tokens[len(s.Info.Tokens)-1]
	if !isMint {
		s.Transmuter.Collaterals[tokenOut].StablecoinsIssued.Sub(
			s.Transmuter.Collaterals[tokenOut].StablecoinsIssued,
			amountIn,
		)
		s.Transmuter.TotalStablecoinIssued.Sub(
			s.Transmuter.TotalStablecoinIssued,
			amountIn,
		)
		s.Transmuter.Collaterals[tokenOut].Balance.Sub(
			s.Transmuter.Collaterals[tokenOut].Balance,
			amountOut,
		)
	} else {
		s.Transmuter.Collaterals[tokenIn].StablecoinsIssued.Add(
			s.Transmuter.Collaterals[tokenIn].StablecoinsIssued,
			amountOut,
		)
		s.Transmuter.TotalStablecoinIssued.Add(
			s.Transmuter.TotalStablecoinIssued,
			amountOut,
		)
		s.Transmuter.Collaterals[tokenIn].Balance.Add(
			s.Transmuter.Collaterals[tokenIn].Balance,
			amountIn,
		)

		if shouldCheckHardCaps(s.GetExchange()) {
			s.Transmuter.Collaterals[tokenIn].StablecoinsFromCollateral.Add(
				s.Transmuter.Collaterals[tokenIn].StablecoinsFromCollateral,
				amountOut,
			)
		}
	}
}
