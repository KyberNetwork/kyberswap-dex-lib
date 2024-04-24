//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple PoolSimulator
//msgp:ignore traceTokens traceParams tradeResult Gas
//msgp:shim *uint256.Int as:[]byte using:msgpencode.EncodeUint256/msgpencode.DecodeUint256
//msgp:shim valueobject.ChainID as:uint using:uint/valueobject.ChainID

package bancorv3

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrZeroValue    = errors.New("zero value")
	ErrOverflow     = errors.New("overflow")
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		collectionByPool map[string]string
		poolCollections  map[string]*poolCollection
		bnt              string
		chainID          valueobject.ChainID
	}

	tradeTokens struct {
		SourceToken string
		TargetToken string
	}

	tradeParams struct {
		Amount         *uint256.Int
		Limit          *uint256.Int
		BySourceAmount bool
		IgnoreFees     bool
	}

	tradeResult struct {
		SourceAmount     *uint256.Int
		TargetAmount     *uint256.Int
		TradingFeeAmount *uint256.Int
		NetworkFeeAmount *uint256.Int

		PoolCollectionTradeInfo *poolCollectionTradeInfo
	}

	Gas struct {
		Swap int64
	}
)

var (
	defaultGas = Gas{Swap: 150000}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var (
		extra       Extra
		staticExtra StaticExtra

		tokens   = make([]string, len(entityPool.Tokens))
		reserves = make([]*big.Int, len(entityPool.Tokens))
	)

	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	for idx := 0; idx < len(entityPool.Tokens); idx++ {
		tokens[idx] = entityPool.Tokens[idx].Address
		reserves[idx] = bignumber.NewBig10(entityPool.Reserves[idx])
	}

	poolInfo := poolpkg.PoolInfo{
		Address:     entityPool.Address,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		Checked:     true,
		BlockNumber: uint64(entityPool.BlockNumber),
	}

	return &PoolSimulator{
		Pool:             poolpkg.Pool{Info: poolInfo},
		collectionByPool: extra.CollectionByPool,
		poolCollections:  extra.PoolCollections,
		bnt:              staticExtra.BNT,
		chainID:          staticExtra.ChainID,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut

	sourceToken, targetToken, isSourceNative, isTargetNative, err := s.transformTokens(tokenAmountIn.Token, tokenOut)
	if err != nil {
		return nil, err
	}

	if err := s.verifyTokens(sourceToken, targetToken); err != nil {
		return nil, err
	}

	sourceAmount, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	amountOut, feeAmount, tradeInfo, err := s.tradeBySourceAmount(
		sourceToken,
		targetToken,
		sourceAmount,
		number.Number_1,
	)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &poolpkg.TokenAmount{
			Token:  tokenOut, // fee is applied on the target token https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/pools/PoolCollection.sol#L1534
			Amount: feeAmount.ToBig(),
		},
		Gas: defaultGas.Swap,
		SwapInfo: SwapInfo{
			IsSourceNative: isSourceNative,
			IsTargetNative: isTargetNative,
			TradeInfo:      tradeInfo,
		},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(params poolpkg.CalcAmountInParams) (*poolpkg.CalcAmountInResult, error) {
	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn

	sourceToken, targetToken, isSourceNative, isTargetNative, err := s.transformTokens(tokenIn, tokenAmountOut.Token)
	if err != nil {
		return nil, err
	}

	if err := s.verifyTokens(sourceToken, targetToken); err != nil {
		return nil, err
	}

	targetAmount, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	amountIn, feeAmount, tradeInfo, err := s.tradeByTargetAmount(
		sourceToken,
		targetToken,
		targetAmount,
		number.MaxU256,
	)
	if err != nil {
		return nil, err
	}

	return &poolpkg.CalcAmountInResult{
		TokenAmountIn: &poolpkg.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn.ToBig(),
		},
		Fee: &poolpkg.TokenAmount{
			Token:  tokenAmountOut.Token, // fee is applied on the target token https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/pools/PoolCollection.sol#L1553
			Amount: feeAmount.ToBig(),
		},
		Gas: defaultGas.Swap,
		SwapInfo: SwapInfo{
			IsSourceNative: isSourceNative,
			IsTargetNative: isTargetNative,
			TradeInfo:      tradeInfo,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	for _, info := range swapInfo.TradeInfo {
		polCol := s.collectionByPool[info.Pool]
		s.poolCollections[polCol].PoolData[info.Pool].Liquidity = info.NewPoolLiquidity
	}
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) interface{} {
	return PoolMetaInfo{}
}

func (s *PoolSimulator) verifyTokens(sourceToken, targetToken string) error {
	if _, err := s.getPoolData(sourceToken); err != nil {
		return err
	}

	if _, err := s.getPoolData(targetToken); err != nil {
		return err
	}

	// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/network/BancorNetwork.sol#L1100
	// This is a part of _verifyTradeParams function
	if sourceToken == targetToken {
		return ErrInvalidToken
	}

	return nil
}

func (s *PoolSimulator) transformTokens(tokenIn, tokenOut string) (string, string, bool, bool, error) {
	weth := strings.ToLower(valueobject.WETHByChainID[s.chainID])
	if tokenIn != weth && tokenOut != weth {
		return tokenIn, tokenOut, false, false, nil
	}

	var (
		sourceToken = tokenIn
		targetToken = tokenOut

		isSourceNative, isTargetNative bool
	)

	var (
		eth = strings.ToLower(valueobject.EtherAddress)

		ethReserve  *uint256.Int
		wethReserve *uint256.Int
	)

	ethPoolData, err := s.getPoolData(eth)
	if err != nil {
		ethReserve = number.Zero
	} else {
		ethReserve = ethPoolData.Liquidity.StakedBalance
	}

	wethPoolData, err := s.getPoolData(weth)
	if err != nil {
		wethReserve = number.Zero
	} else {
		wethReserve = wethPoolData.Liquidity.StakedBalance
	}

	if tokenIn == weth && ethReserve.Cmp(wethReserve) > 0 {
		sourceToken = eth
		isSourceNative = true
	}

	if (tokenOut == weth) &&
		((tokenIn != weth && ethReserve.Cmp(wethReserve) > 0) ||
			(tokenIn == weth && !isSourceNative)) {
		targetToken = eth
		isTargetNative = true
	}

	return sourceToken, targetToken, isSourceNative, isTargetNative, nil
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/network/BancorNetwork.sol#L1115
/**
 * @dev internal trade by source amount logic
 */
func (s *PoolSimulator) tradeBySourceAmount(
	sourceToken,
	targetToken string,
	sourceAmount,
	minReturnAmount *uint256.Int,
) (*uint256.Int, *uint256.Int, []*poolCollectionTradeInfo, error) {
	if err := s._verifyTradeParams(
		sourceToken,
		targetToken,
		sourceAmount,
		minReturnAmount,
	); err != nil {
		return nil, nil, nil, err
	}

	return s._trade(
		&tradeTokens{SourceToken: sourceToken, TargetToken: targetToken},
		&tradeParams{
			BySourceAmount: true,
			Amount:         sourceAmount,
			Limit:          minReturnAmount,
			IgnoreFees:     false,
		},
	)
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/network/BancorNetwork.sol#L1147
/**
 * @dev internal trade by source amount logic
 */
func (s *PoolSimulator) tradeByTargetAmount(
	sourceToken,
	targetToken string,
	targetAmount,
	maxSourceAmount *uint256.Int,
) (*uint256.Int, *uint256.Int, []*poolCollectionTradeInfo, error) {
	if err := s._verifyTradeParams(
		sourceToken,
		targetToken,
		targetAmount,
		maxSourceAmount,
	); err != nil {
		return nil, nil, nil, err
	}

	return s._trade(
		&tradeTokens{SourceToken: sourceToken, TargetToken: targetToken},
		&tradeParams{
			BySourceAmount: false,
			Amount:         targetAmount,
			Limit:          maxSourceAmount,
			IgnoreFees:     false,
		},
	)
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/network/BancorNetwork.sol#L1090
/**
 * @dev verifies that the provided trade params are valid
 */
func (s *PoolSimulator) _verifyTradeParams(
	_,
	_ string,
	amount,
	limit *uint256.Int,
) error {
	if !amount.Gt(number.Zero) || !limit.Gt(number.Zero) {
		return ErrZeroValue
	}

	return nil
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/network/BancorNetwork.sol#L1189
/**
 * @dev performs a trade by providing either the source or target amount:
 *
 * - when trading by the source amount, the amount represents the source amount and the limit is the minimum return
 *   amount
 * - when trading by the target amount, the amount represents the target amount and the limit is the maximum source
 *   amount
 *
 * requirements:
 *
 * - the caller must have approved the network to transfer the source tokens on its behalf (except for in the
 *   native token case)
 */
func (s *PoolSimulator) _trade(
	tokens *tradeTokens,
	params *tradeParams,
) (*uint256.Int, *uint256.Int, []*poolCollectionTradeInfo, error) {
	var (
		firstHopTradeResult *tradeResult
		lastHopTradeResult  *tradeResult

		tradeInfo []*poolCollectionTradeInfo

		err error
	)

	if strings.EqualFold(tokens.SourceToken, s.bnt) {
		lastHopTradeResult, err = s._tradeBNT(
			tokens.TargetToken, true, params,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		firstHopTradeResult = lastHopTradeResult

		tradeInfo = append(tradeInfo, lastHopTradeResult.PoolCollectionTradeInfo)

	} else if strings.EqualFold(tokens.TargetToken, s.bnt) {
		lastHopTradeResult, err = s._tradeBNT(tokens.SourceToken, false, params)
		if err != nil {
			return nil, nil, nil, err
		}

		firstHopTradeResult = lastHopTradeResult

		tradeInfo = append(tradeInfo, lastHopTradeResult.PoolCollectionTradeInfo)

	} else {
		firstHopTradeResult, lastHopTradeResult, err = s._tradeBaseTokens(tokens, params)
		if err != nil {
			return nil, nil, nil, err
		}

		tradeInfo = append(
			tradeInfo,
			firstHopTradeResult.PoolCollectionTradeInfo,
			lastHopTradeResult.PoolCollectionTradeInfo,
		)
	}

	if params.BySourceAmount {
		return lastHopTradeResult.TargetAmount, lastHopTradeResult.TradingFeeAmount, tradeInfo, nil
	}

	return firstHopTradeResult.SourceAmount, firstHopTradeResult.TradingFeeAmount, tradeInfo, nil
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/network/BancorNetwork.sol#L1352
/**
 * @dev performs a double hop trade between two base tokens by providing either the source or the target amount
 *
 * - when trading by the source amount, the amount represents the source amount and the limit is the minimum return
 *   amount
 * - when trading by the target amount, the amount represents the target amount and the limit is the maximum source
 *   amount
 */
func (s *PoolSimulator) _tradeBaseTokens(
	tokens *tradeTokens,
	params *tradeParams,
) (*tradeResult, *tradeResult, error) {
	if params.BySourceAmount {
		sourceAmount, minReturnAmount := params.Amount, params.Limit

		// trade source tokens to BNT (while accepting any return amount)
		targetHop1, err := s._tradeBNT(
			tokens.SourceToken,
			false,
			&tradeParams{
				BySourceAmount: true,
				Amount:         sourceAmount,
				Limit:          number.Number_1,
				IgnoreFees:     params.IgnoreFees,
			},
		)
		if err != nil {
			return nil, nil, err
		}

		// trade the received BNT target amount to target tokens (while respecting the minimum return amount)
		targetHop2, err := s._tradeBNT(
			tokens.TargetToken,
			true,
			&tradeParams{
				BySourceAmount: true,
				Amount:         targetHop1.TargetAmount,
				Limit:          minReturnAmount,
				IgnoreFees:     params.IgnoreFees,
			},
		)
		if err != nil {
			return nil, nil, err
		}

		return targetHop1, targetHop2, nil
	}

	targetAmount, maxSourceAmount := params.Amount, params.Limit

	// trade any amount of BNT to get the requested target amount (we will use the actual traded amount to restrict
	// the trade from the source)
	sourceHop2, err := s._tradeBNT(
		tokens.TargetToken,
		true,
		&tradeParams{
			BySourceAmount: false,
			Amount:         targetAmount,
			Limit:          number.MaxU256,
			IgnoreFees:     params.IgnoreFees,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	// trade source tokens to the required amount of BNT (while respecting the maximum source amount)
	sourceHop1, err := s._tradeBNT(
		tokens.SourceToken,
		false,
		&tradeParams{
			BySourceAmount: false,
			Amount:         sourceHop2.SourceAmount,
			Limit:          maxSourceAmount,
			IgnoreFees:     params.IgnoreFees,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	return sourceHop1, sourceHop2, nil
}

// https://github.com/bancorprotocol/contracts-v3/blob/4d3070bf3a759106ec8e6c9ab5ad5d71ff15f3ee/contracts/network/BancorNetwork.sol#L1297
/**
 * @dev performs a single hop between BNT and a base token trade by providing either the source or the target amount
 *
 * - when trading by the source amount, the amount represents the source amount and the limit is the minimum return
 *   amount
 * - when trading by the target amount, the amount represents the target amount and the limit is the maximum source
 *   amount
 */
func (s *PoolSimulator) _tradeBNT(
	pool string, // token
	fromBNT bool,
	params *tradeParams,
) (*tradeResult, error) {
	var tokens tradeTokens
	if fromBNT {
		tokens = tradeTokens{
			SourceToken: s.bnt,
			TargetToken: pool,
		}
	} else {
		tokens = tradeTokens{
			SourceToken: pool,
			TargetToken: s.bnt,
		}
	}

	poolCollection, err := s.getPoolCollection(pool)
	if err != nil {
		return nil, err
	}

	var (
		tradeAmountsAndFee      *tradeAmountAndFee
		poolCollectionTradeInfo *poolCollectionTradeInfo
	)

	if params.BySourceAmount {
		tradeAmountsAndFee, poolCollectionTradeInfo, err = poolCollection.tradeBySourceAmount(
			tokens.SourceToken,
			tokens.TargetToken,
			params.Amount,
			params.Limit,
			params.IgnoreFees,
		)
		if err != nil {
			return nil, err
		}
	} else {
		tradeAmountsAndFee, poolCollectionTradeInfo, err = poolCollection.tradeByTargetAmount(
			tokens.SourceToken,
			tokens.TargetToken,
			params.Amount,
			params.Limit,
			params.IgnoreFees,
		)
		if err != nil {
			return nil, err
		}
	}

	var tr tradeResult
	if params.BySourceAmount {
		tr = tradeResult{
			SourceAmount:     params.Amount,
			TargetAmount:     tradeAmountsAndFee.Amount,
			TradingFeeAmount: tradeAmountsAndFee.TradingFeeAmount,
			NetworkFeeAmount: tradeAmountsAndFee.NetworkFeeAmount,

			PoolCollectionTradeInfo: poolCollectionTradeInfo,
		}
	} else {
		tr = tradeResult{
			SourceAmount:     tradeAmountsAndFee.Amount,
			TargetAmount:     params.Amount,
			TradingFeeAmount: tradeAmountsAndFee.TradingFeeAmount,
			NetworkFeeAmount: tradeAmountsAndFee.NetworkFeeAmount,

			PoolCollectionTradeInfo: poolCollectionTradeInfo,
		}
	}

	return &tr, nil
}

func (s *PoolSimulator) getPoolCollection(pool string) (*poolCollection, error) {
	poolCollectionAddr, ok := s.collectionByPool[pool]
	if !ok {
		return nil, ErrInvalidToken
	}
	poolCollection, ok := s.poolCollections[poolCollectionAddr]
	if !ok {
		return nil, ErrPoolCollectionNotFound
	}
	return poolCollection, nil
}

func (s *PoolSimulator) getPoolData(pool string) (*pool, error) {
	poolCollection, err := s.getPoolCollection(pool)
	if err != nil {
		return nil, err
	}
	p, ok := poolCollection.PoolData[pool]
	if !ok {
		return nil, ErrPoolDataNotFound
	}
	return p, nil
}
