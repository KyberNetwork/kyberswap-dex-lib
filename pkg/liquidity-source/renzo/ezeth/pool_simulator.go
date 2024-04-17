package ezeth

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrInvalidCollateral      = errors.New("invalid collateral")
	ErrInvalidTokenOut        = errors.New("invalid tokenOut")
	ErrMaxTVLReached          = errors.New("max tvl reached")
	ErrMaxTokenTVLReached     = errors.New("max token tvl reached")
	ErrInvalidTokenAmount     = errors.New("invalid tokenAmount")
	ErrOracleNotFound         = errors.New("oracle not found")
	ErrOracleExpired          = errors.New("oracle expired")
	ErrInvalidOraclePrice     = errors.New("invalid oracle price")
	ErrPoolPaused             = errors.New("pool paused")
	ErrStrategyManagerPaused  = errors.New("strategy manager paused")
	ErrRevertNotFound         = errors.New("revert not found")
	ErrRevertInvalidZeroInput = errors.New("revert invalid zero input")
)

var (
	// Scale factor for all values of prices
	SCALE_FACTOR = big.NewInt(1e18)
	// The maxmimum staleness allowed for a price feed from chainlink
	MAX_TIME_WINDOW int64 = 86400 + 60 // 24 hours + 60 seconds
)

type PoolSimulator struct {
	poolpkg.Pool

	paused bool

	// OperatorDelegator https://etherscan.io/address/0x78524beeac12368e600457478738c233f436e9f6
	// StrategyManager https://etherscan.io/address/0x858646372CC42E1A627fcE94aa7A7033e7CF075A
	strategyManagerPaused bool

	collateralTokenIndex map[string]int

	// RestakeManager.calculateTVLs
	operatorDelegatorTokenTVLs [][]*big.Int
	operatorDelegatorTVLs      []*big.Int
	totalTVL                   *big.Int

	// RestakeManager.chooseOperatorDelegatorForDeposit
	operatorDelegatorAllocations []*big.Int

	// OperatorDelegator.tokenStrategyMapping
	tokenStrategyMapping []map[string]bool

	// ezETH.totalSupply
	totalSupply *big.Int

	// RestakeManager.maxDepositTVL
	maxDepositTVL *big.Int

	// renzoOracle.tokenOracleLookup
	tokenOracleLookup map[string]Oracle

	collateralTokenTvlLimits map[string]*big.Int
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
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
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		paused:                       extra.Paused,
		strategyManagerPaused:        extra.StrategyManagerPaused,
		collateralTokenIndex:         extra.CollateralTokenIndex,
		operatorDelegatorTokenTVLs:   extra.OperatorDelegatorTokenTVLs,
		operatorDelegatorTVLs:        extra.OperatorDelegatorTVLs,
		totalTVL:                     extra.TotalTVL,
		operatorDelegatorAllocations: extra.OperatorDelegatorAllocations,
		tokenStrategyMapping:         extra.TokenStrategyMapping,
		totalSupply:                  extra.TotalSupply,
		maxDepositTVL:                extra.MaxDepositTVL,
		tokenOracleLookup:            extra.TokenOracleLookup,
		collateralTokenTvlLimits:     extra.CollateralTokenTvlLimits,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	if s.paused {
		return nil, ErrPoolPaused
	}

	// NOTE: only support deposit and get back ezETH
	if param.TokenOut != s.Info.Tokens[0] {
		return nil, ErrInvalidTokenOut
	}

	var (
		amountOut *big.Int
		err       error
	)

	if param.TokenAmountIn.Token == WETH {
		amountOut, err = s.depositETH(param.TokenAmountIn.Amount)
		if err != nil {
			return nil, err
		}
	} else {
		amountOut, err = s.deposit(param.TokenAmountIn.Token, param.TokenAmountIn.Amount)
		if err != nil {
			return nil, err
		}
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: param.TokenOut, Amount: amountOut},
		Fee:            &poolpkg.TokenAmount{Token: param.TokenOut, Amount: bignumber.ZeroBI},
	}, nil
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	if address != s.Info.Tokens[0] {
		return []string{}
	}

	result := make([]string, 0, len(s.Info.Tokens))
	var tokenIndex = s.GetTokenIndex(address)
	for i := 0; i < len(s.Info.Tokens); i++ {
		if i != tokenIndex {
			result = append(result, s.Info.Tokens[i])
		}
	}

	return result
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	if address == s.Info.Tokens[0] {
		return []string{}
	}

	return []string{s.Info.Tokens[0]}
}

