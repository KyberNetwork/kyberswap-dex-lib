package validateroute

import (
	"errors"

	"github.com/KyberNetwork/router-service/internal/pkg/core"
	"github.com/KyberNetwork/router-service/internal/pkg/core/synthetix"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type SynthetixValidator struct {
}

func NewSynthetixValidator() *SynthetixValidator {
	return &SynthetixValidator{}
}

func (v *SynthetixValidator) Validate(route core.Route) error {
	err := synthetix.Validate(route)

	if errors.Is(err, synthetix.ErrInvalidLastAtomicVolume) {
		return err
	}

	if errors.Is(err, synthetix.ErrSurpassedVolumeLimit) {
		logger.Error("invalid Synthetix volume for route")

		metrics.IncrInvalidSynthetixVolume()
	}

	return nil
}
