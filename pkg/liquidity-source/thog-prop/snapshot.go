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

// snapshotState builds the quote state from one getPoolData page and the pool's
// share-based holdings, keyed by lowercase token address. A reader must have asked
// the pool for every listed token's stored multiplier; tokens absent from shares
// are ordinary ERC-20s.
func snapshotState(p poolData, shares map[string]ShareBalance) (s State, err error) {
	n := int(p.TokenCount)
	categories := int(p.CategoryCount)
	if p.SchemaVersion != schemaVersion || n < 1 || n > maxTokens || categories < 1 || categories > n || p.StartTokenIndex != 0 || len(p.Tokens) != n || len(p.Balances) != n || len(p.PortfolioBalances) != n || len(p.Categories) != categories || len(p.Words) != pageWords || p.BlockNumber == nil || !p.BlockNumber.IsUint64() {
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
		if int(w.Slot) != i+firstPageSlot {
			return s, ErrInvalidState
		}
		s.Words[w.Slot] = convert(w.Value)
	}
	// The page describes where each token's fields are stored. Accept it only if
	// that matches the layout this adapter decodes.
	for i, t := range p.Tokens {
		category := int(t.Category)
		if category >= categories || t.Token == (common.Address{}) {
			return s, ErrInvalidState
		}
		pairStart := uint32(0)
		if i > 0 {
			pairStart = pairSpreadField(pairIndex(0, i)).ref()
		}
		if t.Price != priceField(i).ref() || t.PriceTail != 0 || t.TokenBidSpread != tokenField(i, tokenBidSpread).ref() || t.TokenAskSpread != tokenField(i, tokenAskSpread).ref() || t.VolumeRate != tokenField(i, tokenVolumeRate).ref() || t.CategorySpread != categoryField(category, categorySpread).ref() || t.Inventory != categoryField(category, categoryOffset).ref() || t.Sides != controlSides(i).ref() || t.PostedBlock != controlBlock.ref() || t.Sequence != controlSequence.ref() || t.PairStart != pairStart || t.InventoryCap != tokenCap(i).ref() {
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
	// Every holding belongs to a listed token and carries the nonzero multiplier
	// that marks it share-based.
	held := 0
	for i := range s.Tokens {
		if s.shareBased(i) {
			held++
		}
	}
	if len(shares) != held {
		return s, ErrInvalidState
	}
	if err != nil {
		return s, err
	}
	if err = s.validate(); err != nil {
		return s, err
	}
	for i, c := range p.Categories {
		anchor := s.small(categoryField(i, categoryAnchor))
		t := p.Tokens[anchor]
		if uint64(c.AnchorTokenIndex) != anchor || c.Price != t.Price || c.PriceTail != 0 || c.Inventory != t.Inventory || c.Spread != t.CategorySpread || c.PostedBlock != t.PostedBlock || c.InventoryCap != categoryCap(i).ref() {
			return s, ErrInvalidState
		}
	}
	return s, nil
}
