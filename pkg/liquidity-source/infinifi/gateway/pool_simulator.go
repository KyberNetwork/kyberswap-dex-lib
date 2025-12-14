package gateway

import (
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	isPaused           bool
	iusdSupply         *big.Int
	siusdTotalAssets   *big.Int
	siusdSupply        *big.Int
	liusdSupplies      []*big.Int
	liusdTotalReceipts []*big.Int

	// Token addresses for quick lookup
	usdc        string
	siusd       string
	liusdTokens []string
}

// Register the pool simulator factory
var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, len(entityPool.Tokens))
	reserves := make([]*big.Int, len(entityPool.Reserves))
	for i, token := range entityPool.Tokens {
		tokens[i] = token.Address
	}
	for i, reserve := range entityPool.Reserves {
		reserves[i] = bignumber.NewBig(reserve)
	}

	// Convert liUSD supply strings to big.Int
	liusdSupplies := make([]*big.Int, len(extra.LIUSDSupplies))
	liusdTotalReceipts := make([]*big.Int, len(extra.LIUSDTotalReceipts))
	for i := range extra.LIUSDSupplies {
		liusdSupplies[i] = bignumber.NewBig(extra.LIUSDSupplies[i])
		liusdTotalReceipts[i] = bignumber.NewBig(extra.LIUSDTotalReceipts[i])
	}

	// Identify token positions
	// tokens[0] = USDC
	// tokens[1] = siUSD
	// tokens[2+] = liUSD tokens
	liusdTokenAddrs := make([]string, 0)
	if len(tokens) > 2 {
		liusdTokenAddrs = tokens[2:]
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     entityPool.Address,
				Exchange:    entityPool.Exchange,
				Type:        entityPool.Type,
				Tokens:      tokens,
				Reserves:    reserves,
				BlockNumber: entityPool.BlockNumber,
			},
		},
		isPaused:           extra.IsPaused,
		iusdSupply:         extra.IUSDSupply,
		siusdTotalAssets:   extra.SIUSDTotalAssets,
		siusdSupply:        extra.SIUSDSupply,
		liusdSupplies:      liusdSupplies,
		liusdTotalReceipts: liusdTotalReceipts,
		usdc:               tokens[0],
		siusd:              tokens[1],
		liusdTokens:        liusdTokenAddrs,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := strings.ToLower(params.TokenAmountIn.Token)
	tokenOut := strings.ToLower(params.TokenOut)
	amountIn := params.TokenAmountIn.Amount

	// Check if contract is paused
	if s.isPaused {
		return nil, ErrContractPaused
	}

	// Determine swap type and calculate output
	var amountOut *big.Int
	var gas int64

	switch {
	case tokenIn == s.usdc && tokenOut == s.siusd:
		// USDC → siUSD (mintAndStake) - combined operation
		iusdAmount := s.calculateMint(amountIn)
		amountOut = s.calculateStake(iusdAmount)
		gas = defaultMintAndStakeGas

	case tokenIn == s.usdc && s.isLIUSD(tokenOut):
		// USDC → liUSD (mintAndLock) - combined operation
		iusdAmount := s.calculateMint(amountIn)
		// Find which liUSD bucket this is for
		bucketIndex := s.getLIUSDIndex(tokenOut)
		amountOut = s.calculateLock(iusdAmount, bucketIndex)
		gas = defaultMintAndLockGas

	default:
		// All other paths are unsupported (reverse paths are async)
		return nil, ErrSwapNotSupported
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: big.NewInt(0), // No fees on these operations
		},
		Gas: gas,
	}, nil
}

// calculateMint: USDC → iUSD (1:1 via MintController)
func (s *PoolSimulator) calculateMint(usdcAmount *big.Int) *big.Int {
	// MintController provides 1:1 conversion
	// USDC has 6 decimals, iUSD has 18 decimals
	// Need to scale: iUSD = USDC * 10^12
	iusdAmount := new(big.Int).Mul(usdcAmount, bignumber.TenPowInt(12))
	return iusdAmount
}

