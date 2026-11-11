package thogprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

type tokenMetadata struct {
	Token          common.Address
	Decimals       uint8
	Category       uint8
	Price          uint32
	PriceTail      uint32
	TokenBidSpread uint32
	TokenAskSpread uint32
	VolumeRate     uint32
	CategorySpread uint32
	Inventory      uint32
	Sides          uint32
	PostedBlock    uint32
	Sequence       uint32
	PairStart      uint32
	InventoryCap   uint32
}
type categoryMetadata struct {
	AnchorTokenIndex uint8
	Price            uint32
	PriceTail        uint32
	Inventory        uint32
	Spread           uint32
	PostedBlock      uint32
	InventoryCap     uint32
}
type storageWord struct {
	Slot  uint16
	Value *big.Int
}
type poolData struct {
	SchemaVersion     uint16
	BlockNumber       *big.Int
	BaseFeePerGas     *big.Int
	ParentBlockHash   [32]byte
	TokenCount        uint8
	CategoryCount     uint8
	StartTokenIndex   uint8
	Tokens            []tokenMetadata
	Balances          []*big.Int
	Categories        []categoryMetadata
	Words             []storageWord
	NativeBalance     *big.Int
	PortfolioBalances []*big.Int
}

func decodeSnapshot(data []byte) (poolData, error) {
	var out struct{ Data poolData }
	err := thogAMMABI.UnpackIntoInterface(&out, methodGetPoolData, data)
	return out.Data, err
}
func snapshotState(p poolData, shares map[string]ShareBalance) (s State, err error) {
	n := int(p.TokenCount)
	categories := int(p.CategoryCount)
	if p.SchemaVersion != 8 || n < 1 || n > maxTokens || categories < 1 || categories > n || p.StartTokenIndex != 0 || len(p.Tokens) != n || len(p.Balances) != n || len(p.PortfolioBalances) != n || len(p.Categories) != categories || len(p.Words) != 126 || p.BlockNumber == nil || !p.BlockNumber.IsUint64() {
		return s, ErrInvalidState
	}
	convert := func(n *big.Int) uint256.Int {
		if n == nil || n.Sign() < 0 {
			err = ErrInvalidState
			return uint256.Int{}
		}
		v, overflow := uint256.FromBig(n)
		if overflow {
			err = ErrOverflow
			return uint256.Int{}
		}
		return *v
	}
	s.BlockNumber = p.BlockNumber.Uint64()
	s.BaseFee = convert(p.BaseFeePerGas)
	s.NativeBalance = convert(p.NativeBalance)
	s.CategoryCount = categories
	s.Tokens = make([]Token, n)
	for i, w := range p.Words {
		if int(w.Slot) != i+2 {
			return s, ErrInvalidState
		}
		s.Words[w.Slot] = convert(w.Value)
	}
	ref := func(slot, offset, width int) uint32 { return uint32(slot<<16 | offset<<8 | width) }
	for i, t := range p.Tokens {
		category := int(t.Category)
		if category >= categories || t.Token == (common.Address{}) {
			return s, ErrInvalidState
		}
		ts, to := 9+i/3, i%3*75
		cs, co := 27+category/3, category%3*85
		pairStart := uint32(0)
		if i > 0 {
			index := i * (i - 1) / 2
			pairStart = ref(63+index/16, index%16*16, 16)
		}
		if t.Price != ref(20+i/4, i%4*64, 64) || t.PriceTail != 0 || t.TokenBidSpread != ref(ts, to+15, 16) || t.TokenAskSpread != ref(ts, to+43, 16) || t.VolumeRate != ref(ts, to+59, 16) || t.CategorySpread != ref(cs, co+64, 16) || t.Inventory != ref(cs, co, 64) || t.Sides != ref(2, 68+i*2, 2) || t.PostedBlock != ref(2, 36, 32) || t.Sequence != ref(2, 20, 16) || t.PairStart != pairStart || t.InventoryCap != ref(101+i/10, i%10*24, 24) {
			return s, ErrInvalidState
		}
		s.Tokens[i] = Token{Address: hexutil.Encode(t.Token[:]), Decimals: t.Decimals, Category: t.Category}
		s.Shares[i] = shares[s.Tokens[i].Address]
		s.Balances[i] = convert(p.Balances[i])
		portfolio := convert(p.PortfolioBalances[i])
		if s.Balances[i] != portfolio {
			return s, ErrInvalidState
		}
	}
	expectedShares := 0
	for _, t := range s.Tokens {
		if isXStock(t.Address) {
			expectedShares++
		}
	}
	if len(shares) != expectedShares {
		return s, ErrInvalidState
	}
	if err != nil {
		return s, err
	}
	if err = s.validate(); err != nil {
		return s, err
	}
	for i, c := range p.Categories {
		anchor := s.small(27+i/3, i%3*85+80, 5)
		t := p.Tokens[anchor]
		if uint64(c.AnchorTokenIndex) != anchor || c.Price != t.Price || c.PriceTail != 0 || c.Inventory != t.Inventory || c.Spread != t.CategorySpread || c.PostedBlock != t.PostedBlock || c.InventoryCap != ref(104+i/10, i%10*24, 24) {
			return s, ErrInvalidState
		}
	}
	return s, nil
}
