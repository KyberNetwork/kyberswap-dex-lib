package bancorv3

import (
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bancor-v3/math"
)

var (
	ErrInsufficientLiquidity    = errors.New("insufficient liquidity")
	ErrInsufficientTargetAmount = errors.New("insufficient target amount")
	ErrInsufficientSourceAmount = errors.New("insufficient source amount")
	ErrDoesNotExit              = errors.New("does not exit")
	ErrTradeDisabled            = errors.New("trade disabled")
)

type (
	poolCollection struct {
		NetworkFeePMM *uint256.Int     `json:"networkFeePMM"`
		PoolData      map[string]*pool `json:"poolData"`
		BNT           string           `json:"bnt"`
	}

	pool struct {
		PoolToken      string         `json:"poolToken"`
		TradingFeePPM  *uint256.Int   `json:"tradingFeePPM"`
		TradingEnabled bool           `json:"tradingEnabled"`
		Liquidity      *poolLiquidity `json:"liquidity"`
	}

	poolLiquidity struct {
		BNTTradingLiquidity       *uint256.Int `json:"bntTradingLiquidity"`
		BaseTokenTradingLiquidity *uint256.Int `json:"baseTokenTradingLiquidity"`
		StakedBalance             *uint256.Int `json:"stakedBalance"`
	}

	tradeAmountAndTradingFee struct {
		Amount           *uint256.Int
		TradingFeeAmount *uint256.Int
	}

	tradeIntermediateResult struct {
		SourceAmount     *uint256.Int
		TargetAmount     *uint256.Int
		Limit            *uint256.Int
		TradingFeeAmount *uint256.Int
		NetworkFeeAmount *uint256.Int
		SourceBalance    *uint256.Int
		TargetBalance    *uint256.Int
		StakedBalance    *uint256.Int
		Pool             string // token
		IsSourceBNT      bool
		BySourceAmount   bool
		TradingFeePPM    *uint256.Int
	}

	tradeAmountAndFee struct {
		Amount           *uint256.Int
		TradingFeeAmount *uint256.Int
		NetworkFeeAmount *uint256.Int
	}

	poolCollectionTradeInfo struct {
		Pool             string
		NewPoolLiquidity *poolLiquidity
	}
)

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/pools/PoolCollection.sol#L816
func (p *poolCollection) tradeBySourceAmount(
	sourceToken,
	targetToken string,
	sourceAmount,
	minReturnAmount *uint256.Int,
	ignoreFees bool,
) (*tradeAmountAndFee, *poolCollectionTradeInfo, error) {
	result, err := p._initTrade(
		sourceToken,
		targetToken,
		sourceAmount,
		minReturnAmount,
		true,
	)
	if err != nil {
		return nil, nil, err
	}

	if ignoreFees {
		result.TradingFeePPM = number.Zero
	}

	newLiquidity, err := p._performTrade(result)
	if err != nil {
		return nil, nil, err
	}

	tradeAmountAndFee := tradeAmountAndFee{
		Amount:           result.TargetAmount,
		TradingFeeAmount: result.TradingFeeAmount,
		NetworkFeeAmount: result.NetworkFeeAmount,
	}

	poolCollectionTradeInfo := poolCollectionTradeInfo{
		Pool:             result.Pool,
		NewPoolLiquidity: newLiquidity,
	}

	return &tradeAmountAndFee, &poolCollectionTradeInfo, nil
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/pools/PoolCollection.sol#L855
func (p *poolCollection) tradeByTargetAmount(
	sourceToken,
	targetToken string,
	targetAmount,
	maxSourceAmount *uint256.Int,
	ignoreFees bool,
) (*tradeAmountAndFee, *poolCollectionTradeInfo, error) {
	result, err := p._initTrade(
		sourceToken,
		targetToken,
		targetAmount,
		maxSourceAmount,
		false,
	)
	if err != nil {
		return nil, nil, err
	}

	if ignoreFees {
		result.TradingFeePPM = number.Zero
	}

	newLiquidity, err := p._performTrade(result)
	if err != nil {
		return nil, nil, err
	}

	tradeAmountAndFee := tradeAmountAndFee{
		Amount:           result.SourceAmount,
		TradingFeeAmount: result.TradingFeeAmount,
		NetworkFeeAmount: result.NetworkFeeAmount,
	}

	poolCollectionTradeInfo := poolCollectionTradeInfo{
		Pool:             result.Pool,
		NewPoolLiquidity: newLiquidity,
	}

	return &tradeAmountAndFee, &poolCollectionTradeInfo, nil
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/pools/PoolCollection.sol#L1466
/**
 * @dev returns initial trading params
 */
func (p *poolCollection) _initTrade(
	sourceToken,
	targetToken string,
	amount,
	limit *uint256.Int,
	bySourceAmount bool,
) (*tradeIntermediateResult, error) {
	result := tradeIntermediateResult{
		SourceAmount:     number.Zero,
		TargetAmount:     number.Zero,
		Limit:            number.Zero,
		TradingFeeAmount: number.Zero,
		NetworkFeeAmount: number.Zero,
		SourceBalance:    number.Zero,
		TargetBalance:    number.Zero,
		StakedBalance:    number.Zero,
		TradingFeePPM:    number.Zero,
	}

	isSourceBNT := strings.EqualFold(sourceToken, p.BNT)
	isTargetBNT := strings.EqualFold(targetToken, p.BNT)

	if isSourceBNT && !isTargetBNT {
		result.IsSourceBNT = true
		result.Pool = targetToken
	} else if !isSourceBNT && isTargetBNT {
		result.IsSourceBNT = false
		result.Pool = sourceToken
	} else {
		return nil, ErrDoesNotExit
	}

	data, err := p._poolStorage(result.Pool)
	if err != nil {
		return nil, err
	}

	if !data.TradingEnabled {
		return nil, ErrTradeDisabled
	}

	result.BySourceAmount = bySourceAmount
	if result.BySourceAmount {
		result.SourceAmount = amount
	} else {
		result.TargetAmount = amount
	}

	result.Limit = limit
	result.TradingFeePPM = data.TradingFeePPM

	liquidity := data.Liquidity
	if result.IsSourceBNT {
		result.SourceBalance = liquidity.BNTTradingLiquidity
		result.TargetBalance = liquidity.BaseTokenTradingLiquidity
	} else {
		result.SourceBalance = liquidity.BaseTokenTradingLiquidity
		result.TargetBalance = liquidity.BNTTradingLiquidity
	}

	result.StakedBalance = liquidity.StakedBalance

	return &result, nil
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/pools/PoolCollection.sol#L1183
/**
 * @dev returns a storage reference to pool data
 */
func (p *poolCollection) _poolStorage(pool string) (*pool, error) {
	data, ok := p.PoolData[pool]
	if !ok {
		return nil, ErrDoesNotExit
	}
	return data, nil
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/pools/PoolCollection.sol#L1648
/**
 * @dev performs a trade
 */
func (p *poolCollection) _performTrade(result *tradeIntermediateResult) (*poolLiquidity, error) {
	if err := p._processTrade(result); err != nil {
		return nil, err
	}

	newLiquidity := poolLiquidity{
		StakedBalance: result.StakedBalance,
	}
	if result.IsSourceBNT {
		newLiquidity.BNTTradingLiquidity = result.SourceBalance
		newLiquidity.BaseTokenTradingLiquidity = result.TargetBalance
	} else {
		newLiquidity.BNTTradingLiquidity = result.TargetBalance
		newLiquidity.BaseTokenTradingLiquidity = result.SourceBalance
	}

	return &newLiquidity, nil
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/pools/PoolCollection.sol#L1564
/**
 * @dev processes a trade by providing either the source or the target amount and updates the in-memory intermediate
 * result
 */
func (p *poolCollection) _processTrade(result *tradeIntermediateResult) error {
	var (
		tradeAmountAndFee *tradeAmountAndTradingFee
		err               error
	)

	if result.BySourceAmount {
		tradeAmountAndFee, err = p._tradeAmountAndFeeBySourceAmount(
			result.SourceBalance,
			result.TargetBalance,
			result.TradingFeePPM,
			result.SourceAmount,
		)
		if err != nil {
			return err
		}

		result.TargetAmount = tradeAmountAndFee.Amount

		// ensure that the target amount is above the requested minimum return amount
		if result.TargetAmount.Cmp(result.Limit) < 0 {
			return ErrInsufficientTargetAmount
		}
	} else {
		tradeAmountAndFee, err = p._tradeAmountAndFeeByTargetAmount(
			result.SourceBalance,
			result.TargetBalance,
			result.TradingFeePPM,
			result.TargetAmount,
		)
		if err != nil {
			return err
		}

		result.SourceAmount = tradeAmountAndFee.Amount

		// ensure that the user has provided enough tokens to make the trade
		if result.SourceAmount.Cmp(number.Zero) == 0 || result.SourceAmount.Cmp(result.Limit) > 0 {
			return ErrInsufficientSourceAmount
		}
	}

	result.TradingFeeAmount = tradeAmountAndFee.TradingFeeAmount

	// sync the trading and staked balance
	result.SourceBalance = new(uint256.Int).Add(result.SourceBalance, result.SourceAmount)
	result.TargetBalance = new(uint256.Int).Sub(result.TargetBalance, result.TargetAmount)

	if result.IsSourceBNT {
		result.StakedBalance = new(uint256.Int).Add(result.StakedBalance, result.TradingFeeAmount)
	}

	return p._processNetworkFee(result)
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/pools/PoolCollection.sol#L1613
/**
 * @dev processes the network fee and updates the in-memory intermediate result
 */
func (p *poolCollection) _processNetworkFee(result *tradeIntermediateResult) error {
	if p.NetworkFeePMM.IsZero() {
		return nil
	}

	targetNetworkFeeAmount, err := math.MathEx.MulDivF(
		result.TradingFeeAmount,
		p.NetworkFeePMM,
		pmmResolution,
	)
	if err != nil {
		return err
	}

	result.TargetBalance = new(uint256.Int).Sub(result.TargetBalance, targetNetworkFeeAmount)

	if !result.IsSourceBNT {
		result.NetworkFeeAmount = targetNetworkFeeAmount
		return nil
	}

	t, err := p._tradeAmountAndFeeBySourceAmount(
		result.TargetBalance,
		result.SourceBalance,
		number.Zero,
		targetNetworkFeeAmount,
	)
	if err != nil {
		return err
	}
	result.NetworkFeeAmount = t.Amount

	result.TargetBalance.Add(result.TargetBalance, targetNetworkFeeAmount)
	result.SourceBalance.Sub(result.SourceBalance, result.NetworkFeeAmount)
	result.StakedBalance.Sub(result.StakedBalance, targetNetworkFeeAmount)

	return nil
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/pools/PoolCollection.sol#L1523
/**
 * @dev returns trade amount and fee by providing the source amount
 */
func (p *poolCollection) _tradeAmountAndFeeBySourceAmount(
	sourceBalance,
	targetBalance *uint256.Int,
	feePPM *uint256.Int,
	sourceAmount *uint256.Int,
) (*tradeAmountAndTradingFee, error) {
	if sourceBalance.IsZero() || targetBalance.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	targetAmount, err := math.MathEx.MulDivF(
		targetBalance, sourceAmount, new(uint256.Int).Add(sourceBalance, sourceAmount),
	)
	if err != nil {
		return nil, err
	}

	tradingFeeAmount, err := math.MathEx.MulDivF(targetAmount, feePPM, pmmResolution)
	if err != nil {
		return nil, err
	}

	return &tradeAmountAndTradingFee{
		Amount:           new(uint256.Int).Sub(targetAmount, tradingFeeAmount),
		TradingFeeAmount: tradingFeeAmount,
	}, nil
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/pools/PoolCollection.sol#L1543
/**
 * @dev returns trade amount and fee by providing either the target amount
 */
func (p *poolCollection) _tradeAmountAndFeeByTargetAmount(
	sourceBalance,
	targetBalance *uint256.Int,
	feePPM *uint256.Int,
	targetAmount *uint256.Int,
) (*tradeAmountAndTradingFee, error) {
	if sourceBalance.IsZero() {
		return nil, ErrInsufficientLiquidity
	}

	tradingFeeAmount, err := math.MathEx.MulDivF(
		targetAmount, feePPM, new(uint256.Int).Sub(pmmResolution, feePPM),
	)
	if err != nil {
		return nil, err
	}

	fullTargetAmount := new(uint256.Int).Add(targetAmount, tradingFeeAmount)

	sourceAmount, err := math.MathEx.MulDivF(sourceBalance, fullTargetAmount, new(uint256.Int).Sub(targetBalance, fullTargetAmount))
	if err != nil {
		return nil, err
	}

	return &tradeAmountAndTradingFee{
		Amount:           sourceAmount,
		TradingFeeAmount: tradingFeeAmount,
	}, nil
}
