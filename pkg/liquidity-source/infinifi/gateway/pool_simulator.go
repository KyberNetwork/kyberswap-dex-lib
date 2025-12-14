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

	isPaused         bool
	iusdSupply       *big.Int
	siusdTotalAssets *big.Int
	siusdSupply      *big.Int
	liusdSupplies    []*big.Int

	// Token addresses for quick lookup
	usdc        string
	iusd        string
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
	for i, supply := range extra.LIUSDSupplies {
		liusdSupplies[i] = bignumber.NewBig(supply)
	}

	// Identify token positions
	// tokens[0] = USDC
	// tokens[1] = iUSD
	// tokens[2] = siUSD
	// tokens[3+] = liUSD tokens
	liusdTokenAddrs := make([]string, 0)
	if len(tokens) > 3 {
		liusdTokenAddrs = tokens[3:]
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
		isPaused:         extra.IsPaused,
		iusdSupply:       extra.IUSDSupply,
		siusdTotalAssets: extra.SIUSDTotalAssets,
		siusdSupply:      extra.SIUSDSupply,
		liusdSupplies:    liusdSupplies,
		usdc:             tokens[0],
		iusd:             tokens[1],
		siusd:            tokens[2],
		liusdTokens:      liusdTokenAddrs,
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
	case tokenIn == s.usdc && tokenOut == s.iusd:
		// USDC → iUSD (mint) - 1:1 conversion
		amountOut = s.calculateMint(amountIn)
		gas = defaultMintGas

	case tokenIn == s.iusd && tokenOut == s.siusd:
		// iUSD → siUSD (stake) - ERC4626 conversion
		amountOut = s.calculateStake(amountIn)
		gas = defaultStakeGas

	case tokenIn == s.iusd && s.isLIUSD(tokenOut):
		// iUSD → liUSD (lock) - 1:1 conversion
		amountOut = s.calculateLock(amountIn)
		gas = defaultLockGas

	default:
		// Check for reverse paths (all async, not supported)
		if tokenIn == s.iusd && tokenOut == s.usdc {
			return nil, ErrAsyncRedemption
		}
		if tokenIn == s.siusd && tokenOut == s.iusd {
			return nil, ErrAsyncRedemption
		}
		if s.isLIUSD(tokenIn) && tokenOut == s.iusd {
			return nil, ErrAsyncRedemption
		}
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

// calculateLock: iUSD → liUSD (1:1 via LockingController)
func (s *PoolSimulator) calculateLock(iusdAmount *big.Int) *big.Int {
	// LockingController provides 1:1 conversion
	// liUSD shares = iUSD amount (both 18 decimals)
	return new(big.Int).Set(iusdAmount)
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
	case tokenIn == s.usdc && tokenOut == s.iusd:
		// USDC → iUSD (mint)
		// iUSD supply increases
		s.iusdSupply = new(big.Int).Add(s.iusdSupply, amountOut)

	case tokenIn == s.iusd && tokenOut == s.siusd:
		// iUSD → siUSD (stake)
		// iUSD moves into siUSD vault, siUSD shares increase
		s.siusdTotalAssets = new(big.Int).Add(s.siusdTotalAssets, amountIn)
		s.siusdSupply = new(big.Int).Add(s.siusdSupply, amountOut)

	case tokenIn == s.iusd && s.isLIUSD(tokenOut):
		// iUSD → liUSD (lock)
		// Find which liUSD token and update its supply
		for i, liusdAddr := range s.liusdTokens {
			if tokenOut == liusdAddr {
				s.liusdSupplies[i] = new(big.Int).Add(s.liusdSupplies[i], amountOut)
				break
			}
		}
	}

	// Update reserves for display
	s.Info.Reserves[0] = new(big.Int).Set(s.iusdSupply)
	s.Info.Reserves[1] = new(big.Int).Set(s.siusdTotalAssets)
	s.Info.Reserves[2] = new(big.Int).Set(s.siusdSupply)
	for i, supply := range s.liusdSupplies {
		if i+3 < len(s.Info.Reserves) {
			s.Info.Reserves[i+3] = new(big.Int).Set(supply)
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

	// Deep copy liUSD supplies
	cloned.liusdSupplies = make([]*big.Int, len(s.liusdSupplies))
	for i, supply := range s.liusdSupplies {
		cloned.liusdSupplies[i] = new(big.Int).Set(supply)
	}

	// Clone reserves
	cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}

	return &cloned
}
