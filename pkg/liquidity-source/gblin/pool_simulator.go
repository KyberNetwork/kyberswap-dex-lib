package gblin

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator prices GBLIN mints purely from tracked state.
// Token order: Info.Tokens[0] = WETH, Info.Tokens[1] = GBLIN (the vault). Only WETH -> GBLIN is served: the vault
// has no exact-output mint, and a redemption pays out the whole basket in kind, which is not a single-token swap.
type PoolSimulator struct {
	pool.Pool
	extra Extra
	gas   int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	if len(entityPool.Tokens) != 2 || len(entityPool.Reserves) != 2 {
		return nil, fmt.Errorf("invalid pool tokens/reserves length: %d/%d",
			len(entityPool.Tokens), len(entityPool.Reserves))
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	for i, token := range entityPool.Tokens {
		tokens[i] = strings.ToLower(token.Address)
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		if reserves[i] == nil {
			reserves[i] = new(big.Int)
		}
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     strings.ToLower(entityPool.Address),
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		}},
		extra: extra,
		gas:   defaultMintGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := p.GetTokenIndex(params.TokenAmountIn.Token), p.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}
	if indexIn != 0 {
		return nil, ErrUnsupportedSwap
	}
	if params.TokenAmountIn.Amount == nil {
		return nil, ErrInvalidAmountIn
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.IsZero() {
		return nil, ErrInvalidAmountIn
	}

	info, fee, err := p.mint(amountIn)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: info.SharesOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: fee.ToBig()},
		Gas:            p.gas,
		SwapInfo:       info,
	}, nil
}

// mint replicates GBLIN._mintGBLIN: the management fee is accrued first, then the deposit is priced on the NAV
// before it, with both fees taken off the input and the protocol fee minted as shares at the same price.
func (p *PoolSimulator) mint(amountIn *uint256.Int) (SwapInfo, *uint256.Int, error) {
	e := &p.extra
	if e.Supply == nil || e.NavEth == nil || e.MinDeposit == nil {
		return SwapInfo{}, nil, ErrStateUnavailable
	}
	if !e.NavReliable {
		return SwapInfo{}, nil, ErrNavUnreliable
	}
	if !e.SequencerUp {
		return SwapInfo{}, nil, ErrSequencerDown
	}
	if amountIn.Lt(e.MinDeposit) {
		return SwapInfo{}, nil, ErrDepositTooSmall
	}

	var accrued uint256.Int
	if e.LastAccrual != 0 && e.Timestamp > e.LastAccrual && !e.Supply.IsZero() && e.ManagementFeeBps != 0 {
		var rate, denom uint256.Int
		rate.Mul(uint256.NewInt(e.ManagementFeeBps), uint256.NewInt(e.Timestamp-e.LastAccrual))
		denom.Mul(bps, year)
		big256.MulDivDown(&accrued, e.Supply, &rate, &denom)
	}

	var supply, assets uint256.Int
	supply.Add(e.Supply, &accrued)
	supply.Add(&supply, virtualShares)
	assets.Add(e.NavEth, virtualAssets)

	var protocolFee, stabilityFee, fee, net uint256.Int
	big256.MulDivDown(&protocolFee, amountIn, uint256.NewInt(e.ProtocolFeeBps), bps)
	big256.MulDivDown(&stabilityFee, amountIn, uint256.NewInt(e.StabilityFeeBps), bps)
	fee.Add(&protocolFee, &stabilityFee)
	if fee.Gt(amountIn) {
		return SwapInfo{}, nil, ErrFeeExceedsAmountIn
	}
	net.Sub(amountIn, &fee)

	out, feeShares := new(uint256.Int), new(uint256.Int)
	big256.MulDivDown(out, &net, &supply, &assets)
	big256.MulDivDown(feeShares, &protocolFee, &supply, &assets)
	if out.IsZero() {
		return SwapInfo{}, nil, ErrZeroAmountOut
	}

	return SwapInfo{
		AccruedShares: new(uint256.Int).Set(&accrued),
		SharesOut:     out,
		FeeShares:     feeShares,
	}, new(uint256.Int).Set(&fee), nil
}

// UpdateBalance applies a mint to the tracked state: the accrued management fee, the minted shares and the fee
// shares join the supply, and the whole deposit joins the NAV (the stability fee stays in the vault, the protocol
// fee is paid in shares).
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	info, ok := params.SwapInfo.(SwapInfo)
	if !ok || info.SharesOut == nil || params.TokenAmountIn.Amount == nil {
		return
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return
	}

	e := &p.extra
	supply := new(uint256.Int).Add(e.Supply, info.AccruedShares)
	supply.Add(supply, info.SharesOut)
	supply.Add(supply, info.FeeShares)
	e.Supply = supply
	e.NavEth = new(uint256.Int).Add(e.NavEth, amountIn)
	e.LastAccrual = e.Timestamp

	p.Info.Reserves = []*big.Int{e.NavEth.ToBig(), e.Supply.ToBig()}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = make([]*big.Int, len(p.Info.Reserves))
	for i, r := range p.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}
	return &cloned
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	if p.GetTokenIndex(address) == 1 {
		return []string{p.Info.Tokens[0]}
	}
	return []string{}
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	if p.GetTokenIndex(address) == 0 {
		return []string{p.Info.Tokens[1]}
	}
	return []string{}
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{BlockNumber: p.Info.BlockNumber}
}

// GetApprovalAddress returns the vault: buyGBLINWithWeth pulls the WETH from the caller.
func (p *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return p.Info.Address
}
