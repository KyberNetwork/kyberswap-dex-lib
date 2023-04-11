package validateroute

import (
	"errors"

	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/core/synthetix"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type SynthetixValidator struct {
}

func NewSynthetixValidator() *SynthetixValidator {
	return &SynthetixValidator{}
}

// Validate will reapply pool update and will have to modify the pool state. Do not use original pools for this
func (v *SynthetixValidator) Validate(poolByAddress map[string]poolPkg.IPool, route *valueobject.Route) error {
	err := synthetix.Validate(poolByAddress, route)

	if errors.Is(err, synthetix.ErrInvalidLastAtomicVolume) {
		return err
	}

	if errors.Is(err, synthetix.ErrSurpassedVolumeLimit) {
		logger.Error("invalid Synthetix volume for route")

		metrics.IncrInvalidSynthetixVolume()
	}

	return nil
}
