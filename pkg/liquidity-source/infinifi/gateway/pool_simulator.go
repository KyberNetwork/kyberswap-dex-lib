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
	Extra

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
		usdc:        tokens[0],
		iusd:        tokens[1],
		siusd:       tokens[2],
		liusdTokens: liusdTokenAddrs,
		Extra:       extra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := strings.ToLower(params.TokenAmountIn.Token)
	tokenOut := strings.ToLower(params.TokenOut)
	amountIn := params.TokenAmountIn.Amount

	// Check if contract is paused
	if s.IsPaused {
		return nil, ErrContractPaused
	}

	// Determine swap type and calculate output
	var amountOut *big.Int
	var gas int64
	var swapInfo SwapInfo
	switch {
	case tokenIn == s.usdc && tokenOut == s.iusd:
		// USDC → iUSD (mint)
		swapInfo = SwapInfo{
			Action: ActionMint,
		}
		amountOut = s.calculateMint(amountIn)
		gas = defaultMintGas

	case tokenIn == s.iusd && tokenOut == s.usdc:
		// iUSD → USDC (redeem)
		swapInfo = SwapInfo{
			Action: ActionRedeem,
		}
		amountOut = s.calculateRedeem(amountIn)
		gas = defaultRedeemGas

	case tokenIn == s.iusd && tokenOut == s.siusd:
		// iUSD → siUSD (stake)
		swapInfo = SwapInfo{
			Action: ActionStake,
		}
		amountOut = s.calculateStake(amountIn)
		gas = defaultStakeGas

	case tokenIn == s.siusd && tokenOut == s.iusd:
		// siUSD → iUSD (unstake)
		swapInfo = SwapInfo{
			Action: ActionUnstake,
		}
		amountOut = s.calculateUnstake(amountIn)
		gas = defaultUnstakeGas

	case tokenIn == s.iusd && s.isLIUSD(tokenOut):
		// iUSD → liUSD (lock/createPosition)
		bucketIndex := s.getLIUSDIndex(tokenOut)
		swapInfo = SwapInfo{
			Action:          ActionCreatePosition,
			UnwindingEpochs: bucketIndex,
		}
		amountOut = s.calculateLock(amountIn, bucketIndex)
		gas = defaultCreatePositionGas

	case tokenIn == s.usdc && tokenOut == s.siusd:
		// USDC → siUSD (mintAndStake)
		swapInfo = SwapInfo{
			Action: ActionMintAndStake,
		}
		amountOut = s.calculateMintAndStake(amountIn)
		gas = defaultMintAndStakeGas

	// case tokenIn == s.usdc && s.isLIUSD(tokenOut):
	// 	// USDC → liUSD (mintAndLock)
	// 	bucketIndex := s.getLIUSDIndex(tokenOut)
	// 	amountOut = s.calculateMintAndLock(amountIn, bucketIndex)
	// 	gas = defaultMintAndLockGas

	default:
		// All other paths are unsupported
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
		SwapInfo: swapInfo,
		Gas:      gas,
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

// calculateRedeem: iUSD → USDC (1:1 via RedeemController)
func (s *PoolSimulator) calculateRedeem(iusdAmount *big.Int) *big.Int {
	// RedeemController provides 1:1 conversion
	// iUSD has 18 decimals, USDC has 6 decimals
	// Need to scale: USDC = iUSD / 10^12
	usdcAmount := new(big.Int).Div(iusdAmount, bignumber.TenPowInt(12))
	return usdcAmount
}

// calculateStake: iUSD → siUSD (ERC4626 share conversion)
func (s *PoolSimulator) calculateStake(iusdAmount *big.Int) *big.Int {
	// ERC4626 formula: shares = assets * totalShares / totalAssets
	// If totalAssets == 0 (first deposit), shares = assets (1:1)
	if s.SIUSDTotalAssets.Sign() == 0 || s.SIUSDSupply.Sign() == 0 {
		return new(big.Int).Set(iusdAmount)
	}

	// shares = iusdAmount * siusdSupply / siusdTotalAssets
	siusdShares := new(big.Int).Mul(iusdAmount, s.SIUSDSupply)
	siusdShares.Div(siusdShares, s.SIUSDTotalAssets)

	return siusdShares
}

// calculateUnstake: siUSD → iUSD (ERC4626 share redemption)
func (s *PoolSimulator) calculateUnstake(siusdAmount *big.Int) *big.Int {
	// ERC4626 formula: assets = shares * totalAssets / totalShares
	// If totalShares == 0, return 0 (edge case)
	if s.SIUSDSupply.Sign() == 0 {
		return big.NewInt(0)
	}

	// assets = siusdAmount * siusdTotalAssets / siusdSupply
	iusdAmount := new(big.Int).Mul(siusdAmount, s.SIUSDTotalAssets)
	iusdAmount.Div(iusdAmount, s.SIUSDSupply)

	return iusdAmount
}

// calculateLock: iUSD → liUSD (share-based conversion via LockingController)
func (s *PoolSimulator) calculateLock(iusdAmount *big.Int, bucketIndex int) *big.Int {
	// LockingController.createPosition (line 216):
	// newShares = totalShares == 0 ? amount : amount.mulDivDown(totalShares, data.totalReceiptTokens)

	totalShares := s.LIUSDBuckets[bucketIndex].TotalSupply
	totalReceiptTokens := s.LIUSDBuckets[bucketIndex].BucketData.TotalReceiptTokens

	// If first deposit (no shares yet), 1:1 conversion
	if totalShares.Sign() == 0 {
		return new(big.Int).Set(iusdAmount)
	}

	// Otherwise: shares = iusdAmount * totalShares / totalReceiptTokens
	liusdShares := new(big.Int).Mul(iusdAmount, totalShares)
	liusdShares.Div(liusdShares, totalReceiptTokens)

	return liusdShares
}

// calculateMintAndStake: USDC → siUSD (combined mint + stake)
func (s *PoolSimulator) calculateMintAndStake(usdcAmount *big.Int) *big.Int {
	// First: USDC → iUSD (mint with decimal scaling)
	iusdAmount := s.calculateMint(usdcAmount)

	// Second: iUSD → siUSD (stake with ERC4626 conversion)
	siusdAmount := s.calculateStake(iusdAmount)

	return siusdAmount
}

// nolint: unused
// calculateMintAndLock: USDC → liUSD (combined mint + lock)
func (s *PoolSimulator) calculateMintAndLock(usdcAmount *big.Int, bucketIndex int) *big.Int {
	// First: USDC → iUSD (mint with decimal scaling)
	iusdAmount := s.calculateMint(usdcAmount)

	// Second: iUSD → liUSD (lock with bucket conversion)
	liusdAmount := s.calculateLock(iusdAmount, bucketIndex)

	return liusdAmount
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
	case tokenIn == s.usdc && tokenOut == s.iusd:
		// USDC → iUSD (mint)
		s.IUSDSupply = new(big.Int).Add(s.IUSDSupply, amountOut)

	case tokenIn == s.iusd && tokenOut == s.usdc:
		// iUSD → USDC (redeem)
		s.IUSDSupply = new(big.Int).Sub(s.IUSDSupply, amountIn)

	case tokenIn == s.iusd && tokenOut == s.siusd:
		// iUSD → siUSD (stake)
		s.SIUSDTotalAssets = new(big.Int).Add(s.SIUSDTotalAssets, amountIn)
		s.SIUSDSupply = new(big.Int).Add(s.SIUSDSupply, amountOut)

	case tokenIn == s.siusd && tokenOut == s.iusd:
		// siUSD → iUSD (unstake)
		s.SIUSDTotalAssets = new(big.Int).Sub(s.SIUSDTotalAssets, amountOut)
		s.SIUSDSupply = new(big.Int).Sub(s.SIUSDSupply, amountIn)

	case tokenIn == s.iusd && s.isLIUSD(tokenOut):
		// iUSD → liUSD (lock/createPosition)
		for i, liusdAddr := range s.liusdTokens {
			if tokenOut == liusdAddr {
				s.LIUSDBuckets[i].TotalSupply = new(big.Int).Add(s.LIUSDBuckets[i].TotalSupply, amountOut)
				s.LIUSDBuckets[i].BucketData.TotalReceiptTokens = new(big.Int).Add(s.LIUSDBuckets[i].BucketData.TotalReceiptTokens, amountIn)
				break
			}
		}

	case tokenIn == s.usdc && tokenOut == s.siusd:
		// USDC → siUSD (mintAndStake)
		// Calculate intermediate iUSD amount
		iusdAmount := s.calculateMint(amountIn)
		// Update iUSD supply (mint)
		s.IUSDSupply = new(big.Int).Add(s.IUSDSupply, iusdAmount)
		// Update siUSD vault state (stake)
		s.SIUSDTotalAssets = new(big.Int).Add(s.SIUSDTotalAssets, iusdAmount)
		s.SIUSDSupply = new(big.Int).Add(s.SIUSDSupply, amountOut)

	case tokenIn == s.usdc && s.isLIUSD(tokenOut):
		// USDC → liUSD (mintAndLock)
		// Calculate intermediate iUSD amount
		iusdAmount := s.calculateMint(amountIn)
		// Update iUSD supply (mint)
		s.IUSDSupply = new(big.Int).Add(s.IUSDSupply, iusdAmount)
		// Update liUSD bucket state (lock)
		for i, liusdAddr := range s.liusdTokens {
			if tokenOut == liusdAddr {
				s.LIUSDBuckets[i].TotalSupply = new(big.Int).Add(s.LIUSDBuckets[i].TotalSupply, amountOut)
				s.LIUSDBuckets[i].BucketData.TotalReceiptTokens = new(big.Int).Add(s.LIUSDBuckets[i].BucketData.TotalReceiptTokens, iusdAmount)
				break
			}
		}
	}
	// Update reserves for display
	s.Info.Reserves[1] = new(big.Int).Set(s.SIUSDTotalAssets)
	s.Info.Reserves[2] = new(big.Int).Set(s.SIUSDSupply)
	for i, bucket := range s.LIUSDBuckets {
		if i+2 < len(s.Info.Reserves) {
			s.Info.Reserves[i+2] = new(big.Int).Set(bucket.TotalSupply)
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
	cloned.IUSDSupply = new(big.Int).Set(s.IUSDSupply)
	cloned.SIUSDTotalAssets = new(big.Int).Set(s.SIUSDTotalAssets)
	cloned.SIUSDSupply = new(big.Int).Set(s.SIUSDSupply)

	// Deep copy liUSD supplies and total receipts
	cloned.LIUSDBuckets = make([]bucket, len(s.LIUSDBuckets))
	for i := range s.LIUSDBuckets {
		cloned.LIUSDBuckets[i] = bucket{
			Index:       s.LIUSDBuckets[i].Index,
			TotalSupply: new(big.Int).Set(s.LIUSDBuckets[i].TotalSupply),
			BucketData: bucketData{
				ShareToken:         s.LIUSDBuckets[i].BucketData.ShareToken,
				TotalReceiptTokens: new(big.Int).Set(s.LIUSDBuckets[i].BucketData.TotalReceiptTokens),
				Multiplier:         new(big.Int).Set(s.LIUSDBuckets[i].BucketData.Multiplier),
			},
		}
	}
	// Clone reserves
	cloned.Info.Reserves = make([]*big.Int, len(s.Info.Reserves))
	for i, r := range s.Info.Reserves {
		cloned.Info.Reserves[i] = new(big.Int).Set(r)
	}

	return &cloned
}
