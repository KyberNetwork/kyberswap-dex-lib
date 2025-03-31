package getrouteencode

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestGetExcludedSources(t *testing.T) {
	t.Run("it should get excluded sources correctly", func(t *testing.T) {
		got := GetExcludedRFQSources()

		assert.Len(t, got, len(valueobject.RFQSourceSet))
		assert.Contains(t, got, string(valueobject.ExchangeKyberSwapLimitOrderDS))
		assert.Contains(t, got, string(valueobject.ExchangeSwaapV2))
	})
}
