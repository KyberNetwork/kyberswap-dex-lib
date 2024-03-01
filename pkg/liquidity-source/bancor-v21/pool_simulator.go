package bancorv21

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/logger"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		poolpkg.Pool
		gas                       Gas
		innerPoolByAnchor         map[string]*entity.Pool
		anchorsByConvertibleToken map[string][]string
		tokensByLpAddress         map[string][]string
		anchorTokenPathFinder     string
	}
)

var (
	ErrInvalidReserve = errors.New("invalid reserve")
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
		gas:                       defaultGas,
		innerPoolByAnchor:         extra.InnerPoolByAnchor,
		anchorsByConvertibleToken: extra.AnchorsByConvertibleToken,
		tokensByLpAddress:         extra.TokensByLpAddress,
		anchorTokenPathFinder:     BancorTokenAddress,
	}, nil
}

// crossReserveTargetAmount calculates the target amount in a cross-reserve operation
// using big integers for arbitrary precision arithmetic.
// reference: https://github.com/bancorprotocol/contracts-solidity/blob/dc378ab9d57d1b4a41dfa95fc5142fac2f4ee307/contracts/converter/types/standard-pool/StandardPoolConverter.sol#L1082
func crossReserveTargetAmount(sourceReserveBalance, targetReserveBalance, sourceAmount *big.Int) (*big.Int, error) {
	// Ensure that both source and target reserve balances are greater than 0
	if sourceReserveBalance.Cmp(utils.ZeroBI) != 1 || targetReserveBalance.Cmp(utils.ZeroBI) != 1 {
		return nil, ErrInvalidReserve
	}

	// Perform the calculation: targetReserveBalance * sourceAmount / (sourceReserveBalance + sourceAmount)
	numerator := new(big.Int).Mul(targetReserveBalance, sourceAmount)
	denominator := new(big.Int).Add(sourceReserveBalance, sourceAmount)
	result := new(big.Int).Div(numerator, denominator)

	return result, nil
}

// getAmountOut calculates the amount of tokenOut to receive for a given amount of tokenIn
// ref: https://github.com/bancorprotocol/contracts-solidity/blob/dc378ab9d57d1b4a41dfa95fc5142fac2f4ee307/contracts/converter/types/standard-pool/StandardPoolConverter.sol#L450
func calculateFee(targetAmount *big.Int, conversionFee *big.Int) *big.Int {
	// Assuming PPM_RESOLUTION is a constant

	// Convert PPM_RESOLUTION to big.Int
	ppmResolution := big.NewInt(PPM_RESOLUTION)

	// Calculate targetAmount * ConversionFee
	numerator := new(big.Int).Mul(targetAmount, conversionFee)

	// Calculate the fee: (targetAmount * ConversionFee) / PPM_RESOLUTION
	fee := new(big.Int).Div(numerator, ppmResolution)

	return fee
}

