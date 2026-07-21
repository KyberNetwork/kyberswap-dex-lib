package pools

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	ekubomath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/quoting"
)

type (
	Ve33PoolState[S any] struct {
		UnderlyingPoolState S      `json:"underlyingPoolState"`
		SwapFee             uint64 `json:"swapFee"`
	}

	Ve33Pool struct {
		Pool
		extension common.Address
		swapFee   uint64
	}
)

func (p *Ve33Pool) GetKey() IPoolKey {
	return p.Pool.GetKey()
}

func (p *Ve33Pool) GetState() any {
	return NewVe33PoolState(p.Pool.GetState(), p.swapFee)
}

func (p *Ve33Pool) CloneSwapStateOnly() Pool {
	cloned := *p
	cloned.Pool = p.Pool.CloneSwapStateOnly()
	return &cloned
}

func (p *Ve33Pool) SetSwapState(state quoting.SwapState) {
	p.Pool.SetSwapState(state)
}

func (p *Ve33Pool) ApplyEvent(event Event, data []byte, blockTimestamp uint64) error {
	if event != EventVoteWeightApplied {
		return p.Pool.ApplyEvent(event, data, blockTimestamp)
	}

	swapFee, matches, err := parseVoteWeightAppliedEventIfMatching(data, p.GetKey())
	if err != nil {
		return err
	}
	if matches {
		p.swapFee = swapFee
	}
	return nil
}

func (p *Ve33Pool) NewBlock() {
	p.Pool.NewBlock()
}

func (p *Ve33Pool) Quote(amount *uint256.Int, isToken1 bool) (*quoting.Quote, error) {
	quote, err := p.Pool.Quote(amount, isToken1)
	if err != nil {
		return nil, err
	}

	quote.SwapInfo.Forward = &p.extension
	quote.Gas += quoting.ExtraBaseGasCostOfOneVe33Swap

	if p.swapFee == 0 || quote.CalculatedAmount.IsZero() {
		return quote, nil
	}

	if amount.Sign() >= 0 {
		fee := ekubomath.ComputeFee(quote.CalculatedAmount, p.swapFee)
		quote.CalculatedAmount.Sub(quote.CalculatedAmount, fee)
		quote.FeesPaid.Add(quote.FeesPaid, fee)
	} else {
		includingFee, err := ekubomath.AmountBeforeFee(quote.CalculatedAmount, p.swapFee)
		if err != nil {
			return nil, fmt.Errorf("amount before Ve33 fee: %w", err)
		}
		fee := new(uint256.Int).Sub(includingFee, quote.CalculatedAmount)
		quote.CalculatedAmount = includingFee
		quote.FeesPaid.Add(quote.FeesPaid, fee)
	}

	return quote, nil
}

func (p *Ve33Pool) CalcBalances() ([]uint256.Int, error) {
	return p.Pool.CalcBalances()
}

func NewVe33PoolState[S any](underlyingPoolState S, swapFee uint64) *Ve33PoolState[S] {
	return &Ve33PoolState[S]{
		UnderlyingPoolState: underlyingPoolState,
		SwapFee:             swapFee,
	}
}

func NewVe33Pool(underlyingPool Pool, swapFee uint64) *Ve33Pool {
	return &Ve33Pool{
		Pool:      underlyingPool,
		extension: underlyingPool.GetKey().Extension(),
		swapFee:   swapFee,
	}
}

func parseVoteWeightAppliedEventIfMatching(data []byte, poolKey IPoolKey) (uint64, bool, error) {
	if len(data) < 192 {
		return 0, false, fmt.Errorf("invalid VoteWeightApplied event data length: %d", len(data))
	}

	expectedPoolID, err := poolKey.NumId()
	if err != nil {
		return 0, false, fmt.Errorf("computing expected pool id: %w", err)
	}
	if !bytes.Equal(data[64:96], expectedPoolID) {
		return 0, false, nil
	}

	return binary.BigEndian.Uint64(data[184:192]), true, nil
}
