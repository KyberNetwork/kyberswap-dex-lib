package flywheelfun

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Additional settlement, fee-vault and reward-accounting work per native parent.
// Covers measured cold-account mainnet-fork execution, beyond the CL swap gas.
const nativeParentGas int64 = 750000

func validateParents(token, quote string, parents []NativeParent) (string, error) {
	if len(parents) > 2 {
		return "", ErrUnsupported
	}
	current := quote
	seen := map[string]bool{token: true}
	pools := map[string]bool{}
	for _, p := range parents {
		if !common.IsHexAddress(p.Token) || !common.IsHexAddress(p.Quote) || p.Token != current || seen[p.Token] || p.Protocol[0] > 1000 || p.Protocol[1] > 1000 || pools[p.Pool.Address] {
			return "", ErrState
		}
		seen[p.Token] = true
		pools[p.Pool.Address] = true
		current = p.Quote
	}
	if seen[current] {
		return "", ErrState
	}
	return current, nil
}
func nativeCore(base pool.IPoolSimulator, expected entity.Pool, block uint64) (*v3.PoolSimulator, error) {
	c, ok := base.(*v3.PoolSimulator)
	if !ok || c.GetAddress() != expected.Address || c.Info.BlockNumber != block || expected.BlockNumber != block || len(c.Info.Tokens) != 2 || len(expected.Tokens) != 2 {
		return nil, ErrState
	}
	for i, t := range expected.Tokens {
		if t == nil || c.Info.Tokens[i] != t.Address {
			return nil, ErrState
		}
	}
	return c, nil
}
func (s *PoolSimulator) bindParents(baseMap map[string]pool.IPoolSimulator) error {
	for _, p := range s.Parents {
		if p.Pool.BlockNumber != s.Info.BlockNumber {
			return ErrState
		}
		base := baseMap[p.Pool.Address]
		if base == nil {
			var err error
			base, err = v3.NewPoolSimulator(p.Pool, 4663)
			if err != nil {
				return err
			}
		}
		c, err := nativeCore(base, p.Pool, s.Info.BlockNumber)
		if err != nil {
			return err
		}
		if c.GetTokenIndex(p.Token) < 0 || c.GetTokenIndex(p.Quote) < 0 {
			return ErrState
		}
		s.ParentPools = append(s.ParentPools, base)
	}
	return nil
}
func (s *PoolSimulator) baseQuote() string {
	if len(s.Parents) > 0 {
		return s.Parents[len(s.Parents)-1].Quote
	}
	return s.Static.Quote
}
func (s *PoolSimulator) parentTokens() []string {
	out := make([]string, len(s.Parents))
	for i, p := range s.Parents {
		out[i] = p.Token
	}
	return out
}

// All legs use the same cloned pool objects, including the reverse refund.
// Parent platform allocations are taken before external routing, in the same
// nearest-parent-first order as settlement's frames; quote allocations occur
// in reverse order as each parent is bought.
func (s *PoolSimulator) routeSwap(amount *uint256.Int, buy bool, bases []pool.IPoolSimulator) (uint256.Int, []SwapStep, int64, error) {
	if len(s.Parents) == 0 {
		return s.externalSwap(amount, buy, bases)
	}
	result := *amount
	var steps []SwapStep
	var gas int64
	platforms := make([]uint256.Int, len(s.Parents))
	inputs := make([]uint256.Int, len(s.Parents))
	gross := make([]uint256.Int, len(s.Parents))
	protocols := make([]uint32, len(s.Parents))
	if buy {
		for i, p := range s.Parents {
			c, err := core(bases[1+i])
			if err != nil {
				return result, nil, 0, err
			}
			protocols[i] = p.Protocol[c.GetTokenIndex(p.Quote)]
			inputs[i] = result
			platforms[i], err = BuyPlatform(&result, protocols[i])
			if err != nil {
				return result, nil, 0, err
			}
			result.Sub(&result, &platforms[i])
		}
	} else {
		for i, p := range s.Parents {
			index := 1 + i
			c, err := core(bases[index])
			if err != nil {
				return result, nil, 0, err
			}
			protocols[i] = p.Protocol[c.GetTokenIndex(p.Token)]
			out, step, g, err := swapCL(bases[index], &result, p.Token, p.Quote, protocols[i], index)
			if err != nil {
				return result, nil, 0, err
			}
			gross[i] = out
			split, err := SellQuote(&out, protocols[i])
			if err != nil {
				return result, nil, 0, err
			}
			result = split.Net
			steps = append(steps, step)
			gas += g + nativeParentGas
		}
	}
	out, external, g, err := s.externalSwap(&result, buy, bases)
	if err != nil {
		return result, nil, 0, err
	}
	result = out
	steps = append(steps, external...)
	gas += g
	for i := len(s.Parents) - 1; i >= 0; i-- {
		p := s.Parents[i]
		if buy {
			bought := result
			split, er := BuyQuote(&bought, protocols[i])
			if er != nil {
				return result, nil, 0, er
			}
			index := 1 + i
			output, step, g, er := swapCL(bases[index], &split.Net, p.Quote, p.Token, protocols[i], index)
			if er != nil {
				return result, nil, 0, er
			}
			result = output
			var quoteFee, routing uint256.Int
			quoteFee.Sub(&bought, &split.Net)
			routing.Sub(&inputs[i], &platforms[i])
			equivalent, er := mulDiv(&routing, &quoteFee, &bought, false)
			if er != nil {
				return result, nil, 0, er
			}
			if _, bad := step.NativeFeeETH.AddOverflow(&platforms[i], &equivalent); bad {
				return result, nil, 0, ErrMath
			}
			steps = append(steps, step)
			gas += g + nativeParentGas
		} else {
			reserves, er := SellQuote(&gross[i], protocols[i])
			if er != nil {
				return result, nil, 0, er
			}
			split, er := SellETH(&result, &gross[i], protocols[i])
			if er != nil {
				return result, nil, 0, er
			}
			value, er := mulDiv(&result, &gross[i], &reserves.Net, false)
			if er != nil {
				return result, nil, 0, er
			}
			if _, bad := steps[i].NativeFeeETH.SubOverflow(&value, &split.Net); bad {
				return result, nil, 0, ErrMath
			}
			result = split.Net
		}
	}
	return result, steps, gas, nil
}
