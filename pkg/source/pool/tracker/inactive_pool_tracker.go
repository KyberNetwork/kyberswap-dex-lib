package pooltrack

import (
	"context"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type TrackInactivePoolsConfig struct {
	Enabled       bool                  `json:"enabled"`
	TimeThreshold durationjson.Duration `json:"timeThreshold"`
}

type InactivePoolTracker struct {
	config *TrackInactivePoolsConfig
}

func NewInactivePoolTracker(config *TrackInactivePoolsConfig) *InactivePoolTracker {
	if config == nil {
		config = &TrackInactivePoolsConfig{Enabled: false}
	}

	return &InactivePoolTracker{
		config: config,
	}
}

func (t *InactivePoolTracker) IsInactive(p *entity.Pool, currentTimestamp int64) bool {
	if t.config == nil || !t.config.Enabled {
		return false
	}

	inactiveTimeThresholdInSecond := int64(t.config.TimeThreshold.Seconds())
	if inactiveTimeThresholdInSecond <= 0 {
		return false
	}

	return currentTimestamp-p.Timestamp > inactiveTimeThresholdInSecond
}

func (d *InactivePoolTracker) GetInactivePools(_ context.Context, currentTimestamp int64,
	pools ...entity.Pool) ([]string, error) {
	if len(pools) == 0 {
		return nil, nil
	}

	inactivePools := lo.Filter(pools, func(p entity.Pool, _ int) bool {
		return d.IsInactive(&p, currentTimestamp)
	})

	return lo.Map(inactivePools, func(p entity.Pool, _ int) string { return p.Address }), nil
}
