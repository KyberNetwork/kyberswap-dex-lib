package woofiv2

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

var (
	ErrBaseTokenIsQuoteToken = errors.New("WooPPV2: baseToken==quoteToken")
	ErrOracleIsNotFeasible   = errors.New("WooPPV2: !ORACLE_FEASIBLE")
)

var (
	Number_1e5 = number.TenPow(5)
)

type PoolSimulator struct {
	quoteToken string
	tokenInfos map[string]TokenInfo
	decimals   map[string]uint8
	balances   map[string]*uint256.Int
	wooracle   Wooracle
}

type Wooracle struct {
	States   map[string]State `json:"states"`
	Decimals map[string]uint8 `json:"decimals"`
}

type TokenInfo struct {
	Reserve *uint256.Int `json:"reserve"`
	FeeRate uint16       `json:"feeRate"`
}

type State struct {
	Price      *uint256.Int `json:"price"`
	Spread     uint64       `json:"spread"`
	Coeff      uint64       `json:"coeff"`
	WoFeasible bool         `json:"woFeasible"`
}

// _sellBase
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L361
func (s *PoolSimulator) _sellBase(
	baseToken string,
	baseAmount *uint256.Int,
) (*uint256.Int, error) {
	if baseToken == s.quoteToken {
		return nil, ErrBaseTokenIsQuoteToken
	}

	state := s.wooracle.States[baseToken]

	quoteAmount, _, err := s._calcQuoteAmountSellBase(baseToken, baseAmount, state)
	if err != nil {
		return nil, err
	}

	swapFee := new(uint256.Int).Div(
		new(uint256.Int).Mul(
			quoteAmount,
			uint256.NewInt(uint64(s.tokenInfos[baseToken].FeeRate)),
		),
		Number_1e5,
	)

	quoteAmount = new(uint256.Int).Sub(quoteAmount, swapFee)

	return quoteAmount, nil
}

// _sellQuote
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L408
func (s *PoolSimulator) _sellQuote(
	baseToken string,
	quoteAmount *uint256.Int,
) (*uint256.Int, error) {
	if baseToken == s.quoteToken {
		return nil, ErrBaseTokenIsQuoteToken
	}

	swapFee := new(uint256.Int).Div(
		new(uint256.Int).Mul(
			quoteAmount,
			uint256.NewInt(uint64(s.tokenInfos[baseToken].FeeRate)),
		),
		Number_1e5,
	)

	quoteAmount = new(uint256.Int).Sub(quoteAmount, swapFee)

	state := s.wooracle.States[baseToken]

	baseAmount, _, err := s._calcBaseAmountSellQuote(baseToken, quoteAmount, state)
	if err != nil {
		return nil, err
	}

	return baseAmount, nil
}

// _swapBaseToBase
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L457
func (s *PoolSimulator) _swapBaseToBase(
	baseToken1 string,
	baseToken2 string,
	base1Amount *uint256.Int,
) (*uint256.Int, error) {
	state1, state2 := s.wooracle.States[baseToken1], s.wooracle.States[baseToken2]

	var spread uint64
	if state1.Spread > state2.Spread {
		spread = state1.Spread / 2
	} else {
		spread = state2.Spread / 2
	}

	var feeRate uint16
	if s.tokenInfos[baseToken1].FeeRate > s.tokenInfos[baseToken2].FeeRate {
		feeRate = s.tokenInfos[baseToken1].FeeRate
	} else {
		feeRate = s.tokenInfos[baseToken2].FeeRate
	}

	state1.Spread, state2.Spread = spread, spread

	quoteAmount, _, err := s._calcQuoteAmountSellBase(baseToken1, base1Amount, state1)
	if err != nil {
		return nil, err
	}

	swapFee := new(uint256.Int).Div(new(uint256.Int).Mul(quoteAmount, uint256.NewInt(uint64(feeRate))), Number_1e5)

	quoteAmount = new(uint256.Int).Sub(quoteAmount, swapFee)

	base2Amount, _, err := s._calcBaseAmountSellQuote(baseToken2, quoteAmount, state2)

	return base2Amount, nil
}

