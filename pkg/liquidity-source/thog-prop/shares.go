package thogprop

import "github.com/holiman/uint256"

// ShareBalance is the pool's holding of a share-based token (an xStock): the
// token's sharesOf(pool) and the pool's storedMultiplier(token), read in the same
// state as getPoolData. The pool values shares at its stored multiplier, which can
// lag the issuer's own. Ordinary tokens have an empty entry.
type ShareBalance struct {
	Shares     uint256.Int `json:"shares"`
	Multiplier uint256.Int `json:"multiplier"`
}

// shareBased reports whether the pool holds this token as shares. The pool stores
// a multiplier only for such tokens, so zero identifies an ordinary ERC-20.
func (s *State) shareBased(index int) bool { return !s.Shares[index].Multiplier.IsZero() }

func saturatingSub(a, b uint256.Int) uint256.Int {
	if a.Lt(&b) {
		return uint256.Int{}
	}
	a.Sub(&a, &b)
	return a
}

func (s *State) physicalUnits(index int) uint256.Int {
	if s.shareBased(index) {
		return s.Shares[index].Shares
	}
	return s.Balances[index]
}

func (s *State) amountForUnits(m *arithmetic, index int, units uint256.Int) uint256.Int {
	if s.shareBased(index) && s.Shares[index].Multiplier != powers[18] {
		return m.md(units, s.Shares[index].Multiplier, powers[18], false)
	}
	return units
}

func (s *State) transferUnits(m *arithmetic, index int, amount uint256.Int) uint256.Int {
	if s.shareBased(index) && s.Shares[index].Multiplier != powers[18] {
		return m.md(amount, powers[18], s.Shares[index].Multiplier, false)
	}
	return amount
}

func (s *State) transferDust(m *arithmetic, index int) uint256.Int {
	if s.shareBased(index) {
		// max(2, ceil(multiplier / 1e18)) is exactly two throughout this range.
		if s.Shares[index].Multiplier.CmpUint64(2_000_000_000_000_000_000) <= 0 {
			return u(2)
		}
		return max(u(2), m.md(s.Shares[index].Multiplier, u(1), powers[18], true))
	}
	return uint256.Int{}
}

// A swap reconciles physical holdings at its two endpoints; other tokens use
// the accounted inventory from the logical page, including issuer shares.
func (s *State) pairBalances(m *arithmetic, input, output int) (balances [maxTokens]uint256.Int) {
	in, out := s.physicalUnits(input), s.physicalUnits(output)
	if in.Gt(&maxInventory) || out.Gt(&maxInventory) {
		m.err = ErrBalance
		return
	}
	for i := range s.Tokens {
		if i == input || i == output {
			balances[i] = s.Balances[i]
		} else {
			balances[i] = s.amountForUnits(m, i, s.field(inventoryField(i)))
		}
	}
	return
}

// settlement reproduces floor-to-shares transfers, including the contract's
// output dust. The returned units also become the endpoints' accounted inventory.
func (s *State) settlement(c *curve, input, output uint256.Int) (swap SwapInfo, err error) {
	m := arithmetic{}
	if !output.IsZero() {
		output = m.add(output, c.outputDust)
	}
	swap.InputIndex, swap.OutputIndex = c.input, c.output
	swap.InputUnits = m.add(s.physicalUnits(c.input), s.transferUnits(&m, c.input, input))
	swap.OutputUnits = m.sub(s.physicalUnits(c.output), s.transferUnits(&m, c.output, output))
	if swap.InputUnits.Gt(&maxInventory) || swap.OutputUnits.Gt(&maxInventory) {
		return swap, ErrBalance
	}
	swap.InputBalance = s.amountForUnits(&m, c.input, swap.InputUnits)
	swap.OutputBalance = s.amountForUnits(&m, c.output, swap.OutputUnits)
	return swap, m.err
}

func (s *State) setAccountedUnits(index int, units uint256.Int) {
	f := inventoryField(index)
	var mask, value uint256.Int
	mask.Lsh(&maxInventory, uint(f.offset))
	mask.Not(&mask)
	value.Lsh(&units, uint(f.offset))
	s.Words[f.slot].And(&s.Words[f.slot], &mask)
	s.Words[f.slot].Or(&s.Words[f.slot], &value)
}