// calculateStake: iUSD → siUSD (ERC4626 share conversion)
func (s *PoolSimulator) calculateStake(iusdAmount *big.Int) *big.Int {
	// ERC4626 formula: shares = assets * totalShares / totalAssets
	// If totalAssets == 0 (first deposit), shares = assets (1:1)
	if s.siusdTotalAssets.Sign() == 0 || s.siusdSupply.Sign() == 0 {
		return new(big.Int).Set(iusdAmount)
	}

	// shares = iusdAmount * siusdSupply / siusdTotalAssets
	siusdShares := new(big.Int).Mul(iusdAmount, s.siusdSupply)
	siusdShares.Div(siusdShares, s.siusdTotalAssets)

	return siusdShares
}

// calculateLock: iUSD → liUSD (share-based conversion via LockingController)
func (s *PoolSimulator) calculateLock(iusdAmount *big.Int, bucketIndex int) *big.Int {
	// LockingController.createPosition (line 216):
	// newShares = totalShares == 0 ? amount : amount.mulDivDown(totalShares, data.totalReceiptTokens)

	totalShares := s.liusdSupplies[bucketIndex]
	totalReceiptTokens := s.liusdTotalReceipts[bucketIndex]

	// If first deposit (no shares yet), 1:1 conversion
	if totalShares.Sign() == 0 {
		return new(big.Int).Set(iusdAmount)
	}

	// Otherwise: shares = iusdAmount * totalShares / totalReceiptTokens
	liusdShares := new(big.Int).Mul(iusdAmount, totalShares)
	liusdShares.Div(liusdShares, totalReceiptTokens)

	return liusdShares
}

// getLIUSDIndex finds the index of a liUSD token in the liusdTokens array
func (s *PoolSimulator) getLIUSDIndex(tokenAddr string) int {
	for i, addr := range s.liusdTokens {
		if addr == tokenAddr {
			return i
		}
	}
	return -1
}

// isLIUSD checks if a token address is one of the liUSD tokens
func (s *PoolSimulator) isLIUSD(tokenAddr string) bool {
	return lo.Contains(s.liusdTokens, tokenAddr)
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn := strings.ToLower(params.TokenAmountIn.Token)
	tokenOut := strings.ToLower(params.TokenAmountOut.Token)
	amountIn := params.TokenAmountIn.Amount
	amountOut := params.TokenAmountOut.Amount

	// Update state based on swap type
	switch {
	case tokenIn == s.usdc && tokenOut == s.siusd:
		// USDC → siUSD (mintAndStake)
		// iUSD is minted then immediately staked
		iusdAmount := s.calculateMint(amountIn)
		s.iusdSupply = new(big.Int).Add(s.iusdSupply, iusdAmount)
		s.siusdTotalAssets = new(big.Int).Add(s.siusdTotalAssets, iusdAmount)
		s.siusdSupply = new(big.Int).Add(s.siusdSupply, amountOut)

	case tokenIn == s.usdc && s.isLIUSD(tokenOut):
		// USDC → liUSD (mintAndLock)
		// iUSD is minted then immediately locked
		iusdAmount := s.calculateMint(amountIn)
		s.iusdSupply = new(big.Int).Add(s.iusdSupply, iusdAmount)
		for i, liusdAddr := range s.liusdTokens {
			if tokenOut == liusdAddr {
				s.liusdSupplies[i] = new(big.Int).Add(s.liusdSupplies[i], amountOut)
				break
			}
		}
	}

	// Update reserves for display
	s.Info.Reserves[0] = new(big.Int).Set(s.siusdTotalAssets)
	s.Info.Reserves[1] = new(big.Int).Set(s.siusdSupply)
	for i, supply := range s.liusdSupplies {
		if i+2 < len(s.Info.Reserves) {
			s.Info.Reserves[i+2] = new(big.Int).Set(supply)
		}
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return Meta{
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s

	// Deep copy big.Int fields
	cloned.iusdSupply = new(big.Int).Set(s.iusdSupply)
	cloned.siusdTotalAssets = new(big.Int).Set(s.siusdTotalAssets)
	cloned.siusdSupply = new(big.Int).Set(s.siusdSupply)

	// Deep copy liUSD supplies and total receipts
	cloned.liusdSupplies = make([]*big.Int, len(s.liusdSupplies))
	cloned.liusdTotalReceipts = make([]*big.Int, len(s.liusdTotalReceipts))
	for i := range s.liusdSupplies {
		cloned.liusdSupplies[i] = new(big.Int).Set(s.liusdSupplies[i])
		cloned.liusdTotalReceipts[i] = new(big.Int).Set(s.liusdTotalReceipts[i])
	}

	// Clone reserves
	cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}

	return &cloned
}
