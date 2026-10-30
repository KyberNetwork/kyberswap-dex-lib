package thogprop

import "github.com/holiman/uint256"

// ShareBalance is issuer state captured at the same block as getPoolData.
// Ordinary tokens have an empty entry; xStocks require a nonzero multiplier.
type ShareBalance struct {
	Shares     uint256.Int `json:"shares"`
	Multiplier uint256.Int `json:"multiplier"`
}

// These are the nine issuer tokens recognized by the deployed contract's catalog.
// Active listings and their ordering still come from the snapshot.
func isXStock(token string) bool {
	switch token {
	case "0x90a2a4c76b5d8c0bc892a69ea28aa775a8f2dd48", // SPYx
		"0xa753a7395cae905cd615da0b82a53e0560f250af", // QQQx
		"0x8d512a03600981b8e86556fd5c0fbdd726fe1821", // SMHx
		"0x299c95eb52bfd5bc98d924bce17197b64864589b", // EWYx
		"0xc845b2894dbddd03858fd2d643b4ef725fe0849d", // NVDAx
		"0x8ad3c73f833d3f9a523ab01476625f269aeb7cf0", // TSLAx
		"0x68fa48b1c2fe52b3d776e1953e0e782b5044ce28", // SPCXx
		"0xf6a873bae4ba1b304e45df52a4b7d176e1c6a8c4", // MUx
		"0x58100046a4afcd4ee4fadbd4244f3f895a341c56": // SKHYx
		return true
	}
	return false
}

func saturatingSub(a, b uint256.Int) uint256.Int {
	if a.Lt(&b) {
		return uint256.Int{}
	}
	a.Sub(&a, &b)
	return a
}

func (s *State) physicalUnits(index int) uint256.Int {
	if isXStock(s.Tokens[index].Address) {
		return s.Shares[index].Shares
	}
	return s.Balances[index]
}

func (s *State) amountForUnits(m *arithmetic, index int, units uint256.Int) uint256.Int {
	if isXStock(s.Tokens[index].Address) {
		return m.md(units, s.Shares[index].Multiplier, powers[18], false)
	}
	return units
}

func (s *State) transferUnits(m *arithmetic, index int, amount uint256.Int) uint256.Int {
	if isXStock(s.Tokens[index].Address) {
		return m.md(amount, powers[18], s.Shares[index].Multiplier, false)
	}
	return amount
}

func (s *State) transferDust(m *arithmetic, index int) uint256.Int {
	if isXStock(s.Tokens[index].Address) {
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
			balances[i] = s.amountForUnits(m, i, s.packed(107, i, 128, 2))
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
	slot, shift := 107+index/2, uint(index%2*128)
	var mask, value uint256.Int
	mask.Lsh(&maxInventory, shift)
	mask.Not(&mask)
	value.Lsh(&units, shift)
	s.Words[slot].And(&s.Words[slot], &mask)
	s.Words[slot].Or(&s.Words[slot], &value)
}
