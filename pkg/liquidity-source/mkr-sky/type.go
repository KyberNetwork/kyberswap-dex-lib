package mkr_sky

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// PoolItem represents a pool configuration item
type PoolItem struct {
	ID      string             `json:"id"`      // Pool contract address
	Type    string             `json:"type"`    // Pool type
	LpToken string             `json:"lpToken"` // LP token address
	Tokens  []entity.PoolToken `json:"tokens"`  // List of tokens in the pool
}

// Validate validates the PoolItem fields
func (p *PoolItem) Validate() error {
	if p.ID == "" {
		return errors.New("pool ID is required")
	}
	if p.Type == "" {
		return errors.New("pool type is required")
	}
	if len(p.Tokens) == 0 {
		return errors.New("pool must have at least one token")
	}
	return nil
}

// StaticExtra contains static pool configuration
type StaticExtra struct {
	Rate *big.Int `json:"rate"` // Exchange rate between MKR and SKY
}

// Validate validates the StaticExtra fields
func (s *StaticExtra) Validate() error {
	if s.Rate == nil {
		return errors.New("rate is required")
	}
	if s.Rate.Sign() <= 0 {
		return errors.New("rate must be positive")
	}
	return nil
}
