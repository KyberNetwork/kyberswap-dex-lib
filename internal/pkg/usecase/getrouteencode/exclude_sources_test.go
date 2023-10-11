package getrouteencode

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestGetExcludedSources(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "it should get excluded sources correctly",
			want: []string{string(valueobject.ExchangeKyberPMM), string(valueobject.ExchangeKyberSwapLimitOrderDS)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetExcludedSources()

			assert.ElementsMatch(t, tt.want, got)
		})
	}
}
