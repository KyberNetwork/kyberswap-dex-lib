package l2feecalculator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestNewL2FeeCalculator(t *testing.T) {
	type args struct {
		chainID valueobject.ChainID
	}
	tests := []struct {
		name string
		args args
		want usecase.IL2FeeCalculator
	}{
		{
			name: "it should return correct fee calculator for Optimism",
			args: args{
				chainID: valueobject.ChainIDOptimism,
			},
			want: &OptimismFeeCalculator{},
		},
		{
			name: "it should return correct fee calculator for Arbitrum",
			args: args{
				chainID: valueobject.ChainIDArbitrumOne,
			},
			want: &ArbitrumFeeCalculator{},
		},
		{
			name: "it should return nil if the chain is not L2 (Arbitrum, Optimism)",
			args: args{
				chainID: valueobject.ChainIDEthereum,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.IsType(t, tt.want, NewL2FeeCalculator(tt.args.chainID))
		})
	}
}
