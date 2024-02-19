package bancorv3

import (
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrZeroValue    = errors.New("zero value")
)

type (
	PoolSimulator struct {
		poolpkg.Pool

		_collectionByPool map[string]*poolCollection
		_bnt              string
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
)

func (p *PoolSimulator) tradeBySourceAmount(
	sourceToken,
	targetToken string,
	sourceAmount,
	minReturnAmount *uint256.Int,
) (*uint256.Int, []*poolCollectionTradeInfo, error) {
	if err := p._verifyTradeParams(
		sourceToken,
		targetToken,
		sourceAmount,
		minReturnAmount,
	); err != nil {
		return nil, nil, err
	}

	return p._trade(
		&tradeTokens{SourceToken: sourceToken, TargetToken: targetToken},
		&tradeParams{
			BySourceAmount: true,
			Amount:         sourceAmount,
			Limit:          minReturnAmount,
			IgnoreFees:     false,
		},
	)
}

func (p *PoolSimulator) _verifyTradeParams(
	sourceToken,
	targetToken string,
	amount,
	limit *uint256.Int,
) error {
	if sourceToken == targetToken {
		return ErrInvalidToken
	}

	if !amount.Gt(number.Zero) || !limit.Gt(number.Zero) {
		return ErrZeroValue
	}

	return nil
}

func (p *PoolSimulator) _trade(
	tokens *tradeTokens,
	params *tradeParams,
) (*uint256.Int, []*poolCollectionTradeInfo, error) {
	var (
		firstHopTradeResult *tradeResult
		lastHopTradeResult  *tradeResult

		tradeInfo []*poolCollectionTradeInfo

		err error
	)

	if strings.EqualFold(tokens.SourceToken, p._bnt) {
		lastHopTradeResult, err = p._tradeBNT(
			tokens.TargetToken, true, params,
		)
		if err != nil {
			return nil, nil, err
		}
		firstHopTradeResult = lastHopTradeResult

		tradeInfo = append(tradeInfo, lastHopTradeResult.PoolCollectionTradeInfo)

	} else if strings.EqualFold(tokens.TargetToken, p._bnt) {
		lastHopTradeResult, err = p._tradeBNT(tokens.SourceToken, false, params)
		if err != nil {
			return nil, nil, err
		}

		firstHopTradeResult = lastHopTradeResult

		tradeInfo = append(tradeInfo, lastHopTradeResult.PoolCollectionTradeInfo)

	} else {
		firstHopTradeResult, lastHopTradeResult, err = p._tradeBaseTokens(tokens, params)
		if err != nil {
			return nil, nil, err
		}

		tradeInfo = append(
			tradeInfo,
			firstHopTradeResult.PoolCollectionTradeInfo,
			lastHopTradeResult.PoolCollectionTradeInfo,
		)
	}

	return lastHopTradeResult.TargetAmount, tradeInfo, nil
}

func (p *PoolSimulator) _tradeBaseTokens(
	tokens *tradeTokens,
	params *tradeParams,
) (*tradeResult, *tradeResult, error) {
	sourceAmount, minReturnAmount := params.Amount, params.Limit

	targetHop1, err := p._tradeBNT(
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

	targetHop2, err := p._tradeBNT(
		tokens.TargetToken,
		true,
		&tradeParams{
			BySourceAmount: true,
			Amount:         targetHop1.TargetAmount,
			Limit:          minReturnAmount,
			IgnoreFees:     params.IgnoreFees,
		},
	)

	return targetHop1, targetHop2, nil
}

func (p *PoolSimulator) _tradeBNT(
	pool string, // token
	fromBNT bool,
	params *tradeParams,
) (*tradeResult, error) {
	var tokens tradeTokens
	if fromBNT {
		tokens = tradeTokens{
			SourceToken: p._bnt,
			TargetToken: pool,
		}
	} else {
		tokens = tradeTokens{
			SourceToken: pool,
			TargetToken: p._bnt,
		}
	}

	poolCollection, ok := p._collectionByPool[pool]
	if !ok {
		return nil, ErrInvalidToken
	}

	tradeAmountsAndFee, poolCollectionTradeInfo, err := poolCollection.tradeBySourceAmount(
		tokens.SourceToken,
		tokens.TargetToken,
		params.Amount,
		params.Limit,
		params.IgnoreFees,
	)
	if err != nil {
		return nil, err
	}

	tradeResult := tradeResult{
		SourceAmount:     params.Amount,
		TargetAmount:     tradeAmountsAndFee.Amount,
		TradingFeeAmount: tradeAmountsAndFee.TradingFeeAmount,
		NetworkFeeAmount: tradeAmountsAndFee.NetworkFeeAmount,

		PoolCollectionTradeInfo: poolCollectionTradeInfo,
	}

	return &tradeResult, nil
}
