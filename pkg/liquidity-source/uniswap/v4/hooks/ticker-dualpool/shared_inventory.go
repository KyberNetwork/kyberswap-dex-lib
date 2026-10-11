package tickerdualpool

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const snapshotBlockKey = "ticker:143:block"

// Pool shares/raw inventory belong to one pool. Wrapper totals, reserve totals
// and PoolManager cash can also change when a different pool trades.
func (h *Hook) sharedFields() map[string]**uint256.Int {
	fields := make(map[string]**uint256.Int)
	for side, currency := range h.Currencies {
		if currency == (common.Address{}) {
			continue
		}
		asset := hexutil.Encode(currency[:])
		if h.ManagerCash[side] != nil {
			fields["ticker:143:manager:"+asset] = &h.ManagerCash[side]
		}
		if r := h.Reserves[side]; r != nil {
			prefix := "ticker:143:aave:" + asset + ":"
			fields[prefix+"cash"] = &r.Cash
			fields[prefix+"supply"] = &r.ScaledSupply
			if r.PendingTreasury != nil {
				fields[prefix+"pending"] = &r.PendingTreasury
			}
		}
		if v := h.Aave[side]; v != nil {
			prefix := "ticker:143:vault:" + hexutil.Encode(h.Vaults[side][:]) + ":"
			fields[prefix+"supply"] = &v.Supply
			fields[prefix+"idle"] = &v.Idle
			fields[prefix+"strategyShares"] = &v.StrategyShares
			fields[prefix+"pendingYield"] = &v.PendingYield
		}
	}
	return fields
}
func (h *Hook) CalculateLimit() map[string]*big.Int {
	fields := h.sharedFields()
	if len(fields) == 0 {
		return nil
	}
	result := make(map[string]*big.Int, len(fields)+1)
	result[snapshotBlockKey] = new(big.Int).SetUint64(h.BlockNumber)
	for key, field := range fields {
		if *field != nil {
			result[key] = (*field).ToBig()
		}
	}
	return result
}
func (h *Hook) beforeSwapWithLimit(p *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	local := h.CloneState().(*Hook)
	block := p.Limit.GetLimit(snapshotBlockKey)
	if block == nil || block.Cmp(new(big.Int).SetUint64(h.BlockNumber)) != 0 {
		return nil, ErrInvalidState
	}
	for key, field := range local.sharedFields() {
		value := p.Limit.GetLimit(key)
		if value == nil || value.Sign() < 0 || value.BitLen() > 256 {
			return nil, ErrInvalidState
		}
		*field = uint256.MustFromBig(value)
	}
	for side, v := range local.Aave {
		if v != nil {
			total, err := v.poolAssets()
			if err != nil {
				return nil, err
			}
			if side == 0 {
				local.Total0 = total
			} else {
				local.Total1 = total
			}
		}
		if local.Claims[side] != nil && local.ManagerCash[side] != nil {
			unpaid := unbackedClaims(local.Claims[side], local.ManagerCash[side])
			if side == 0 {
				local.Unbacked0 = unpaid
			} else {
				local.Unbacked1 = unpaid
			}
		}
	}
	params := *p
	params.Limit = nil
	result, err := local.BeforeSwap(&params)
	if err != nil {
		return nil, err
	}
	quote := result.SwapInfo.(Quote)
	quote.SharedBefore = local.CalculateLimit()
	local.UpdateBalance(quote)
	quote.SharedAfter = local.CalculateLimit()
	result.SwapInfo = quote
	return result, nil
}

// Quotes only read the route inventory; execution of the simulated leg updates
// it. Comparing the prestate prevents accidentally applying a stale quote.
func (h *Hook) UpdateBalanceWithLimit(info any, limit pool.SwapLimit) {
	q, ok := info.(Quote)
	if !ok {
		return
	}
	if limit != nil {
		if len(q.SharedBefore) == 0 || len(q.SharedBefore) != len(q.SharedAfter) {
			h.invalidate(ErrInvalidState)
			return
		}
		for key, before := range q.SharedBefore {
			actual := limit.GetLimit(key)
			if actual == nil || actual.Cmp(before) != 0 || q.SharedAfter[key] == nil {
				h.invalidate(ErrInvalidState)
				return
			}
		}
		for key, before := range q.SharedBefore {
			after := q.SharedAfter[key]
			if before.Cmp(after) == 0 {
				continue
			}
			if _, _, err := limit.UpdateLimit(key, key, before, after); err != nil {
				h.invalidate(err)
				return
			}
		}
	}
	h.UpdateBalance(q)
}