// _calcBaseAmountSellQuote
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L559
func (s *PoolSimulator) _calcBaseAmountSellQuote(
	baseToken string,
	quoteAmount *uint256.Int,
	state State,
) (*uint256.Int, *uint256.Int, error) {
	if !state.WoFeasible {
		return nil, nil, ErrOracleIsNotFeasible
	}

	desc := s.decimalInfo(baseToken)

	coef := new(uint256.Int).Sub(
		new(uint256.Int).Sub(
			number.Number_1e18,
			new(uint256.Int).Div(
				new(uint256.Int).Mul(quoteAmount, uint256.NewInt(state.Coeff)),
				desc.quoteDec,
			),
		),
		uint256.NewInt(state.Spread),
	)

	baseAmount := new(uint256.Int).Div(
		new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Div(
					new(uint256.Int).Mul(
						new(uint256.Int).Mul(
							quoteAmount,
							desc.baseDec,
						),
						desc.priceDec,
					),
					state.Price,
				),
				coef,
			),
			number.Number_1e18,
		),
		desc.quoteDec,
	)

	newPrice := new(uint256.Int).Div(
		new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Add(
					new(uint256.Int).Mul(number.Number_1e18, desc.quoteDec),
					new(uint256.Int).Mul(number.Number_2, new(uint256.Int).Mul(uint256.NewInt(state.Coeff), quoteAmount)),
				),
				state.Price,
			),
			desc.quoteDec,
		),
		number.Number_1e18,
	)

	return baseAmount, newPrice, nil
}

// _calcQuoteAmountSellBase
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L559
func (s *PoolSimulator) _calcQuoteAmountSellBase(
	baseToken string,
	baseAmount *uint256.Int,
	state State,
) (*uint256.Int, *uint256.Int, error) {
	if !state.WoFeasible {
		return nil, nil, ErrOracleIsNotFeasible
	}

	decs := s.decimalInfo(baseToken)

	coef := new(uint256.Int).Sub(
		new(uint256.Int).Sub(
			number.Number_1e18,
			new(uint256.Int).Div(
				new(uint256.Int).Div(
					new(uint256.Int).Mul(uint256.NewInt(state.Coeff), new(uint256.Int).Mul(baseAmount, state.Price)),
					decs.baseDec,
				),
				decs.priceDec,
			),
		),
		uint256.NewInt(state.Spread),
	)

	quoteAmount := new(uint256.Int).Div(
		new(uint256.Int).Div(
			new(uint256.Int).Mul(
				new(uint256.Int).Div(
					new(uint256.Int).Mul(baseAmount, new(uint256.Int).Mul(decs.quoteDec, state.Price)),
					decs.priceDec,
				),
				coef,
			),
			number.Number_1e18,
		),
		decs.baseDec,
	)

	newPrice := new(uint256.Int).Div(
		new(uint256.Int).Sub(
			number.Number_1e18,
			new(uint256.Int).Mul(
				new(uint256.Int).Div(
					new(uint256.Int).Div(
						new(uint256.Int).Mul(
							new(uint256.Int).Mul(number.Number_2, uint256.NewInt(state.Coeff)),
							new(uint256.Int).Mul(state.Price, baseAmount),
						),
						decs.priceDec,
					),
					decs.baseDec,
				),
				state.Price,
			),
		),
		number.Number_1e18,
	)

	return quoteAmount, newPrice, nil
}

// decimalInfo
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/WooPPV2.sol#L181
func (s *PoolSimulator) decimalInfo(baseToken string) DecimalInfo {
	return DecimalInfo{
		priceDec: number.TenPow(s.wooracle.Decimals[baseToken]), // 8
		quoteDec: number.TenPow(s.decimals[s.quoteToken]),       // 18 or 6
		baseDec:  number.TenPow(s.decimals[baseToken]),          // 18 or 8
	}
}