func (s *PoolSimulator) UpdateBalance(param poolpkg.UpdateBalanceParams) {
	s.totalSupply.Add(s.totalSupply, param.TokenAmountOut.Amount)
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return PoolMeta{
		BlockNumber: s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) depositETH(amountIn *big.Int) (*big.Int, error) {
	if s.maxDepositTVL.Cmp(bignumber.ZeroBI) != 0 && new(big.Int).Add(s.totalTVL, amountIn).Cmp(s.maxDepositTVL) > 0 {
		return nil, ErrMaxTVLReached
	}

	return s.calculateMintAmount(s.totalTVL, amountIn, s.totalSupply)
}

func (s *PoolSimulator) deposit(collateralToken string, amount *big.Int) (*big.Int, error) {
	tokenIndex, ok := s.collateralTokenIndex[collateralToken]
	if !ok {
		return nil, ErrInvalidCollateral
	}

	collateralTokenValue, err := s.lookupTokenValue(collateralToken, amount)
	if err != nil {
		return nil, err
	}

	if s.maxDepositTVL.Cmp(bignumber.ZeroBI) != 0 && new(big.Int).Add(s.totalTVL, collateralTokenValue).Cmp(s.maxDepositTVL) > 0 {
		return nil, ErrMaxTVLReached
	}

	if s.collateralTokenTvlLimits[collateralToken].Cmp(bignumber.ZeroBI) != 0 {
		currentTokenTVL := bignumber.ZeroBI

		odLength := len(s.operatorDelegatorTVLs)

		for i := 0; i < odLength; i++ {
			currentTokenTVL = new(big.Int).Add(currentTokenTVL, s.operatorDelegatorTokenTVLs[i][tokenIndex])
		}

		if new(big.Int).Add(currentTokenTVL, collateralTokenValue).Cmp(s.collateralTokenTvlLimits[collateralToken]) > 0 {
			return nil, ErrMaxTokenTVLReached
		}
	}

	if err := s.checkAbleToDeposit(collateralToken, amount); err != nil {
		return nil, err
	}

	return s.calculateMintAmount(
		s.totalTVL,
		collateralTokenValue,
		s.totalSupply,
	)
}

// calculateMintAmount: renzoOracle.calculateMintAmount
func (s *PoolSimulator) calculateMintAmount(
	currentValueInProtocol *big.Int,
	newValueAdded *big.Int,
	existingEzETHSupply *big.Int,
) (*big.Int, error) {
	if currentValueInProtocol.Cmp(bignumber.ZeroBI) == 0 || existingEzETHSupply.Cmp(bignumber.ZeroBI) == 0 {
		return newValueAdded, nil
	}

	inflationPercentage := new(big.Int).Div(
		new(big.Int).Mul(SCALE_FACTOR, newValueAdded),
		new(big.Int).Add(currentValueInProtocol, newValueAdded),
	)

	newEzETHSupply := new(big.Int).Div(
		new(big.Int).Mul(existingEzETHSupply, SCALE_FACTOR),
		new(big.Int).Sub(SCALE_FACTOR, inflationPercentage),
	)

	mintAmount := new(big.Int).Sub(newEzETHSupply, existingEzETHSupply)

	if mintAmount.Cmp(bignumber.ZeroBI) == 0 {
		return nil, ErrInvalidTokenAmount
	}

	return mintAmount, nil
}

// lookupTokenValue: renzoOracle.lookupTokenValue
func (s *PoolSimulator) lookupTokenValue(
	token string,
	value *big.Int,
) (*big.Int, error) {
	oracle, ok := s.tokenOracleLookup[token]
	if !ok {
		return nil, ErrOracleNotFound
	}

	price, timestamp := oracle.LatestRoundData()

	if timestamp.Int64() < time.Now().Unix()-MAX_TIME_WINDOW {
		return nil, ErrOracleExpired
	}

	if price.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrInvalidOraclePrice
	}

	return new(big.Int).Div(new(big.Int).Mul(value, SCALE_FACTOR), price), nil
}

func (s *PoolSimulator) checkAbleToDeposit(collateralToken string, amount *big.Int) error {
	if s.strategyManagerPaused {
		return ErrStrategyManagerPaused
	}

	operatorDelegatorIndex, err := s.chooseOperatorDelegatorForDeposit()
	if err != nil {
		return err
	}

	if operatorDelegatorIndex >= len(s.tokenStrategyMapping) {
		return ErrRevertNotFound
	}

	tokenStrategyMapping := s.tokenStrategyMapping[operatorDelegatorIndex]
	if _, exist := tokenStrategyMapping[collateralToken]; !exist {
		return ErrRevertInvalidZeroInput
	}
	if amount.Cmp(bignumber.ZeroBI) == 0 {
		return ErrRevertInvalidZeroInput
	}

	return nil
}

// chooseOperatorDelegatorForDeposit: RestakeManager.chooseOperatorDelegatorForDeposit.
// Returns the index instead of the address of the chosen operator delegator.
func (s *PoolSimulator) chooseOperatorDelegatorForDeposit() (int, error) {
	if len(s.operatorDelegatorAllocations) == 0 {
		return 0, ErrRevertNotFound
	}
	if len(s.operatorDelegatorAllocations) == 1 {
		return 0, nil
	}

	var operatorDelegatorAllocationValue big.Int
	for i := 0; i < len(s.operatorDelegatorTVLs); i++ {
		operatorDelegatorAllocationValue.Mul(
			s.operatorDelegatorAllocations[i],
			s.totalTVL,
		)
		operatorDelegatorAllocationValue.Div(
			&operatorDelegatorAllocationValue,
			big.NewInt(10000),
		)

		if s.operatorDelegatorTVLs[i].Cmp(&operatorDelegatorAllocationValue) < 0 {
			return i, nil
		}
	}

	// Default to the first operator delegator
	return 0, nil
}