func targetAmountAndFee(sourceToken, targetToken string, sourceBalance, targetBalance, sourceAmount, conversionFee *big.Int) (*big.Int, *big.Int, error) {
	targetAmount, err := crossReserveTargetAmount(sourceBalance, targetBalance, sourceAmount)
	if err != nil {
		return nil, nil, err
	}
	fee := calculateFee(targetAmount, conversionFee)
	targetAmount = targetAmount.Sub(targetAmount, fee)
	return targetAmount, fee, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)

	path := s.findPath(tokenAmountIn.Token, tokenOut)
	if len(path) == 0 {
		return nil, ErrInvalidPath
	}
	amountOut, err := s.rateByPath(path, tokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}

	swapInfo, err := json.Marshal(path)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: tokenOut, Amount: amountOut},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee:      &poolpkg.TokenAmount{Token: tokenAmountIn.Token, Amount: integer.Zero()},
		Gas:      s.gas.Swap,
		SwapInfo: string(swapInfo),
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	var path []string
	if err := json.Unmarshal([]byte(params.SwapInfo.(string)), &path); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": s.Pool.Info.Address,
		}).Errorf("failed to unmarshal banchor conversionPath")

		return
	}
	for i := 2; i < len(path); i += 2 {
		sourceToken := path[i-2]
		anchor := path[i-1]
		targetToken := path[i]
		p, exist := s.innerPoolByAnchor[anchor]
		if !exist {
			logger.WithFields(logger.Fields{
				"poolAddress": s.Pool.Info.Address,
			}).Errorf("path contains invalid anchor")
			return
		}

		indexIn := -1
		indexOut := -1
		for j, token := range p.Tokens {
			if token.Address == sourceToken {
				indexIn = j
			}
			if token.Address == targetToken {
				indexOut = j
			}
		}
		if indexIn < 0 || indexOut < 0 {
			logger.WithFields(logger.Fields{
				"poolAddress": s.Pool.Info.Address,
			}).Errorf("path contains invalid token")
			return
		}

		oldReserveIn, ok := new(big.Int).SetString(p.Reserves[indexIn], 10)
		if !ok {
			logger.WithFields(logger.Fields{
				"poolAddress": s.Pool.Info.Address,
			}).Errorf("failed to parse reserve")
			return
		}
		oldReserveOut, ok := new(big.Int).SetString(p.Reserves[indexOut], 10)
		if !ok {
			logger.WithFields(logger.Fields{
				"poolAddress": s.Pool.Info.Address,
			}).Errorf("failed to parse reserve")
			return
		}

		p.Reserves[indexIn] = new(big.Int).Add(oldReserveIn, params.TokenAmountIn.Amount).String()
		p.Reserves[indexOut] = new(big.Int).Sub(oldReserveOut, params.TokenAmountOut.Amount).String()
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMetaInner{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

// Simulates the _getExtendedArray Solidity function
func getExtendedArray(item0, item1 string, array []string) []string {
	newArray := make([]string, 2+len(array))
	newArray[0] = item0
	newArray[1] = item1
	copy(newArray[2:], array)
	return newArray
}

// getInitialArray creates a new slice containing a single Ethereum address.
func getInitialArray(item string) []string {
	newArray := make([]string, 1)
	newArray[0] = item

	return newArray
}

// getPartialArray extracts the prefix of a given slice of Ethereum addresses.
func getPartialArray(array []string, length uint) []string {
	// Ensure length does not exceed the length of the input slice
	if length > uint(len(array)) {
		length = uint(len(array))
	}

	newArray := make([]string, length)
	for i := 0; i < int(length); i++ {
		newArray[i] = array[i]
	}

	return newArray
}

func (s *PoolSimulator) findPath(sourceToken string, targetToken string) []string {
	sourcePath := s.getPath(sourceToken)
	targetPath := s.getPath(targetToken)
	return getShortestPath(sourcePath, targetPath)
}

func (s *PoolSimulator) getPath(reserveToken string) []string {
	if reserveToken == s.anchorTokenPathFinder {
		return getInitialArray(reserveToken)
	}

	anchors := s.anchorsByConvertibleToken[reserveToken]
	for _, anchor := range anchors {
		tokens := s.tokensByLpAddress[anchor]
		for i := 0; i < len(tokens); i++ {
			connectorToken := tokens[i]
			if connectorToken != reserveToken {
				path := s.getPath(connectorToken)
				if len(path) > 0 {
					return getExtendedArray(reserveToken, anchor, path)
				}
			}
		}
	}

	return []string{}
}

// getShortestPath merges two paths with a common suffix into one, while avoiding the copy function for specific parts as requested.
func getShortestPath(sourcePath []string, targetPath []string) []string {
	if len(sourcePath) > 0 && len(targetPath) > 0 {
		i := len(sourcePath)
		j := len(targetPath)
		for i > 0 && j > 0 && sourcePath[i-1] == targetPath[j-1] {
			i--
			j--
		}

		path := make([]string, i+j+1)
		for m := 0; m <= i; m++ {
			path[m] = sourcePath[m]
		}
		for n := j; n > 0; n-- {
			path[len(path)-n] = targetPath[n-1]
		}

		length := 0
		for p := 0; p < len(path); p++ {
			for q := p + 2; q < len(path)-(p%2); q += 2 {
				if path[p] == path[q] {
					p = q
				}
			}
			path[length] = path[p]
			length++
		}

		return getPartialArray(path, uint(length))
	}

	return []string{}
}

func (s *PoolSimulator) getReturn(anchor, sourceToken, targetToken string, sourceAmount *big.Int) (*big.Int, *big.Int, error) {
	var (
		ok                           bool
		sourceReserve, targetReserve *big.Int
	)

	p, exist := s.innerPoolByAnchor[anchor]
	if !exist {
		return nil, nil, ErrInvalidAnchor
	}
	tokenIndexFrom := -1

	for i, token := range p.Tokens {
		if token.Address == sourceToken {
			tokenIndexFrom = i
			sourceReserve, ok = new(big.Int).SetString(p.Reserves[i], 10)
			if !ok {
				return nil, nil, ErrInvalidReserve
			}
			break
		}
	}
	if tokenIndexFrom < 0 {
		return nil, nil, ErrInvalidToken
	}

	tokenIndexTo := -1
	for i, token := range p.Tokens {
		if token.Address == targetToken {
			tokenIndexTo = i
			targetReserve, ok = new(big.Int).SetString(p.Reserves[i], 10)
			if !ok {
				return nil, nil, ErrInvalidReserve
			}
			break
		}
	}
	if tokenIndexTo < 0 {
		return nil, nil, ErrInvalidToken
	}

	conversionFee := new(big.Int).SetInt64(int64(p.SwapFee))

	amountOut, fee, err := targetAmountAndFee(sourceToken, targetToken, sourceReserve, targetReserve, sourceAmount, conversionFee)
	if err != nil {
		return nil, nil, err
	}

	return amountOut, fee, nil
}

func (s *PoolSimulator) rateByPath(path []string, sourceAmount *big.Int) (*big.Int, error) {
	var err error
	// Verify that the number of elements is larger than 2 and odd.
	if len(path) <= 2 || len(path)%2 != 1 {
		return nil, fmt.Errorf("ERR_INVALID_PATH")
	}

	amount := new(big.Int).Set(sourceAmount)

	// Iterate over the conversion path.
	for i := 2; i < len(path); i += 2 {
		sourceToken := path[i-2]
		anchor := path[i-1]
		targetToken := path[i]
		amount, _, err = s.getReturn(anchor, sourceToken, targetToken, amount)
		if err != nil {
			return nil, err
		}
	}

	return amount, nil
}
